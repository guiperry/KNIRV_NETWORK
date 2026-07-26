import {
  Bracket,
  BracketSize,
  DeltaTypeI,
  DeltaTypeP,
  encodeBracket,
  decodeBracket,
  xorProjections,
  applyProjectionDelta,
  packSyntacticByte,
  unpackSyntacticByte,
  encodeSyntactic,
  decodeSyntactic,
  encodeIntentDomain,
  decodeIntentDomain,
  getPOSTag,
  setPOSTag,
  getTense,
  setTense,
  getPlurality,
  setPlurality,
} from '../bracket';

describe('Bracket wire layout', () => {
  it('BracketSize should be 80', () => {
    expect(BracketSize).toBe(80);
  });

  it('encodes bracket with correct wire layout', () => {
    const projections = new Uint8Array(32);
    for (let i = 0; i < 32; i++) projections[i] = i + 1;

    const memory = new Uint8Array(14);
    for (let i = 0; i < 14; i++) memory[i] = i + 1;

    const reserved = new Uint8Array(17);
    for (let i = 0; i < 17; i++) reserved[i] = 0x55;

    const b: Bracket = {
      id: 'test-bracket',
      projections,
      subSecondUS: 0x12345678,
      syntactic: packSyntacticByte(0x0A, 0x02, 0x01),
      depHead: 5,
      intentFlags: 0x03,
      domainSig: 0x1234,
      goldenSeed: 0xDEADBEEF,
      memory,
      lshSalt: 0xAABBCCDD,
      reserved,
      frameId: 'frame-123',
      frameUnix: 1700000000,
    };

    const encoded = encodeBracket(b);

    expect(encoded.length).toBe(80);

    expect(encoded[36]).toBe(b.syntactic);
    expect(encoded[37]).toBe(b.depHead);
    expect(encoded[38]).toBe(b.intentFlags);

    const view = new DataView(encoded.buffer, encoded.byteOffset);
    expect(view.getUint32(41, true)).toBe(b.goldenSeed);
    expect(view.getUint32(59, true)).toBe(b.lshSalt);
  });

  it('decode reverses encode', () => {
    const projections = new Uint8Array(32);
    for (let i = 0; i < 32; i++) projections[i] = (i * 17) % 256;

    const memory = new Uint8Array(14);
    for (let i = 0; i < 14; i++) memory[i] = (i * 7) % 256;

    const reserved = new Uint8Array(17);
    for (let i = 0; i < 17; i++) reserved[i] = 200 + i;

    const b: Bracket = {
      id: 'test-bracket-id',
      projections,
      subSecondUS: 987654321,
      syntactic: packSyntacticByte(5, 2, 1),
      depHead: -3,
      intentFlags: 0x07,
      domainSig: 0x5678,
      goldenSeed: 0x12345678,
      memory,
      lshSalt: 0xABCD1234,
      reserved,
      frameId: 'frame-123',
      frameUnix: 1700000000,
    };

    const encoded = encodeBracket(b);
    const decoded = decodeBracket(encoded);

    expect(decoded.subSecondUS).toBe(b.subSecondUS);
    expect(decoded.syntactic).toBe(b.syntactic);
    expect(decoded.depHead).toBe(b.depHead);
    expect(decoded.intentFlags).toBe(b.intentFlags);
    expect(decoded.domainSig).toBe(b.domainSig);
    expect(decoded.goldenSeed).toBe(b.goldenSeed);
    expect(decoded.lshSalt).toBe(b.lshSalt);
    expect(decoded.reserved).toEqual(b.reserved);

    expect(encoded.length).toBe(BracketSize);
  });

  it('LSH salt format encodes position and temporal salt', () => {
    const posIndex = 42;
    const temporalSalt = 0xABCD;
    const expected = (posIndex << 16) | temporalSalt;

    const b: Bracket = {
      id: 'test',
      projections: new Uint8Array(32),
      subSecondUS: 0,
      syntactic: 0,
      depHead: 0,
      intentFlags: 0,
      domainSig: 0,
      goldenSeed: 0,
      memory: new Uint8Array(14),
      lshSalt: expected,
      reserved: new Uint8Array(17),
      frameId: '',
      frameUnix: 0,
    };

    const encoded = encodeBracket(b);
    const view = new DataView(encoded.buffer, encoded.byteOffset);
    const gotLSH = view.getUint32(59, true);

    expect(gotLSH).toBe(expected);

    const decoded = decodeBracket(encoded);
    expect(decoded.lshSalt).toBe(expected);
  });

  it('slot alignment calculates to 80 bytes', () => {
    const slots = [
      { name: 'Projections', size: 32 },
      { name: 'SubSecondUS', size: 4 },
      { name: 'Syntactic+DepHead+IntentFlags', size: 3 },
      { name: 'DomainSig', size: 2 },
      { name: 'GoldenSeed', size: 4 },
      { name: 'Memory', size: 14 },
      { name: 'LSHSalt', size: 4 },
      { name: 'Reserved', size: 17 },
    ];

    const total = slots.reduce((sum, s) => sum + s.size, 0);
    expect(total).toBe(80);
  });

  it('zero value bracket encodes and decodes', () => {
    const b: Bracket = {
      id: '',
      projections: new Uint8Array(32),
      subSecondUS: 0,
      syntactic: 0,
      depHead: 0,
      intentFlags: 0,
      domainSig: 0,
      goldenSeed: 0,
      memory: new Uint8Array(14),
      lshSalt: 0,
      reserved: new Uint8Array(17),
      frameId: '',
      frameUnix: 0,
    };

    const encoded = encodeBracket(b);
    const decoded = decodeBracket(encoded);

    expect(decoded.subSecondUS).toBe(0);
    expect(decoded.syntactic).toBe(0);
    expect(decoded.depHead).toBe(0);
    expect(decoded.intentFlags).toBe(0);
    expect(decoded.domainSig).toBe(0);
    expect(decoded.goldenSeed).toBe(0);
    expect(decoded.lshSalt).toBe(0);
  });
});

