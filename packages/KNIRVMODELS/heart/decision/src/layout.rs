//! Choice-set layout (Task 4.2): positions and the additive attention mask that make a
//! set-head input order-invariant and choice-isolated.
//!
//! An input is `[summary slots | question+state | choice 1 | choice 2 | ... | padding]`
//! (the model owns the summary slots; callers' tokens are everything after them).
//!
//! * **Positions.** The summary slots keep their slot ids; question and state tokens keep the
//!   ids they have in the fixed-head model; **every choice starts at the question length**, so
//!   a choice reads the same wherever it is listed and however many others there are.
//! * **Mask.** Additive: `0` where attention is allowed, `-inf` where it is not, never `1/0`.
//!   Question and state attend question and state only (bidirectional); a choice attends the
//!   question and state and **its own span only**; a summary slot attends everything;
//!   **nobody except a summary slot attends to a summary column**, which would let one choice
//!   leak into another through the question state; padding is never a key.

use candle_core::{Device, Result, Tensor};

use crate::tokenizer::{OPT_END, OPT_OPEN, PAD};

/// Where the pieces of a choice-set input sit, in token coordinates (summary slots excluded).
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct ChoiceLayout {
    /// Tokens `0..question_len` are the question and state.
    pub question_len: usize,
    /// `(start, len)` of each choice, in order, after the question. Each span covers its
    /// delimiters (`OPT_OPEN .. OPT_END`) when the prompt uses them.
    pub choice_spans: Vec<(usize, usize)>,
    /// Tokens at index `valid_len` and after are padding and are never attended to.
    /// `None` means the input has no padding.
    pub valid_len: Option<usize>,
}

impl ChoiceLayout {
    /// Checks the layout against an input of `s` tokens: the question is non-empty, the choices
    /// follow it in order without overlap or gaps, and nothing but padding follows them.
    pub fn validate(&self, s: usize) -> Result<()> {
        if self.question_len == 0 {
            candle_core::bail!("layout: the question must have at least one token");
        }
        let valid = self.valid_len.unwrap_or(s);
        if valid > s || valid < self.question_len {
            candle_core::bail!("layout: valid length {valid} is outside {}..={s}", self.question_len);
        }
        let mut at = self.question_len;
        for (i, &(start, len)) in self.choice_spans.iter().enumerate() {
            if len == 0 {
                candle_core::bail!("layout: choice {i} is empty");
            }
            if start != at {
                candle_core::bail!("layout: choice {i} starts at {start}, expected {at} (choices follow the question without gaps or overlap)");
            }
            at = start + len;
        }
        if at != valid {
            candle_core::bail!("layout: the choices end at {at} but the input is valid up to {valid}");
        }
        Ok(())
    }

    /// Reads the layout from a tokenised set prompt: the question is everything before the first
    /// `OPT_OPEN`, each choice runs from an `OPT_OPEN` to its `OPT_END`, and trailing `PAD` is
    /// padding. Anything else after the question is an error.
    pub fn from_tokens(tokens: &[u32]) -> Result<Self> {
        let valid = tokens.iter().rposition(|&t| t != PAD).map_or(0, |i| i + 1);
        let Some(first) = tokens.iter().position(|&t| t == OPT_OPEN) else {
            candle_core::bail!("layout: the prompt has no choice");
        };
        let mut spans = Vec::new();
        let mut i = first;
        while i < valid {
            if tokens[i] != OPT_OPEN {
                candle_core::bail!("layout: expected OPT_OPEN at {i}");
            }
            let Some(end) = tokens[i..valid].iter().position(|&t| t == OPT_END) else {
                candle_core::bail!("layout: choice at {i} is not closed (truncated?)");
            };
            if tokens[i + 1..i + end].contains(&OPT_OPEN) {
                candle_core::bail!("layout: choice at {i} contains another OPT_OPEN");
            }
            spans.push((i, end + 1));
            i += end + 1;
        }
        let layout = Self { question_len: first, choice_spans: spans, valid_len: (valid < tokens.len()).then_some(valid) };
        layout.validate(tokens.len())?;
        Ok(layout)
    }

    /// Position ids for the whole sequence (`slots` summary rows, then the tokens).
    pub fn pos_ids(&self, slots: usize, s: usize) -> Vec<u32> {
        let mut ids: Vec<u32> = (0..slots as u32).collect();
        let q = self.question_len;
        ids.extend((0..q).map(|t| (slots + t) as u32));
        for &(start, len) in &self.choice_spans {
            ids.extend((0..len).map(|j| (slots + q + j) as u32));
            debug_assert_eq!(ids.len(), slots + start + len);
        }
        // Padding keeps the id it would have had without a layout.
        while ids.len() < slots + s {
            ids.push(ids.len() as u32);
        }
        ids
    }

