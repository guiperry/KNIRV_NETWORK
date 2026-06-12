import { create } from "zustand";

// ── Synthesized sound effects ──────────────────────────────────────────────
// All gameplay SFX are generated with the WebAudio API so no asset files are
// required. Each effect is a short oscillator/noise burst with an envelope.

export type SfxName =
  | "click"        // generic UI button press
  | "open"         // DVE workspace modal opens
  | "close"        // modal closes
  | "select"       // error node / agent selected
  | "place"        // validation anchor placed on a spike
  | "set"          // anchor configured + set
  | "sculpt"       // ring committed — deep sink rumble
  | "deploy"       // agent deployed — rising whoosh
  | "straighten"   // straightening sequence begins
  | "resolve"      // error node resolved
  | "win"          // epoch won / skill slot hijacked
  | "error"        // invalid action / not enough NRN
  | "sabotage"     // sabotage applied — zap
  | "epoch";       // epoch run started

interface AudioState {
  backgroundMusic: HTMLAudioElement | null;
  hitSound: HTMLAudioElement | null;
  successSound: HTMLAudioElement | null;
  isMuted: boolean;

  // Setter functions
  setBackgroundMusic: (music: HTMLAudioElement) => void;
  setHitSound: (sound: HTMLAudioElement) => void;
  setSuccessSound: (sound: HTMLAudioElement) => void;

  // Control functions
  toggleMute: () => void;
  playHit: () => void;
  playSuccess: () => void;
  playSfx: (name: SfxName) => void;
}

let audioCtx: AudioContext | null = null;

function getCtx(): AudioContext | null {
  if (typeof window === "undefined") return null;
  const Ctor = window.AudioContext ??
    (window as unknown as { webkitAudioContext?: typeof AudioContext }).webkitAudioContext;
  if (!Ctor) return null;
  if (!audioCtx) audioCtx = new Ctor();
  if (audioCtx.state === "suspended") {
    audioCtx.resume().catch(() => undefined);
  }
  return audioCtx;
}

interface ToneSpec {
  type: OscillatorType;
  from: number;        // start frequency (Hz)
  to?: number;         // end frequency (defaults to `from`)
  duration: number;    // seconds
  volume?: number;     // peak gain 0..1
  delay?: number;      // seconds before this tone starts
}

function playTones(tones: ToneSpec[]) {
  const ctx = getCtx();
  if (!ctx) return;
  const now = ctx.currentTime;
  for (const t of tones) {
    const osc = ctx.createOscillator();
    const gain = ctx.createGain();
    const start = now + (t.delay ?? 0);
    const end = start + t.duration;
    osc.type = t.type;
    osc.frequency.setValueAtTime(t.from, start);
    osc.frequency.exponentialRampToValueAtTime(Math.max(t.to ?? t.from, 1), end);
    const peak = t.volume ?? 0.2;
    gain.gain.setValueAtTime(0.0001, start);
    gain.gain.exponentialRampToValueAtTime(peak, start + 0.015);
    gain.gain.exponentialRampToValueAtTime(0.0001, end);
    osc.connect(gain).connect(ctx.destination);
    osc.start(start);
    osc.stop(end + 0.05);
  }
}

function playNoise(duration: number, volume: number, fromFreq: number, toFreq: number, delay = 0) {
  const ctx = getCtx();
  if (!ctx) return;
  const frames = Math.max(1, Math.floor(ctx.sampleRate * duration));
  const buffer = ctx.createBuffer(1, frames, ctx.sampleRate);
  const data = buffer.getChannelData(0);
  for (let i = 0; i < frames; i++) data[i] = Math.random() * 2 - 1;

  const src = ctx.createBufferSource();
  src.buffer = buffer;
  const filter = ctx.createBiquadFilter();
  filter.type = "bandpass";
  const gain = ctx.createGain();
  const start = ctx.currentTime + delay;
  const end = start + duration;
  filter.frequency.setValueAtTime(fromFreq, start);
  filter.frequency.exponentialRampToValueAtTime(Math.max(toFreq, 1), end);
  filter.Q.value = 1.2;
  gain.gain.setValueAtTime(0.0001, start);
  gain.gain.exponentialRampToValueAtTime(volume, start + 0.02);
  gain.gain.exponentialRampToValueAtTime(0.0001, end);
  src.connect(filter).connect(gain).connect(ctx.destination);
  src.start(start);
  src.stop(end + 0.05);
}

