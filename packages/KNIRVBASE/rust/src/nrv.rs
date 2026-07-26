//! NRV binary file format reader/writer.
//!
//! The .nrv format (source of truth: Go implementation):
//!   - 12-byte header:  [magic u32 LE | version u32 LE | total_length u32 LE]
//!   - 5 MiB registry:  JSON padded to NRV_REGISTRY_PADDING bytes with NUL bytes
//!   - Frame data:      consecutive binary frames appended after the registry
//!
//! Each frame binary layout:
//!   [0..48]   vector  – 12 × f32 LE
//!   [48..80]  seed    – 32 bytes
//!   [80..96]  thermo  – temp_celsius, voltage_v, freq_mhz, fan_rpm as f32 LE
//!   [96..N]   proof   – variable bytes; total frame size aligned to 8 bytes

use std::collections::HashMap;
use std::fs::{self, File, OpenOptions};
use std::io::{Read, Seek, SeekFrom, Write};
use std::path::Path;
use std::sync::Mutex;

use crate::wal::{WAL, WALEntry};
use crate::{Frame, FrameEntry, GlobalMetrics, ModalityIndex, PQCManifest, Registry, ThermoData};
use crate::{NRV_MAGIC, NRV_VERSION};

pub const NRV_HEADER_SIZE: usize = 12;
pub const NRV_REGISTRY_PADDING: usize = 5 * 1024 * 1024; // 5 MiB

// ── helpers ─────────────────────────────────────────────────────────────────

fn align8(n: usize) -> usize {
    (n + 7) & !7
}

fn encode_header(total_length: u32) -> [u8; 12] {
    let mut buf = [0u8; 12];
    buf[0..4].copy_from_slice(&NRV_MAGIC.to_le_bytes());
    buf[4..8].copy_from_slice(&NRV_VERSION.to_le_bytes());
    buf[8..12].copy_from_slice(&total_length.to_le_bytes());
    buf
}

fn check_header(data: &[u8]) -> Result<(), String> {
    if data.len() < NRV_HEADER_SIZE {
        return Err("file too small for NRV header".into());
    }
    let magic = u32::from_le_bytes(data[0..4].try_into().unwrap());
    if magic != NRV_MAGIC {
        return Err(format!(
            "invalid NRV magic: got 0x{:X}, want 0x{:X}",
            magic, NRV_MAGIC
        ));
    }
    Ok(())
}

/// Encode a Frame into binary and return the byte buffer + modality map.
pub fn encode_frame(frame: &Frame) -> (Vec<u8>, HashMap<String, ModalityIndex>) {
    let proof_len = frame.proof.len();
    let proof_aligned = align8(proof_len);
    let total = 48 + 32 + 16 + proof_aligned;
    let mut buf = vec![0u8; total];

    for (i, &v) in frame.vector.iter().enumerate() {
        buf[i * 4..i * 4 + 4].copy_from_slice(&v.to_le_bytes());
    }
    buf[48..80].copy_from_slice(&frame.seed);
    buf[80..84].copy_from_slice(&frame.thermo.temp_celsius.to_le_bytes());
    buf[84..88].copy_from_slice(&frame.thermo.voltage_v.to_le_bytes());
    buf[88..92].copy_from_slice(&frame.thermo.freq_mhz.to_le_bytes());
    buf[92..96].copy_from_slice(&frame.thermo.fan_rpm.to_le_bytes());
    if proof_len > 0 {
        buf[96..96 + proof_len].copy_from_slice(&frame.proof);
    }

    let mut modalities = HashMap::new();
    modalities.insert("vector".into(), ModalityIndex { offset: 0, length: 48 });
    modalities.insert("seed".into(), ModalityIndex { offset: 48, length: 32 });
    modalities.insert("thermo".into(), ModalityIndex { offset: 80, length: 16 });
    modalities.insert("proof".into(), ModalityIndex { offset: 96, length: proof_len as i32 });

    (buf, modalities)
}

