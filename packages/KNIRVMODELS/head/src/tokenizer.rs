//! `byte-v1` token space. Mirrors `src/sensory-shell/headTokenizer.ts` in the arena;
//! both are covered by the same vectors so browser, trainer and HEART agree.

pub const NAME: &str = "byte-v1";
pub const PAD: u32 = 256;
pub const BOS: u32 = 257;
pub const SEP: u32 = 258;
pub const EOS: u32 = 259;
/// Delimit one candidate option in a set-head prompt. These are appended to
/// byte-v1 so every pre-v2 token id keeps its original meaning.
pub const OPT_OPEN: u32 = 260;
pub const OPT_END: u32 = 261;
pub const VOCAB: usize = 262;

/// `[BOS, bytes...]`, truncated to `max_len`.
pub fn encode_text(text: &str, max_len: usize) -> Vec<u32> {
    std::iter::once(BOS).chain(text.bytes().map(u32::from)).take(max_len).collect()
}

/// Right-pads with PAD (or truncates) to exactly `width`: the model always sees a fixed layout.
pub fn pad_to(mut tokens: Vec<u32>, width: usize) -> Vec<u32> {
    tokens.truncate(width);
    tokens.resize(width, PAD);
    tokens
}

/// Plan target for a fixed-width layout: plan bytes, then EOS, then PAD.
pub fn encode_plan(plan: &str, width: usize) -> Vec<u32> {
    let mut t: Vec<u32> = plan.bytes().map(u32::from).collect();
    t.truncate(width.saturating_sub(1));
    t.push(EOS);
    pad_to(t, width)
}

/// Canonical input for a choice-set head. The model owns the summary slot, so
/// callers supply only the tokens after it:
/// `BOS question SEP state SEP OPT_OPEN choice OPT_END ...`.
///
/// This deliberately has no `intent:` or `risk:` fields: emitting a label in
/// its own set-head input would make the task a leakage shortcut. As with the
/// v1 encoder, the fixed-width model input is right-padded and excess bytes are
/// truncated; the trainer records and limits truncation separately.
pub fn encode_set_prompt(question: &str, state: &str, choices: &[String], width: usize) -> Vec<u32> {
    let mut tokens = vec![BOS];
    tokens.extend(question.bytes().map(u32::from));
    tokens.push(SEP);
    tokens.extend(state.bytes().map(u32::from));
    tokens.push(SEP);
    for choice in choices {
        tokens.push(OPT_OPEN);
        tokens.extend(choice.bytes().map(u32::from));
        tokens.push(OPT_END);
    }
    pad_to(tokens, width)
}

/// Bytes back to text up to the first EOS; other specials are dropped.
pub fn decode(tokens: &[u32]) -> String {
    let bytes: Vec<u8> = tokens.iter().take_while(|&&t| t != EOS).filter(|&&t| t < 256).map(|&t| t as u8).collect();
    String::from_utf8_lossy(&bytes).into_owned()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn matches_the_typescript_vectors() {
        assert_eq!(encode_text("abcdef", 3), vec![BOS, 97, 98]);
        assert_eq!(decode(&encode_text("héllo", 64)), "héllo");
        assert_eq!(pad_to(vec![BOS, 1], 4), vec![BOS, 1, PAD, PAD]);
        assert_eq!(encode_plan("ok", 5), vec![111, 107, EOS, PAD, PAD]);
        assert_eq!(decode(&encode_plan("ok", 5)), "ok");
        assert_eq!(encode_plan("toolong", 4), vec![116, 111, 111, EOS]);
    }

    #[test]
    fn set_prompt_keeps_v1_ids_and_delimits_choices() {
        assert_eq!(
            encode_set_prompt("why?", "host=idle", &["wait".into(), "restart".into()], 32),
            vec![BOS, 119, 104, 121, 63, SEP, 104, 111, 115, 116, 61, 105, 100, 108, 101, SEP, OPT_OPEN, 119, 97, 105, 116, OPT_END, OPT_OPEN, 114, 101, 115, 116, 97, 114, 116, OPT_END, PAD],
        );
    }
}
