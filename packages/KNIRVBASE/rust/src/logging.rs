use chrono::Utc;
use std::collections::HashMap;
use std::fmt;
use std::sync::Mutex;

pub struct Logger {
    level: Level,
    output: Mutex<Vec<LogEntry>>,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Level {
    Debug,
    Info,
    Warn,
    Error,
}

impl Level {
    fn from_str(s: &str) -> Self {
        match s.to_lowercase().as_str() {
            "debug" => Level::Debug,
            "info" => Level::Info,
            "warn" => Level::Warn,
            "error" => Level::Error,
            _ => Level::Info,
        }
    }

    fn is_enabled(&self, other: Level) -> bool {
        match (self, other) {
            (Level::Debug, _) => true,
            (Level::Info, Level::Debug) => false,
            (Level::Info, _) => true,
            (Level::Warn, Level::Error) => false,
            (Level::Warn, _) => true,
            (Level::Error, Level::Error) => true,
            (Level::Error, _) => false,
        }
    }
}

impl fmt::Display for Level {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Level::Debug => write!(f, "DEBUG"),
            Level::Info => write!(f, "INFO"),
            Level::Warn => write!(f, "WARN"),
            Level::Error => write!(f, "ERROR"),
        }
    }
}

#[derive(Debug, Clone)]
pub struct LogEntry {
    pub timestamp: String,
    pub level: Level,
    pub message: String,
    pub block_id: Option<String>,
    pub user_id: Option<String>,
    pub error: Option<String>,
    pub context: HashMap<String, String>,
}

impl Logger {
    pub fn new(level: &str) -> Self {
        Self {
            level: Level::from_str(level),
            output: Mutex::new(Vec::new()),
        }
    }

    pub fn debug(&self, msg: &str) {
        self.log(Level::Debug, msg);
    }

    pub fn info(&self, msg: &str) {
        self.log(Level::Info, msg);
    }

    pub fn warn(&self, msg: &str) {
        self.log(Level::Warn, msg);
    }

    pub fn error(&self, msg: &str) {
        self.log(Level::Error, msg);
    }

    fn log(&self, level: Level, msg: &str) {
        if !self.level.is_enabled(level) {
            return;
        }

        let entry = LogEntry {
            timestamp: Utc::now().format("%Y-%m-%dT%H:%M:%S%.3fZ").to_string(),
            level,
            message: msg.to_string(),
            block_id: None,
            user_id: None,
            error: None,
            context: HashMap::new(),
        };

        let mut output = self.output.lock().unwrap();
        output.push(entry);
    }

    pub fn with_block_id(&self, block_id: &str, msg: &str) {
        self.log_with_context(Level::Info, msg, Some(block_id), None, None);
    }

    pub fn with_user_id(&self, user_id: &str, msg: &str) {
        self.log_with_context(Level::Info, msg, None, Some(user_id), None);
    }

    pub fn with_error(&self, err: &str, msg: &str) {
        self.log_with_context(Level::Error, msg, None, None, Some(err));
    }

    fn log_with_context(
        &self,
        level: Level,
        msg: &str,
        block_id: Option<&str>,
        user_id: Option<&str>,
        err: Option<&str>,
    ) {
        if !self.level.is_enabled(level) {
            return;
        }

        let entry = LogEntry {
            timestamp: Utc::now().format("%Y-%m-%dT%H:%M:%S%.3fZ").to_string(),
            level,
            message: msg.to_string(),
            block_id: block_id.map(String::from),
            user_id: user_id.map(String::from),
            error: err.map(String::from),
            context: HashMap::new(),
        };

        let mut output = self.output.lock().unwrap();
        output.push(entry);
    }

    pub fn get_entries(&self) -> Vec<LogEntry> {
        self.output.lock().unwrap().clone()
    }

    pub fn clear(&self) {
        self.output.lock().unwrap().clear();
    }
}

impl Default for Logger {
    fn default() -> Self {
        Self::new("info")
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_logger() {
        let logger = Logger::new("debug");
        logger.info("test info message");
        logger.error("test error message");

        let entries = logger.get_entries();
        assert_eq!(entries.len(), 2);
    }

    #[test]
    fn test_log_context() {
        let logger = Logger::new("debug");
        logger.with_block_id("block123", "test with block");
        logger.with_user_id("user456", "test with user");
        logger.with_error("error occurred", "test with error");

        let entries = logger.get_entries();
        assert_eq!(entries.len(), 3);
    }
}