/// Decode a Frame from a raw byte slice using its registry FrameEntry for offsets.
pub fn decode_frame(data: &[u8], entry: &FrameEntry) -> Result<Frame, String> {
    if data.len() < 96 {
        return Err(format!("frame too small: {} bytes", data.len()));
    }

    let mut vector = [0f32; 12];
    for i in 0..12 {
        vector[i] = f32::from_le_bytes(data[i * 4..i * 4 + 4].try_into().unwrap());
    }

    let mut seed = [0u8; 32];
    seed.copy_from_slice(&data[48..80]);

    let thermo = ThermoData {
        temp_celsius: f32::from_le_bytes(data[80..84].try_into().unwrap()),
        voltage_v: f32::from_le_bytes(data[84..88].try_into().unwrap()),
        freq_mhz: f32::from_le_bytes(data[88..92].try_into().unwrap()),
        fan_rpm: f32::from_le_bytes(data[92..96].try_into().unwrap()),
    };

    let proof = if let Some(m) = entry.modalities.get("proof") {
        let start = m.offset as usize;
        let end = start + m.length as usize;
        data.get(start..end).unwrap_or(&[]).to_vec()
    } else {
        vec![]
    };

    Ok(Frame { id: entry.id.clone(), vector, seed, thermo, proof })
}

fn parse_registry(file_data: &[u8]) -> Result<Registry, String> {
    if file_data.len() < NRV_HEADER_SIZE + NRV_REGISTRY_PADDING {
        return Err("file too small to contain registry".into());
    }
    let region = &file_data[NRV_HEADER_SIZE..NRV_HEADER_SIZE + NRV_REGISTRY_PADDING];
    let end = json_end(region);
    if end == 0 {
        return Err("empty NRV registry".into());
    }
    let s = std::str::from_utf8(&region[..end]).map_err(|e| e.to_string())?;
    serde_json::from_str(s).map_err(|e| format!("registry parse error: {}", e))
}

fn json_end(buf: &[u8]) -> usize {
    for i in (0..buf.len()).rev() {
        let b = buf[i];
        if b != 0 && b != b' ' && b != b'\n' && b != b'\r' && b != b'\t' {
            return i + 1;
        }
    }
    0
}

fn write_registry(file: &mut File, registry: &Registry, total_len: u64) -> Result<(), String> {
    let json = serde_json::to_string_pretty(registry)
        .map_err(|e| format!("registry serialise: {}", e))?;
    let bytes = json.as_bytes();
    if bytes.len() > NRV_REGISTRY_PADDING {
        return Err("registry exceeds 5 MiB padding".into());
    }
    let padding = vec![0u8; NRV_REGISTRY_PADDING - bytes.len()];

    file.seek(SeekFrom::Start(NRV_HEADER_SIZE as u64))
        .map_err(|e| e.to_string())?;
    file.write_all(bytes).map_err(|e| e.to_string())?;
    file.write_all(&padding).map_err(|e| e.to_string())?;

    let header = encode_header(total_len as u32);
    file.seek(SeekFrom::Start(0)).map_err(|e| e.to_string())?;
    file.write_all(&header).map_err(|e| e.to_string())?;
    file.flush().map_err(|e| e.to_string())?;
    Ok(())
}

// ── NRVWriter ────────────────────────────────────────────────────────────────

/// Writes frames to a .nrv file with WAL-protected append semantics.
///
/// Creating an NRVWriter also creates any missing parent directories, matching
/// the production usage pattern where the app data directory may not exist yet.
pub struct NRVWriter {
    path: String,
    file: Mutex<File>,
    registry: Mutex<Registry>,
    position: Mutex<u64>,
    wal: WAL,
}