const SFX: Record<SfxName, () => void> = {
  click: () => playTones([{ type: "square", from: 880, to: 660, duration: 0.06, volume: 0.08 }]),
  open: () => playTones([
    { type: "sine", from: 330, to: 660, duration: 0.16, volume: 0.14 },
    { type: "sine", from: 660, to: 990, duration: 0.14, volume: 0.1, delay: 0.1 },
  ]),
  close: () => playTones([
    { type: "sine", from: 660, to: 330, duration: 0.14, volume: 0.12 },
  ]),
  select: () => playTones([{ type: "triangle", from: 520, to: 700, duration: 0.09, volume: 0.12 }]),
  place: () => playTones([
    { type: "triangle", from: 980, to: 1240, duration: 0.08, volume: 0.14 },
    { type: "sine", from: 1480, duration: 0.06, volume: 0.07, delay: 0.06 },
  ]),
  set: () => playTones([
    { type: "sine", from: 740, duration: 0.08, volume: 0.13 },
    { type: "sine", from: 1110, duration: 0.12, volume: 0.13, delay: 0.07 },
  ]),
  sculpt: () => {
    // Deep descending rumble as the ring sinks beneath the grid
    playTones([
      { type: "sawtooth", from: 180, to: 40, duration: 1.2, volume: 0.18 },
      { type: "sine", from: 90, to: 30, duration: 1.4, volume: 0.22 },
    ]);
    playNoise(1.1, 0.08, 600, 80);
  },
  deploy: () => {
    playNoise(0.5, 0.12, 300, 2400);
    playTones([{ type: "sine", from: 440, to: 880, duration: 0.4, volume: 0.1 }]);
  },
  straighten: () => playTones([
    { type: "triangle", from: 220, to: 440, duration: 0.3, volume: 0.12 },
    { type: "triangle", from: 330, to: 660, duration: 0.3, volume: 0.1, delay: 0.15 },
  ]),
  resolve: () => playTones([
    { type: "sine", from: 523, duration: 0.12, volume: 0.16 },
    { type: "sine", from: 659, duration: 0.12, volume: 0.16, delay: 0.1 },
    { type: "sine", from: 784, duration: 0.2, volume: 0.16, delay: 0.2 },
    { type: "sine", from: 1046, duration: 0.3, volume: 0.14, delay: 0.32 },
  ]),
  win: () => playTones([
    { type: "square", from: 523, duration: 0.1, volume: 0.08 },
    { type: "square", from: 784, duration: 0.1, volume: 0.08, delay: 0.1 },
    { type: "square", from: 1046, duration: 0.25, volume: 0.1, delay: 0.2 },
  ]),
  error: () => playTones([
    { type: "sawtooth", from: 220, to: 110, duration: 0.18, volume: 0.12 },
  ]),
  sabotage: () => {
    playNoise(0.25, 0.14, 3000, 200);
    playTones([{ type: "sawtooth", from: 1200, to: 80, duration: 0.25, volume: 0.1 }]);
  },
  epoch: () => playTones([
    { type: "triangle", from: 392, duration: 0.1, volume: 0.12 },
    { type: "triangle", from: 587, duration: 0.16, volume: 0.12, delay: 0.09 },
  ]),
};

export const useAudio = create<AudioState>((set, get) => ({
  backgroundMusic: null,
  hitSound: null,
  successSound: null,
  isMuted: true, // Start muted by default

  setBackgroundMusic: (music) => set({ backgroundMusic: music }),
  setHitSound: (sound) => set({ hitSound: sound }),
  setSuccessSound: (sound) => set({ successSound: sound }),

  toggleMute: () => {
    const { isMuted, backgroundMusic } = get();
    const newMutedState = !isMuted;
    set({ isMuted: newMutedState });

    if (backgroundMusic) {
      if (newMutedState) {
        backgroundMusic.pause();
      } else {
        backgroundMusic.play().catch(() => undefined);
      }
    }
  },

  playHit: () => {
    const { hitSound, isMuted } = get();
    if (isMuted) return;
    if (hitSound) {
      // Clone sound to allow overlapping playback
      const soundClone = hitSound.cloneNode() as HTMLAudioElement;
      soundClone.volume = 0.3;
      soundClone.play().catch(() => undefined);
    } else {
      SFX.click();
    }
  },

  playSuccess: () => {
    const { successSound, isMuted } = get();
    if (isMuted) return;
    if (successSound) {
      successSound.currentTime = 0;
      successSound.play().catch(() => undefined);
    } else {
      SFX.resolve();
    }
  },

  playSfx: (name) => {
    if (get().isMuted) return;
    try {
      SFX[name]();
    } catch {
      // Audio not available (e.g. test environment) — ignore
    }
  },
}));
