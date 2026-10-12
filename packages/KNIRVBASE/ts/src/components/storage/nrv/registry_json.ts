// Registry (de)serialisation in the Go on-disk schema (pkg/nrv/frame.go).
// Go is the source of truth: snake_case keys, `brackets`/`z3`/`linguistic` per
// frame, tombstones as unix nanoseconds. TS-only per-frame data (`modalities`)
// is written under an extra key, which Go's decoder ignores.
import { FrameIndexEntry, GlobalMetrics, PQCManifest, Registry } from './codec';
import { createDefaultGlobalMetrics, BracketMeta } from './bracket';
import { createModalityMap, ModalityMap, NRV_REGISTRY_PADDING } from './spec';

const NS_PER_MS = 1_000_000;

type Json = Record<string, any>;

function metaToGo(m: BracketMeta): Json {
  return {
    id: m.id,
    type: m.type,
    anchor_id: m.anchorId ?? null,
    offset: m.offset,
    drift_score: m.driftScore,
  };
}

function metaFromGo(j: Json): BracketMeta {
  return {
    id: j.id ?? '',
    type: j.type ?? 'I',
    anchorId: j.anchor_id ?? undefined,
    offset: j.offset ?? 0,
    driftScore: j.drift_score ?? 0,
  };
}

function entryToGo(e: FrameIndexEntry): Json {
  return {
    id: e.id,
    timestamp_unix: e.timestampUnix ?? 0,
    tombstone: e.tombstone === undefined ? null : e.tombstone * NS_PER_MS,
    linguistic: e.linguistic ?? { token: '', unit: '' },
    thermo: {
      avg_temp_c: e.thermo?.avgTempC ?? 0,
      peak_volt_v: e.thermo?.peakVoltV ?? 0,
      clock_mhz: e.thermo?.clockMHz ?? 0,
    },
    z3: { status: e.verified ? 'VALID' : 'INVALID', relevance: e.ergoRank },
    brackets: { count: e.bracketCount ?? 0, offset: e.offset, length: e.length },
    bracket_index: (e.bracketIndex ?? []).map(metaToGo),
    modalities: e.modalities,
  };
}

function entryFromGo(j: Json): FrameIndexEntry {
  const length = j.brackets?.length ?? 0;
  const modalities: ModalityMap =
    j.modalities ?? createModalityMap(Math.max(0, length - 96));
  const entry: FrameIndexEntry = {
    id: j.id,
    offset: j.brackets?.offset ?? 0,
    length,
    verified: (j.z3?.status ?? 'VALID') === 'VALID',
    ergoRank: j.z3?.relevance ?? 0,
    modalities,
    timestampUnix: j.timestamp_unix ?? 0,
    linguistic: j.linguistic ?? { token: '', unit: '' },
    thermo: {
      avgTempC: j.thermo?.avg_temp_c ?? 0,
      peakVoltV: j.thermo?.peak_volt_v ?? 0,
      clockMHz: j.thermo?.clock_mhz ?? 0,
    },
    bracketCount: j.brackets?.count ?? 0,
    bracketIndex: (j.bracket_index ?? []).map(metaFromGo),
  };
  if (j.tombstone !== null && j.tombstone !== undefined) {
    entry.tombstone = Math.floor(j.tombstone / NS_PER_MS);
  }
  return entry;
}

function metricsToGo(m: GlobalMetrics): Json {
  return {
    avg_temp_c_mean: m.avgTempCMean,
    avg_temp_c_max: m.avgTempCMax,
    peak_volt_v_mean: m.peakVoltVMean,
    clock_mhz_mean: m.clockMHzMean,
    total_bracket_count: m.totalBracketCount,
    valid_frame_count: m.validFrameCount,
    invalid_frame_count: m.invalidFrameCount,
    compacted_at: m.compactedAt ?? null,
  };
}

function metricsFromGo(j: Json | undefined): GlobalMetrics {
  const d = createDefaultGlobalMetrics();
  if (!j) return d;
  const m: GlobalMetrics = {
    avgTempCMean: j.avg_temp_c_mean ?? 0,
    avgTempCMax: j.avg_temp_c_max ?? 0,
    peakVoltVMean: j.peak_volt_v_mean ?? 0,
    clockMHzMean: j.clock_mhz_mean ?? 0,
    totalBracketCount: j.total_bracket_count ?? 0,
    validFrameCount: j.valid_frame_count ?? 0,
    invalidFrameCount: j.invalid_frame_count ?? 0,
  };
  if (j.compacted_at) m.compactedAt = j.compacted_at;
  return m;
}

function manifestToGo(p: PQCManifest): Json {
  return {
    key_id: p.keyId,
    algorithm: p.algorithm,
    file_signature: p.fileSignature,
    frame_signatures: p.frameSignatures,
  };
}

function manifestFromGo(j: Json | undefined): PQCManifest {
  return {
    keyId: j?.key_id ?? '',
    algorithm: j?.algorithm ?? '',
    fileSignature: j?.file_signature ?? '',
    frameSignatures: j?.frame_signatures ?? {},
  };
}

export function registryToGoJSON(r: Registry): string {
  return JSON.stringify(
    {
      version: r.version,
      dataset_id: r.datasetId,
      dataset_version: r.datasetVersion,
      chunk0_length: r.chunk0Length,
      frame_count: r.frameCount,
      tombstone_count: r.tombstoneCount,
      global_metrics: metricsToGo(r.globalMetrics),
      frames: r.frames.map(entryToGo),
      pqc_manifest: manifestToGo(r.pqcManifest),
    },
    null,
    2
  );
}

export function registryFromGoJSON(text: string): Registry {
  const j = JSON.parse(text) as Json;
  return {
    version: j.version ?? 1,
    datasetId: j.dataset_id ?? '',
    datasetVersion: j.dataset_version ?? {},
    chunk0Length: j.chunk0_length ?? NRV_REGISTRY_PADDING,
    frameCount: j.frame_count ?? 0,
    tombstoneCount: j.tombstone_count ?? 0,
    globalMetrics: metricsFromGo(j.global_metrics),
    frames: (j.frames ?? []).map(entryFromGo),
    pqcManifest: manifestFromGo(j.pqc_manifest),
  };
}

/**
 * Serialise the registry into the fixed 5 MiB region. Uses the byte length
 * (not UTF-16 length) and refuses to overflow, matching Go's saveRegistry.
 */
export function registryToPaddedBuffer(r: Registry): Uint8Array {
  const json = new TextEncoder().encode(registryToGoJSON(r));
  if (json.length > NRV_REGISTRY_PADDING) {
    throw new Error('nrv: registry exceeds padding');
  }
  const out = new Uint8Array(NRV_REGISTRY_PADDING);
  out.set(json);
  return out;
}

/** Parse the registry region (trailing NUL/whitespace padding allowed). */
export function registryFromPaddedBuffer(buf: Uint8Array): Registry {
  let end = 0;
  for (let i = buf.length - 1; i >= 0; i--) {
    const b = buf[i];
    if (b !== 0 && b !== 0x20 && b !== 0x0a && b !== 0x0d && b !== 0x09) {
      end = i + 1;
      break;
    }
  }
  if (end === 0) {
    throw new Error('nrv: empty registry');
  }
  return registryFromGoJSON(new TextDecoder().decode(buf.subarray(0, end)));
}