impl NRVWriter {
    /// Open or create the .nrv file at `path`.
    ///
    /// Parent directories are created automatically, mirroring what
    /// `NRVStorage.New(baseDir, keyPair)` does in the Go implementation.
    pub fn create(path: &str) -> Result<Self, Box<dyn std::error::Error + Send + Sync>> {
        if let Some(parent) = Path::new(path).parent() {
            fs::create_dir_all(parent)?;
        }

        let wal_path = format!("{}.wal", path);
        let wal = WAL::new(wal_path);

        let is_new = !Path::new(path).exists();
        let mut file = OpenOptions::new().read(true).write(true).create(true).open(path)?;

        let (position, registry) = if is_new || file.metadata()?.len() == 0 {
            let id = uuid::Uuid::new_v4().to_string();
            let registry = Registry {
                version: 1,
                dataset_id: id,
                pqc_manifest: PQCManifest {
                    algorithm: "Dilithium-3".into(),
                    frame_signatures: HashMap::new(),
                    ..PQCManifest::default()
                },
                global_metrics: GlobalMetrics::default(),
                frames: Vec::new(),
                ..Registry::default()
            };

            // Write header + registry padding
            let header = encode_header((NRV_HEADER_SIZE + NRV_REGISTRY_PADDING) as u32);
            file.seek(SeekFrom::Start(0))?;
            file.write_all(&header)?;
            let json = serde_json::to_string_pretty(&registry)?;
            let jbytes = json.as_bytes();
            file.write_all(jbytes)?;
            file.write_all(&vec![0u8; NRV_REGISTRY_PADDING - jbytes.len()])?;
            file.flush()?;

            ((NRV_HEADER_SIZE + NRV_REGISTRY_PADDING) as u64, registry)
        } else {
            // Existing file: WAL recovery
            let recover_len = wal.recover().map_err(|e| e.clone())?;
            let pos = if recover_len > 0 {
                file.set_len(recover_len as u64)?;
                recover_len as u64
            } else {
                file.metadata()?.len()
            };

            let mut buf = vec![0u8; NRV_HEADER_SIZE + NRV_REGISTRY_PADDING];
            file.seek(SeekFrom::Start(0))?;
            file.read_exact(&mut buf)?;
            check_header(&buf)?;
            let registry = parse_registry(&buf)?;
            (pos, registry)
        };

        Ok(Self {
            path: path.to_string(),
            file: Mutex::new(file),
            registry: Mutex::new(registry),
            position: Mutex::new(position),
            wal,
        })
    }

    /// Append a frame to the file.
    pub fn append_frame(
        &self,
        frame: &Frame,
        verified: bool,
        ergo_rank: f64,
    ) -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
        let mut file = self.file.lock().unwrap();
        let mut registry = self.registry.lock().unwrap();
        let mut position = self.position.lock().unwrap();

        self.wal.begin(WALEntry {
            frame_id: frame.id.clone(),
            last_good_length: *position as i64,
            committed: false,
        })?;

        let (frame_bytes, modalities) = encode_frame(frame);

        file.seek(SeekFrom::Start(*position))?;
        file.write_all(&frame_bytes)?;
        let new_pos = *position + frame_bytes.len() as u64;

        let entry = FrameEntry {
            id: frame.id.clone(),
            offset: *position as i64,
            length: frame_bytes.len() as i32,
            tombstone: None,
            verified,
            ergo_rank,
            modalities,
        };

        registry.frames.push(entry);
        registry.frame_count += 1;
        if verified {
            registry.global_metrics.verified_frame_count += 1;
        }
        registry.global_metrics.ergo_rank_sum += ergo_rank;

        write_registry(&mut file, &registry, new_pos)?;
        *position = new_pos;

        self.wal.commit(&frame.id)?;
        Ok(())
    }

    /// Soft-delete a frame by timestamping its tombstone in the registry.
    pub fn set_tombstone(
        &self,
        id: &str,
    ) -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
        let mut file = self.file.lock().unwrap();
        let mut registry = self.registry.lock().unwrap();
        let position = self.position.lock().unwrap();

        let now = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .map(|d| d.as_nanos() as i64)
            .unwrap_or(0);

        let mut found = false;
        for entry in &mut registry.frames {
            if entry.id == id {
                entry.tombstone = Some(now);
                registry.tombstone_count += 1;
                found = true;
                break;
            }
        }
        if !found {
            return Err(format!("nrv: frame not found: {}", id).into());
        }

        write_registry(&mut file, &registry, *position)?;
        Ok(())
    }

    pub fn get_registry(&self) -> Registry {
        self.registry.lock().unwrap().clone()
    }

    pub fn path(&self) -> &str {
        &self.path
    }
}

impl Drop for NRVWriter {
    fn drop(&mut self) {
        // Best-effort WAL cleanup on drop, matching NRVWriter.Close() in Go.
        let _ = self.wal.truncate();
    }
}