    /// The additive bias `[slots+s, slots+s]`: `0` allowed, `-inf` forbidden.
    pub fn bias(&self, slots: usize, s: usize, device: &Device) -> Result<Tensor> {
        self.validate(s)?;
        let total = slots + s;
        let q = self.question_len;
        let valid = self.valid_len.unwrap_or(s);
        // Which choice (if any) owns token t.
        let mut owner: Vec<Option<usize>> = vec![None; s];
        for (c, &(start, len)) in self.choice_spans.iter().enumerate() {
            for t in start..start + len {
                owner[t] = Some(c);
            }
        }
        let mut m = vec![f32::NEG_INFINITY; total * total];
        let mut allow = |row: usize, col: usize| m[row * total + col] = 0.0;
        for row in 0..total {
            if row < slots {
                // A summary slot sees every slot and every real token.
                (0..slots).for_each(|c| allow(row, c));
                (0..valid).for_each(|t| allow(row, slots + t));
                continue;
            }
            let t = row - slots;
            // Question, state and (for padding rows, so no row is ever empty) padding rows see
            // the question and state; choice rows see them too.
            (0..q).for_each(|k| allow(row, slots + k));
            if let Some(Some(c)) = owner.get(t) {
                let (start, len) = self.choice_spans[*c];
                (start..start + len).for_each(|k| allow(row, slots + k));
            }
        }
        Tensor::from_vec(m, (total, total), device)
    }
}

