use std::collections::HashMap;
use std::sync::{Arc, RwLock};

pub struct Metrics {
    counters: Arc<RwLock<HashMap<String, u64>>>,
    gauges: Arc<RwLock<HashMap<String, f64>>>,
    histograms: Arc<RwLock<HashMap<String, Vec<f64>>>>,
}

#[derive(Debug, Clone)]
pub struct Counter {
    name: String,
    metrics: Arc<RwLock<HashMap<String, u64>>>,
}

impl Counter {
    pub fn new(name: String, metrics: Arc<RwLock<HashMap<String, u64>>>) -> Self {
        {
            let mut m = metrics.write().unwrap();
            m.entry(name.clone()).or_insert(0);
        }
        Self { name, metrics }
    }

    pub fn inc(&self) {
        let mut m = self.metrics.write().unwrap();
        if let Some(v) = m.get_mut(&self.name) {
            *v += 1;
        }
    }

    pub fn inc_by(&self, n: u64) {
        let mut m = self.metrics.write().unwrap();
        if let Some(v) = m.get_mut(&self.name) {
            *v += n;
        }
    }

    pub fn get(&self) -> u64 {
        *self.metrics.read().unwrap().get(&self.name).unwrap_or(&0)
    }
}

#[derive(Debug, Clone)]
pub struct Gauge {
    name: String,
    metrics: Arc<RwLock<HashMap<String, f64>>>,
}

impl Gauge {
    pub fn new(name: String, metrics: Arc<RwLock<HashMap<String, f64>>>) -> Self {
        {
            let mut m = metrics.write().unwrap();
            m.entry(name.clone()).or_insert(0.0);
        }
        Self { name, metrics }
    }

    pub fn set(&self, v: f64) {
        let mut m = self.metrics.write().unwrap();
        m.insert(self.name.clone(), v);
    }

    pub fn get(&self) -> f64 {
        *self.metrics.read().unwrap().get(&self.name).unwrap_or(&0.0)
    }

    pub fn inc(&self) {
        let mut m = self.metrics.write().unwrap();
        if let Some(v) = m.get_mut(&self.name) {
            *v += 1.0;
        }
    }

    pub fn dec(&self) {
        let mut m = self.metrics.write().unwrap();
        if let Some(v) = m.get_mut(&self.name) {
            *v -= 1.0;
        }
    }
}

pub struct Histogram {
    name: String,
    metrics: Arc<RwLock<HashMap<String, Vec<f64>>>>,
}

impl Histogram {
    pub fn new(name: String, metrics: Arc<RwLock<HashMap<String, Vec<f64>>>>) -> Self {
        Self { name, metrics }
    }

    pub fn observe(&self, v: f64) {
        let mut m = self.metrics.write().unwrap();
        m.entry(self.name.clone()).or_insert_with(Vec::new).push(v);
    }
}

impl Metrics {
    pub fn new() -> Self {
        Self {
            counters: Arc::new(RwLock::new(HashMap::new())),
            gauges: Arc::new(RwLock::new(HashMap::new())),
            histograms: Arc::new(RwLock::new(HashMap::new())),
        }
    }

    pub fn blocks_committed_total(&self) -> Counter {
        Counter::new(
            "knirvbase_blocks_committed_total".to_string(),
            self.counters.clone(),
        )
    }

    pub fn block_commit_duration_seconds(&self) -> Histogram {
        Histogram::new(
            "knirvbase_block_commit_duration_seconds".to_string(),
            self.histograms.clone(),
        )
    }

    pub fn memory_store_ops_total(&self) -> Counter {
        Counter::new(
            "knirvbase_memory_store_ops_total".to_string(),
            self.counters.clone(),
        )
    }

    pub fn memory_retrieve_ops_total(&self) -> Counter {
        Counter::new(
            "knirvbase_memory_retrieve_ops_total".to_string(),
            self.counters.clone(),
        )
    }

    pub fn cache_hits_total(&self) -> Counter {
        Counter::new(
            "knirvbase_cache_hits_total".to_string(),
            self.counters.clone(),
        )
    }

    pub fn cache_misses_total(&self) -> Counter {
        Counter::new(
            "knirvbase_cache_misses_total".to_string(),
            self.counters.clone(),
        )
    }

    pub fn active_connections(&self) -> Gauge {
        Gauge::new(
            "knirvbase_active_connections".to_string(),
            self.gauges.clone(),
        )
    }

    pub fn nrn_balance(&self) -> Gauge {
        Gauge::new("knirvbase_nrn_balance".to_string(), self.gauges.clone())
    }

    pub fn query_latency_seconds(&self) -> Histogram {
        Histogram::new(
            "knirvbase_query_latency_seconds".to_string(),
            self.histograms.clone(),
        )
    }

    pub fn error_count_total(&self) -> Counter {
        Counter::new(
            "knirvbase_error_count_total".to_string(),
            self.counters.clone(),
        )
    }

    pub fn index_size(&self) -> Gauge {
        Gauge::new("knirvbase_index_size".to_string(), self.gauges.clone())
    }
}

impl Default for Metrics {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_counter() {
        let metrics = Metrics::new();
        let counter = metrics.blocks_committed_total();
        counter.inc();
        counter.inc();
        assert_eq!(counter.get(), 2);
    }

    #[test]
    fn test_gauge() {
        let metrics = Metrics::new();
        let gauge = metrics.active_connections();
        gauge.set(5.0);
        assert_eq!(gauge.get(), 5.0);
    }

    #[test]
    fn test_histogram() {
        let metrics = Metrics::new();
        let hist = metrics.query_latency_seconds();
        hist.observe(0.1);
        hist.observe(0.2);
    }
}