// ── NRVReader ────────────────────────────────────────────────────────────────

/// Reads frames from a fully-loaded .nrv file.
pub struct NRVReader {
    data: Vec<u8>,
    registry: Registry,
}

impl NRVReader {
    /// Load the entire .nrv file into memory and parse its registry.
    pub fn open(path: &str) -> Result<Self, Box<dyn std::error::Error + Send + Sync>> {
        let data = fs::read(path)?;
        check_header(&data)?;
        let registry = parse_registry(&data)?;
        Ok(Self { data, registry })
    }

    /// Look up a live (non-tombstoned) frame by ID.
    pub fn get_frame(&self, id: &str) -> Option<Frame> {
        let entry = self.registry.frames.iter().find(|e| e.id == id)?;
        if entry.tombstone.is_some() {
            return None;
        }
        let start = entry.offset as usize;
        let end = start + entry.length as usize;
        decode_frame(self.data.get(start..end)?, entry).ok()
    }

    /// Return raw bytes for one modality of a frame.
    pub fn get_modality(&self, frame_id: &str, modality: &str) -> Option<Vec<u8>> {
        let entry = self.registry.frames.iter().find(|e| e.id == frame_id)?;
        if entry.tombstone.is_some() {
            return None;
        }
        let m = entry.modalities.get(modality)?;
        let start = entry.offset as usize + m.offset as usize;
        let end = start + m.length as usize;
        Some(self.data.get(start..end)?.to_vec())
    }

    /// Iterate over all live frames.
    pub fn stream_frames(&self) -> impl Iterator<Item = Frame> + '_ {
        self.registry.frames.iter().filter_map(move |entry| {
            if entry.tombstone.is_some() {
                return None;
            }
            let start = entry.offset as usize;
            let end = start + entry.length as usize;
            decode_frame(self.data.get(start..end)?, entry).ok()
        })
    }

    pub fn registry(&self) -> &Registry {
        &self.registry
    }
}

// ── tests ────────────────────────────────────────────────────────────────────

#[cfg(test)]
mod tests {
    use super::*;
    use tempfile::TempDir;

    /// Build a frame with known, deterministic values for round-trip checks.
    fn test_frame(id: &str) -> Frame {
        let mut seed = [0u8; 32];
        for (i, b) in seed.iter_mut().enumerate() {
            *b = (i + 1) as u8;
        }
        Frame {
            id: id.to_string(),
            vector: [1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0, 11.0, 12.0],
            seed,
            thermo: ThermoData {
                temp_celsius: 50.5,
                voltage_v: 12.0,
                freq_mhz: 500.0,
                fan_rpm: 3000.0,
            },
            proof: b"test proof data".to_vec(),
        }
    }

    /// Returns `<tmp>/knirvbase/datasets/` — the production-style app data path.
    fn app_data_dir(tmp: &TempDir) -> std::path::PathBuf {
        tmp.path().join("knirvbase").join("datasets")
    }

    // ── Writer ───────────────────────────────────────────────────────────────

    #[test]
    fn test_writer_creates_nrv_file() {
        let tmp = TempDir::new().unwrap();
        let path = tmp.path().join("new.nrv");

        let writer = NRVWriter::create(path.to_str().unwrap()).unwrap();

        assert!(path.exists(), ".nrv file must be created on disk");

        let size = fs::metadata(&path).unwrap().len() as usize;
        assert_eq!(
            size,
            NRV_HEADER_SIZE + NRV_REGISTRY_PADDING,
            "empty .nrv must be exactly header + registry padding"
        );

        let reg = writer.get_registry();
        assert_eq!(reg.version, 1);
        assert!(!reg.dataset_id.is_empty());
        assert_eq!(reg.frame_count, 0);
    }