/// Accepts only an additive mask: every entry is exactly `0` or `-inf`. A 1/0 keep-mask
/// (the other common convention) would be added to the scores as `+1` and silently leak, so it
/// is refused outright.
pub fn assert_additive(bias: &Tensor) -> Result<()> {
    if bias.dtype() != candle_core::DType::F32 {
        candle_core::bail!("attention bias must be f32, got {:?}", bias.dtype());
    }
    let dims = bias.dims2()?;
    if dims.0 != dims.1 {
        candle_core::bail!("attention bias must be square, got {dims:?}");
    }
    for v in bias.flatten_all()?.to_vec1::<f32>()? {
        if v != 0.0 && v != f32::NEG_INFINITY {
            candle_core::bail!("attention bias must be additive (0 or -inf); found {v}, which looks like a 1/0 mask");
        }
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::tokenizer::{encode_set_prompt, BOS, SEP};

    fn prompt(choices: &[&str], width: usize) -> Vec<u32> {
        encode_set_prompt("why?", "host=idle", &choices.iter().map(|c| c.to_string()).collect::<Vec<_>>(), width)
    }

    #[test]
    fn layout_is_read_from_a_padded_set_prompt() {
        let t = prompt(&["wait", "restart"], 40);
        let l = ChoiceLayout::from_tokens(&t).unwrap();
        // BOS why? SEP host=idle SEP = 1+4+1+9+1 = 16 question tokens.
        assert_eq!(l.question_len, 16);
        assert_eq!(l.choice_spans, vec![(16, 6), (22, 9)]);
        assert_eq!(l.valid_len, Some(31));
        assert_eq!(t[0], BOS);
        assert_eq!(t[15], SEP);
        l.validate(t.len()).unwrap();
        // No padding: valid_len is None.
        let t = prompt(&["a", "b"], 22);
        assert_eq!(ChoiceLayout::from_tokens(&t).unwrap().valid_len, None);
    }

    #[test]
    fn malformed_prompts_and_layouts_are_refused() {
        assert!(ChoiceLayout::from_tokens(&[BOS, 1, SEP]).is_err(), "no choice");
        assert!(ChoiceLayout::from_tokens(&[BOS, 1, SEP, OPT_OPEN, 5]).is_err(), "unclosed (truncated)");
        assert!(ChoiceLayout::from_tokens(&[BOS, SEP, OPT_OPEN, OPT_OPEN, OPT_END]).is_err(), "nested open");
        assert!(ChoiceLayout::from_tokens(&[BOS, SEP, OPT_OPEN, 5, OPT_END, 7]).is_err(), "stray token after the choices");
        let ok = ChoiceLayout { question_len: 2, choice_spans: vec![(2, 2), (4, 2)], valid_len: None };
        ok.validate(6).unwrap();
        for bad in [
            ChoiceLayout { question_len: 0, ..ok.clone() },
            ChoiceLayout { choice_spans: vec![(2, 2), (5, 1)], ..ok.clone() },            // gap
            ChoiceLayout { choice_spans: vec![(2, 3), (4, 2)], ..ok.clone() },            // overlap
            ChoiceLayout { choice_spans: vec![(2, 2), (4, 0)], ..ok.clone() },            // empty
            ChoiceLayout { choice_spans: vec![(2, 2)], ..ok.clone() },                    // does not reach the end
            ChoiceLayout { valid_len: Some(9), ..ok.clone() },                            // past the input
            ChoiceLayout { valid_len: Some(1), ..ok.clone() },                            // inside the question
        ] {
            assert!(bad.validate(6).is_err(), "{bad:?}");
        }
    }

    #[test]
    fn every_choice_starts_at_the_question_length() {
        let l = ChoiceLayout { question_len: 3, choice_spans: vec![(3, 2), (5, 3)], valid_len: Some(8) };
        // 1 summary slot: slot id 0, question tokens 1..=3, both choices start at 1+3=4, padding continues.
        assert_eq!(l.pos_ids(1, 10), vec![0, 1, 2, 3, 4, 5, 4, 5, 6, 9, 10]);
        // Without a second choice the first choice's ids are unchanged.
        let one = ChoiceLayout { question_len: 3, choice_spans: vec![(3, 2)], valid_len: None };
        assert_eq!(&one.pos_ids(1, 5)[..], &[0, 1, 2, 3, 4, 5]);
    }

    #[test]
    fn the_bias_is_additive_and_isolates_choices() {
        let l = ChoiceLayout { question_len: 2, choice_spans: vec![(2, 2), (4, 1)], valid_len: Some(5) };
        let (slots, s) = (1usize, 6usize);
        let b = l.bias(slots, s, &Device::Cpu).unwrap();
        assert_additive(&b).unwrap();
        let m = b.to_vec2::<f32>().unwrap();
        let ok = |r: usize, c: usize| m[r][c] == 0.0;
        let (sum, q0, q1, a0, a1, b0, pad) = (0, 1, 2, 3, 4, 5, 6);
        // The summary slot sees itself and every real token, not padding.
        assert!([sum, q0, q1, a0, a1, b0].iter().all(|&c| ok(sum, c)) && !ok(sum, pad));
        // Question rows see the question only, never a choice or the summary.
        for r in [q0, q1] {
            assert!(ok(r, q0) && ok(r, q1));
            assert!(!ok(r, sum) && !ok(r, a0) && !ok(r, a1) && !ok(r, b0) && !ok(r, pad));
        }
        // A choice sees the question and its own span, not the other choice, not the summary.
        for r in [a0, a1] {
            assert!(ok(r, q0) && ok(r, q1) && ok(r, a0) && ok(r, a1));
            assert!(!ok(r, b0) && !ok(r, sum) && !ok(r, pad));
        }
        assert!(ok(b0, q0) && ok(b0, b0) && !ok(b0, a0) && !ok(b0, a1) && !ok(b0, sum));
        // Nobody but a summary slot attends to the summary column.
        assert!((1..=pad).all(|r| !ok(r, sum)));
        // Padding is never a key and a padding row still has somewhere to look.
        assert!((0..=pad).all(|r| !ok(r, pad)) || ok(pad, pad) == false);
        assert!(ok(pad, q0));
        // Every row has at least one finite entry (no NaN softmax).
        assert!(m.iter().all(|row| row.iter().any(|&v| v == 0.0)));
    }

    #[test]
    fn a_keep_mask_or_a_wrong_dtype_is_refused() {
        let keep = Tensor::from_vec(vec![1f32, 0., 0., 1.], (2, 2), &Device::Cpu).unwrap();
        let err = assert_additive(&keep).unwrap_err().to_string();
        assert!(err.contains("additive"), "{err}");
        let f64s = Tensor::from_vec(vec![0f64, 0., 0., 0.], (2, 2), &Device::Cpu).unwrap();
        assert!(assert_additive(&f64s).is_err());
        let rect = Tensor::from_vec(vec![0f32; 6], (2, 3), &Device::Cpu).unwrap();
        assert!(assert_additive(&rect).is_err());
        let good = Tensor::from_vec(vec![0f32, f32::NEG_INFINITY, 0., 0.], (2, 2), &Device::Cpu).unwrap();
        assert_additive(&good).unwrap();
        let nan = Tensor::from_vec(vec![0f32, f32::NAN, 0., 0.], (2, 2), &Device::Cpu).unwrap();
        assert!(assert_additive(&nan).is_err());
    }
}
