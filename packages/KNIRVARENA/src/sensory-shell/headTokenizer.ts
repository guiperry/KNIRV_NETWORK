/**
 * `byte-v1` token space shared by the browser, KNIRVMODELS's HEAD trainer and
 * HEART: ids 0-255 are raw bytes, then four specials. A model whose config
 * names another tokenizer cannot run in the browser and falls back to the server.
 */
export const BYTE_V1 = 'byte-v1';
export const PAD = 256;
export const BOS = 257;
export const SEP = 258;
export const EOS = 259;
export const BYTE_V1_VOCAB = 260;

export function encodeText(text: string, maxLen: number): number[] {
  const bytes = Array.from(new TextEncoder().encode(text));
  return [BOS, ...bytes].slice(0, maxLen);
}

/** Numeric sensor data in [0,1] (voice/visual) quantised to byte tokens. */
export function encodeSamples(samples: ArrayLike<number>, maxLen: number): number[] {
  const out: number[] = [BOS];
  for (let i = 0; i < samples.length && out.length < maxLen; i++) {
    const v = Number.isFinite(samples[i]) ? samples[i] : 0;
    out.push(Math.max(0, Math.min(255, Math.round(v * 255))));
  }
  return out;
}

/** Bytes back to text; specials are dropped, invalid UTF-8 is replaced. */
export function decodeTokens(tokens: ArrayLike<number>): string {
  const bytes: number[] = [];
  for (let i = 0; i < tokens.length; i++) if (tokens[i] >= 0 && tokens[i] < 256) bytes.push(tokens[i]);
  return new TextDecoder().decode(new Uint8Array(bytes));
}

/** Right-pads (or truncates) to the model's fixed input width; training used the same layout. */
export function padTo(tokens: number[], width: number): number[] {
  const out = tokens.slice(0, width);
  while (out.length < width) out.push(PAD);
  return out;
}