    #[test]
    fn test_binary_header_spec() {
        let tmp = TempDir::new().unwrap();
        let path = tmp.path().join("spec.nrv");
        NRVWriter::create(path.to_str().unwrap()).unwrap();

        let bytes = fs::read(&path).unwrap();

        // Magic 0x4E525621 in little-endian = [0x21, 0x56, 0x52, 0x4E]
        assert_eq!(&bytes[0..4], &[0x21, 0x56, 0x52, 0x4E], "magic bytes must match spec");
        assert_eq!(
            u32::from_le_bytes(bytes[4..8].try_into().unwrap()),
            1,
            "version must be 1"
        );
        let total = u32::from_le_bytes(bytes[8..12].try_into().unwrap()) as usize;
        assert_eq!(total, NRV_HEADER_SIZE + NRV_REGISTRY_PADDING);
    }

    #[test]
    fn test_writer_append_frame_updates_registry() {
        let tmp = TempDir::new().unwrap();
        let path = tmp.path().join("reg.nrv");
        let writer = NRVWriter::create(path.to_str().unwrap()).unwrap();

        writer.append_frame(&test_frame("f1"), true, 0.9).unwrap();
        writer.append_frame(&test_frame("f2"), false, 0.4).unwrap();

        let reg = writer.get_registry();
        assert_eq!(reg.frame_count, 2);
        assert_eq!(reg.global_metrics.verified_frame_count, 1);
        assert!((reg.global_metrics.ergo_rank_sum - 1.3).abs() < 1e-9);
    }

    #[test]
    fn test_writer_reopen_persists_frames() {
        let tmp = TempDir::new().unwrap();
        let path = tmp.path().join("persist.nrv");

        {
            let w = NRVWriter::create(path.to_str().unwrap()).unwrap();
            w.append_frame(&test_frame("frame-a"), true, 0.8).unwrap();
        } // Drop triggers WAL cleanup

        {
            let w = NRVWriter::create(path.to_str().unwrap()).unwrap();
            assert_eq!(w.get_registry().frame_count, 1, "frame from first session must survive reopen");
            w.append_frame(&test_frame("frame-b"), false, 0.5).unwrap();
        }

        let reader = NRVReader::open(path.to_str().unwrap()).unwrap();
        assert_eq!(reader.registry().frame_count, 2);
        assert!(reader.get_frame("frame-a").is_some());
        assert!(reader.get_frame("frame-b").is_some());
    }

    // ── Reader ───────────────────────────────────────────────────────────────

    #[test]
    fn test_reader_roundtrip_frame_data() {
        let tmp = TempDir::new().unwrap();
        let path = tmp.path().join("rt.nrv");

        let frame = test_frame("rt-frame");
        {
            let w = NRVWriter::create(path.to_str().unwrap()).unwrap();
            w.append_frame(&frame, true, 0.95).unwrap();
        }

        let reader = NRVReader::open(path.to_str().unwrap()).unwrap();
        let got = reader.get_frame("rt-frame").expect("frame must be readable");

        assert_eq!(got.id, frame.id);
        assert_eq!(got.vector, frame.vector);
        assert_eq!(got.seed, frame.seed);
        assert_eq!(got.thermo.temp_celsius, frame.thermo.temp_celsius);
        assert_eq!(got.thermo.voltage_v, frame.thermo.voltage_v);
        assert_eq!(got.thermo.freq_mhz, frame.thermo.freq_mhz);
        assert_eq!(got.thermo.fan_rpm, frame.thermo.fan_rpm);
        assert_eq!(got.proof, frame.proof);
    }

    #[test]
    fn test_reader_get_frame_not_found() {
        let tmp = TempDir::new().unwrap();
        let path = tmp.path().join("nf.nrv");
        {
            let w = NRVWriter::create(path.to_str().unwrap()).unwrap();
            w.append_frame(&test_frame("exists"), true, 0.8).unwrap();
        }
        let reader = NRVReader::open(path.to_str().unwrap()).unwrap();
        assert!(reader.get_frame("does-not-exist").is_none());
    }

