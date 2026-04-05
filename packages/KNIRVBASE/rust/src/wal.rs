use parking_lot::Mutex;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::fs::{self, File, OpenOptions};
use std::io::{BufRead, BufReader, Write};
use std::path::Path;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct WALEntry {
    pub frame_id: String,
    pub last_good_length: i64,
    pub committed: bool,
}

pub struct WAL {
    path: String,
    mu: Mutex<()>,
}

impl WAL {
    pub fn new(path: String) -> Self {
        Self {
            path,
            mu: Mutex::new(()),
        }
    }

    pub fn begin(&self, entry: WALEntry) -> Result<(), String> {
        let _guard = self.mu.lock();

        let mut file = OpenOptions::new()
            .create(true)
            .append(true)
            .open(&self.path)
            .map_err(|e| format!("failed to open WAL: {}", e))?;

        let json = serde_json::to_string(&entry)
            .map_err(|e| format!("failed to serialize entry: {}", e))?;

        writeln!(file, "{}", json).map_err(|e| format!("failed to write entry: {}", e))?;

        Ok(())
    }

    pub fn commit(&self, frame_id: &str) -> Result<(), String> {
        let _guard = self.mu.lock();

        let entries = self.read_entries()?;

        let mut file =
            File::create(&self.path).map_err(|e| format!("failed to create WAL file: {}", e))?;

        for mut entry in entries {
            if entry.frame_id == frame_id {
                entry.committed = true;
            }
            let json = serde_json::to_string(&entry)
                .map_err(|e| format!("failed to serialize entry: {}", e))?;
            writeln!(file, "{}", json).map_err(|e| format!("failed to write entry: {}", e))?;
        }

        Ok(())
    }

    pub fn recover(&self) -> Result<i64, String> {
        let _guard = self.mu.lock();

        let entries = self.read_entries()?;

        let mut min_uncommitted: Option<i64> = None;
        let mut has_any_entry = false;

        for entry in &entries {
            has_any_entry = true;
            if !entry.committed {
                min_uncommitted = Some(
                    min_uncommitted
                        .map(|m| m.min(entry.last_good_length))
                        .unwrap_or(entry.last_good_length),
                );
            }
        }

        if min_uncommitted.is_none() && has_any_entry {
            return Ok(-1);
        }

        Ok(min_uncommitted.unwrap_or(-1))
    }

    pub fn truncate(&self) -> Result<(), String> {
        let _guard = self.mu.lock();

        if Path::new(&self.path).exists() {
            fs::remove_file(&self.path).map_err(|e| format!("failed to remove WAL: {}", e))?;
        }

        Ok(())
    }

    fn read_entries(&self) -> Result<Vec<WALEntry>, String> {
        let file = match File::open(&self.path) {
            Ok(f) => f,
            Err(e) if e.kind() == std::io::ErrorKind::NotFound => return Ok(Vec::new()),
            Err(e) => return Err(format!("failed to open WAL: {}", e)),
        };

        let reader = BufReader::new(file);
        let mut entries = Vec::new();

        for line in reader.lines() {
            let line = match line {
                Ok(l) => l,
                Err(_) => continue,
            };

            if let Ok(entry) = serde_json::from_str::<WALEntry>(&line) {
                entries.push(entry);
            }
        }

        Ok(entries)
    }

    pub fn len(&self) -> usize {
        self.read_entries().map(|e| e.len()).unwrap_or(0)
    }

    pub fn is_empty(&self) -> bool {
        self.len() == 0
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use tempfile::NamedTempFile;

    #[test]
    fn test_wal_begin_and_commit() {
        let temp_file = NamedTempFile::new().unwrap();
        let path = temp_file.path().to_string_lossy().to_string();
        drop(temp_file);

        let wal = WAL::new(path.clone());

        let entry = WALEntry {
            frame_id: "frame1".to_string(),
            last_good_length: 100,
            committed: false,
        };

        wal.begin(entry).unwrap();
        assert_eq!(wal.len(), 1);

        wal.commit("frame1").unwrap();

        let recovered = wal.recover().unwrap();
        assert_eq!(recovered, -1);
    }

    #[test]
    fn test_wal_recover_uncommitted() {
        let temp_file = NamedTempFile::new().unwrap();
        let path = temp_file.path().to_string_lossy().to_string();
        drop(temp_file);

        let wal = WAL::new(path.clone());

        wal.begin(WALEntry {
            frame_id: "frame1".to_string(),
            last_good_length: 100,
            committed: false,
        })
        .unwrap();

        wal.begin(WALEntry {
            frame_id: "frame2".to_string(),
            last_good_length: 200,
            committed: false,
        })
        .unwrap();

        let recovered = wal.recover().unwrap();
        assert_eq!(recovered, 100);
    }

    #[test]
    fn test_wal_truncate() {
        let temp_file = NamedTempFile::new().unwrap();
        let path = temp_file.path().to_string_lossy().to_string();
        drop(temp_file);

        let wal = WAL::new(path.clone());

        wal.begin(WALEntry {
            frame_id: "frame1".to_string(),
            last_good_length: 100,
            committed: false,
        })
        .unwrap();

        assert_eq!(wal.len(), 1);

        wal.truncate().unwrap();
        assert!(wal.is_empty());
    }
}