describe('XORProjections', () => {
  it('is invertible', () => {
    const a = new Uint8Array(32);
    const b = new Uint8Array(32);
    for (let i = 0; i < 32; i++) {
      a[i] = i * 3;
      b[i] = i * 7;
    }

    const diff = xorProjections(a, b);
    const recovered = applyProjectionDelta(diff, b);

    expect(recovered).toEqual(a);
  });

  it('same input produces zero', () => {
    const a = new Uint8Array(32);
    for (let i = 0; i < 32; i++) a[i] = i;

    const diff = xorProjections(a, a);

    const zero = new Uint8Array(32);
    expect(diff).toEqual(zero);
  });
});

describe('DeltaType constants', () => {
  it('DeltaTypeI is "I"', () => {
    expect(DeltaTypeI).toBe('I');
  });

  it('DeltaTypeP is "P"', () => {
    expect(DeltaTypeP).toBe('P');
  });
});

describe('Syntactic pack/unpack', () => {
  it('packSyntacticByte then unpack recovers values', () => {
    const packed = packSyntacticByte(9, 2, 1);
    const { post, tense, plurality } = unpackSyntacticByte(packed);

    expect(post).toBe(9);
    expect(tense).toBe(2);
    expect(plurality).toBe(1);
  });

  it('encodeSyntactic roundtrip', () => {
    const sp = { syntactic: packSyntacticByte(2, 2, 1), depHead: -17 };
    const encoded = encodeSyntactic(sp);
    const decoded = decodeSyntactic(encoded);

    expect(decoded.syntactic).toBe(sp.syntactic);
  });
});

describe('IntentDomain pack/unpack', () => {
  it('encodeIntentDomain roundtrip', () => {
    const id = { intentFlags: 0x7, domainSig: 0x2000 };
    const encoded = encodeIntentDomain(id);
    const decoded = decodeIntentDomain(encoded);

    expect(decoded).toEqual(id);
  });
});

describe('Bracket getters/setters', () => {
  it('getPOSTag and setPOSTag', () => {
    const b: Bracket = {
      id: 'test',
      projections: new Uint8Array(32),
      subSecondUS: 0,
      syntactic: 0,
      depHead: 0,
      intentFlags: 0,
      domainSig: 0,
      goldenSeed: 0,
      memory: new Uint8Array(14),
      lshSalt: 0,
      reserved: new Uint8Array(17),
      frameId: '',
      frameUnix: 0,
    };

    setPOSTag(b, 9);
    expect(getPOSTag(b)).toBe(9);
  });

  it('getTense and setTense', () => {
    const b: Bracket = {
      id: 'test',
      projections: new Uint8Array(32),
      subSecondUS: 0,
      syntactic: 0,
      depHead: 0,
      intentFlags: 0,
      domainSig: 0,
      goldenSeed: 0,
      memory: new Uint8Array(14),
      lshSalt: 0,
      reserved: new Uint8Array(17),
      frameId: '',
      frameUnix: 0,
    };

    setTense(b, 2);
    expect(getTense(b)).toBe(2);
  });

  it('getPlurality and setPlurality', () => {
    const b: Bracket = {
      id: 'test',
      projections: new Uint8Array(32),
      subSecondUS: 0,
      syntactic: 0,
      depHead: 0,
      intentFlags: 0,
      domainSig: 0,
      goldenSeed: 0,
      memory: new Uint8Array(14),
      lshSalt: 0,
      reserved: new Uint8Array(17),
      frameId: '',
      frameUnix: 0,
    };

    setPlurality(b, 1);
    expect(getPlurality(b)).toBe(1);
  });
});