    #[test]
    fn test_reader_get_modality_bytes() {
        let tmp = TempDir::new().unwrap();
        let path = tmp.path().join("mod.nrv");
        {
            let w = NRVWriter::create(path.to_str().unwrap()).unwrap();
            w.append_frame(&test_frame("m"), true, 0.9).unwrap();
        }

        let reader = NRVReader::open(path.to_str().unwrap()).unwrap();

        let vec_bytes = reader.get_modality("m", "vector").unwrap();
        assert_eq!(vec_bytes.len(), 48, "vector modality must be 48 bytes");
        // First float32 LE = 1.0
        assert_eq!(
            f32::from_le_bytes(vec_bytes[0..4].try_into().unwrap()),
            1.0f32
        );

        let seed_bytes = reader.get_modality("m", "seed").unwrap();
        assert_eq!(seed_bytes.len(), 32, "seed modality must be 32 bytes");
        assert_eq!(seed_bytes[0], 1); // seed[0] = 1

        let thermo_bytes = reader.get_modality("m", "thermo").unwrap();
        assert_eq!(thermo_bytes.len(), 16, "thermo modality must be 16 bytes");
        let temp = f32::from_le_bytes(thermo_bytes[0..4].try_into().unwrap());
        assert!((temp - 50.5).abs() < 0.001, "temp_celsius must round-trip");
    }

    #[test]
    fn test_reader_stream_frames() {
        let tmp = TempDir::new().unwrap();
        let path = tmp.path().join("stream.nrv");
        {
            let w = NRVWriter::create(path.to_str().unwrap()).unwrap();
            for i in 0..4u8 {
                let mut f = test_frame(&format!("f{}", i));
                f.vector = [i as f32; 12];
                w.append_frame(&f, true, 0.5).unwrap();
            }
        }

        let reader = NRVReader::open(path.to_str().unwrap()).unwrap();
        let frames: Vec<Frame> = reader.stream_frames().collect();
        assert_eq!(frames.len(), 4);
    }

    #[test]
    fn test_tombstone_hides_frame() {
        let tmp = TempDir::new().unwrap();
        let path = tmp.path().join("tomb.nrv");

        let w = NRVWriter::create(path.to_str().unwrap()).unwrap();
        w.append_frame(&test_frame("alive"), true, 0.8).unwrap();
        w.append_frame(&test_frame("dead"), true, 0.8).unwrap();
        w.set_tombstone("dead").unwrap();
        drop(w);

        let reader = NRVReader::open(path.to_str().unwrap()).unwrap();
        assert!(reader.get_frame("alive").is_some());
        assert!(reader.get_frame("dead").is_none(), "tombstoned frame must not be returned");

        let live: Vec<Frame> = reader.stream_frames().collect();
        assert_eq!(live.len(), 1);
        assert_eq!(live[0].id, "alive");
    }

    // ── App data directory ───────────────────────────────────────────────────

    /// Mirrors production: app writes .nrv files into an OS data directory.
    ///
    /// On Linux/macOS production systems the real path would be:
    ///   $HOME/.local/share/knirvbase/datasets/<collection>.nrv
    /// In tests we substitute a TempDir root while keeping the same structure.
    #[test]
    fn test_app_data_directory_lifecycle() {
        let tmp = TempDir::new().unwrap();
        let data_dir = app_data_dir(&tmp);

        let collection = "training_set_v1";
        let nrv_path = data_dir.join(format!("{}.nrv", collection));

        // NRVWriter must create all missing parent directories.
        let writer = NRVWriter::create(nrv_path.to_str().unwrap()).unwrap();
        assert!(data_dir.exists(), "app data dir must be created by NRVWriter");
        assert!(nrv_path.exists(), ".nrv file must exist in app data dir");

        // Simulate production inserts.
        for i in 0..5u8 {
            let mut f = test_frame(&format!("node-{}", i));
            f.vector = [i as f32; 12];
            f.seed = [i; 32];
            writer.append_frame(&f, i % 2 == 0, 0.6 + (i as f64 * 0.05)).unwrap();
        }

        let reg = writer.get_registry();
        assert_eq!(reg.frame_count, 5);
        assert_eq!(reg.global_metrics.verified_frame_count, 3); // frames 0, 2, 4
        drop(writer);

        // Re-open from the same path (simulates app restart).
        let reader = NRVReader::open(nrv_path.to_str().unwrap()).unwrap();
        assert_eq!(reader.registry().frame_count, 5);

        let frames: Vec<Frame> = reader.stream_frames().collect();
        assert_eq!(frames.len(), 5);

        let f2 = reader.get_frame("node-2").expect("node-2 must survive app restart");
        assert_eq!(f2.vector, [2.0f32; 12]);
        assert_eq!(f2.seed, [2u8; 32]);
    }

