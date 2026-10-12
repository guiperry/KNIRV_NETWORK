//! `byte-v1` token space. Mirrors `src/sensory-shell/headTokenizer.ts` in the arena;
//! both are covered by the same vectors so browser, trainer and HEART agree.

pub const NAME: &str = "byte-v1";
pub const PAD: u32 = 256;
pub const BOS: u32 = 257;
pub const SEP: u32 = 258;
pub const EOS: u32 = 259;
pub const VOCAB: usize = 260;

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
}
