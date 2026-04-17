/**
 * Generates knirv-192x192.png and knirv-512x512.png using only Node built-ins.
 * Design: dark #000818 background + concentric #4888ff rings + center dot.
 */
import { deflateSync } from 'zlib';
import { writeFileSync, mkdirSync } from 'fs';
import { fileURLToPath } from 'url';
import { dirname, join } from 'path';

const __dir = dirname(fileURLToPath(import.meta.url));

// ── CRC32 ────────────────────────────────────────────────────────────────────
const crcTable = new Uint32Array(256);
for (let n = 0; n < 256; n++) {
  let c = n;
  for (let k = 0; k < 8; k++) c = (c & 1) ? (0xEDB88320 ^ (c >>> 1)) : (c >>> 1);
  crcTable[n] = c >>> 0;
}
function crc32(buf) {
  let c = 0xFFFFFFFF;
  for (const b of buf) c = (crcTable[(c ^ b) & 0xFF] ^ (c >>> 8)) >>> 0;
  return ((c ^ 0xFFFFFFFF) >>> 0);
}

function pngChunk(type, data) {
  const t = Buffer.from(type, 'ascii');
  const len = Buffer.allocUnsafe(4); len.writeUInt32BE(data.length);
  const crcB = Buffer.allocUnsafe(4); crcB.writeUInt32BE(crc32(Buffer.concat([t, data])));
  return Buffer.concat([len, t, data, crcB]);
}

function generatePNG(size) {
  const px = new Uint8Array(size * size * 4);
  const cx = size / 2, cy = size / 2;
  const rw = Math.max(2, size * 0.018); // ring line width

  const rings = [
    { r: size * 0.43, alpha: 1.0 },
    { r: size * 0.33, alpha: 0.6 },
    { r: size * 0.22, alpha: 0.35 },
  ];

  for (let y = 0; y < size; y++) {
    for (let x = 0; x < size; x++) {
      const i = (y * size + x) * 4;
      const dist = Math.sqrt((x - cx) ** 2 + (y - cy) ** 2);

      // Background #000818
      let R = 0, G = 8, B = 24;

      // Rings
      for (const { r, alpha } of rings) {
        const d = Math.abs(dist - r);
        if (d < rw) {
          const t = alpha * (1 - d / rw);
          R = Math.round(R * (1 - t) + 72  * t);
          G = Math.round(G * (1 - t) + 136 * t);
          B = Math.round(B * (1 - t) + 255 * t);
        }
      }

      // Center dot
      const dotR = size * 0.05;
      if (dist < dotR) {
        const t = 1 - dist / dotR;
        R = Math.round(R * (1 - t) + 72  * t);
        G = Math.round(G * (1 - t) + 136 * t);
        B = Math.round(B * (1 - t) + 255 * t);
      }

      px[i] = R; px[i + 1] = G; px[i + 2] = B; px[i + 3] = 255;
    }
  }

  // Scanlines: filter byte 0 (None) + RGBA row
  const rows = [];
  for (let y = 0; y < size; y++) {
    rows.push(Buffer.from([0]));
    rows.push(Buffer.from(px.subarray(y * size * 4, (y + 1) * size * 4)));
  }

  const ihdr = Buffer.allocUnsafe(13);
  ihdr.writeUInt32BE(size, 0); ihdr.writeUInt32BE(size, 4);
  ihdr[8] = 8; ihdr[9] = 6; ihdr[10] = 0; ihdr[11] = 0; ihdr[12] = 0;

  return Buffer.concat([
    Buffer.from([0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A]),
    pngChunk('IHDR', ihdr),
    pngChunk('IDAT', deflateSync(Buffer.concat(rows), { level: 9 })),
    pngChunk('IEND', Buffer.alloc(0)),
  ]);
}

const outDir = join(__dir, '../public/icons');
mkdirSync(outDir, { recursive: true });
writeFileSync(join(outDir, 'knirv-192x192.png'), generatePNG(192));
writeFileSync(join(outDir, 'knirv-512x512.png'), generatePNG(512));
console.log('Generated knirv-192x192.png and knirv-512x512.png');