    #[test]
    fn test_multiple_collections_in_app_data_dir() {
        let tmp = TempDir::new().unwrap();
        let data_dir = app_data_dir(&tmp);

        let collections = ["errors", "capabilities", "ideas"];
        for name in &collections {
            let path = data_dir.join(format!("{}.nrv", name));
            let w = NRVWriter::create(path.to_str().unwrap()).unwrap();
            w.append_frame(&test_frame(&format!("{}-frame-0", name)), true, 0.9)
                .unwrap();
        }

        for name in &collections {
            let path = data_dir.join(format!("{}.nrv", name));
            assert!(path.exists(), "{}.nrv must exist", name);
            let r = NRVReader::open(path.to_str().unwrap()).unwrap();
            assert_eq!(r.registry().frame_count, 1, "{} must have 1 frame", name);
            assert!(r.get_frame(&format!("{}-frame-0", name)).is_some());
        }
    }

    // ── Cross-language binary format spec ─────────────────────────────────────
    //
    // These tests verify the binary frame encoding is identical to what the Go
    // and TypeScript implementations produce for the same input.  Any change
    // that breaks these assertions breaks cross-language .nrv compatibility.

    #[test]
    fn test_frame_binary_encoding_matches_spec() {
        let mut seed = [0u8; 32];
        for (i, b) in seed.iter_mut().enumerate() {
            *b = (i + 1) as u8;
        }

        let frame = Frame {
            id: "spec-frame".into(),
            vector: [1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0, 11.0, 12.0],
            seed,
            thermo: ThermoData {
                temp_celsius: 50.5,
                voltage_v: 12.0,
                freq_mhz: 500.0,
                fan_rpm: 3000.0,
            },
            proof: b"ok".to_vec(),
        };

        let (bytes, mods) = encode_frame(&frame);

        // Total = 48 + 32 + 16 + align8(2) = 104 bytes
        assert_eq!(bytes.len(), 104);

        // Vector offsets
        assert_eq!(mods["vector"], ModalityIndex { offset: 0, length: 48 });
        assert_eq!(mods["seed"], ModalityIndex { offset: 48, length: 32 });
        assert_eq!(mods["thermo"], ModalityIndex { offset: 80, length: 16 });
        assert_eq!(mods["proof"], ModalityIndex { offset: 96, length: 2 });

        // vector[0] = 1.0f32 LE
        assert_eq!(&bytes[0..4], &1.0f32.to_le_bytes());
        // vector[11] = 12.0f32 LE
        assert_eq!(&bytes[44..48], &12.0f32.to_le_bytes());

        // seed bytes 1..32
        assert_eq!(&bytes[48..80], &seed);

        // thermo.temp_celsius = 50.5
        assert_eq!(&bytes[80..84], &50.5f32.to_le_bytes());
        assert_eq!(&bytes[84..88], &12.0f32.to_le_bytes());
        assert_eq!(&bytes[88..92], &500.0f32.to_le_bytes());
        assert_eq!(&bytes[92..96], &3000.0f32.to_le_bytes());

        // proof "ok"
        assert_eq!(&bytes[96..98], b"ok");
        // remaining 6 bytes of alignment padding must be zero
        assert_eq!(&bytes[98..104], &[0u8; 6]);
    }

    #[test]
    fn test_encode_decode_roundtrip_binary() {
        let frame = test_frame("roundtrip");
        let entry = FrameEntry {
            id: "roundtrip".into(),
            offset: 0,
            length: 0,
            tombstone: None,
            verified: true,
            ergo_rank: 0.9,
            modalities: {
                let (_, m) = encode_frame(&frame);
                m
            },
        };

        let (bytes, _) = encode_frame(&frame);
        let decoded = decode_frame(&bytes, &entry).unwrap();

        assert_eq!(decoded.id, frame.id);
        assert_eq!(decoded.vector, frame.vector);
        assert_eq!(decoded.seed, frame.seed);
        assert_eq!(decoded.thermo.temp_celsius, frame.thermo.temp_celsius);
        assert_eq!(decoded.proof, frame.proof);
    }
}
