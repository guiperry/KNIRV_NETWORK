pub mod lsa;
pub mod tfidf;

pub use lsa::LsaReducer;
pub use tfidf::TfidfVectorizer;

use std::collections::HashMap;
use std::sync::{Arc, RwLock};

pub trait Embedder: Send + Sync {
    fn generate(&self, text: &str) -> Result<Vec<f32>, String>;
    fn fit(&self, documents: &[String]) -> Result<(), String>;
    fn fit_incremental(&self, document: &str) -> Result<(), String>;
    fn dimension(&self) -> usize;
}

pub trait Storage: Send + Sync {
    fn put(&self, key: &str, value: &[u8]) -> Result<(), String>;
    fn get(&self, key: &str) -> Result<Option<Vec<u8>>, String>;
    fn has(&self, key: &str) -> Result<bool, String>;
}

pub struct TfidfEmbedder {
    vectorizer: RwLock<TfidfVectorizer>,
    reducer: RwLock<LsaReducer>,
    storage: Option<Arc<dyn Storage>>,
    dimension: usize,
    fitted: RwLock<bool>,
}

impl TfidfEmbedder {
    pub fn new(storage: Option<Arc<dyn Storage>>, dimension: usize) -> Result<Self, String> {
        if dimension == 0 {
            return Err("dimension must be positive".to_string());
        }

        let embedder = Self {
            vectorizer: RwLock::new(TfidfVectorizer::new()),
            reducer: RwLock::new(LsaReducer::new(dimension)),
            storage,
            dimension,
            fitted: RwLock::new(false),
        };

        if let Some(ref st) = embedder.storage {
            if st.has("embedding:vocabulary").unwrap_or(false) {
                embedder.load_vocabulary()?;
                *embedder.fitted.write().unwrap() = true;
            }
        }

        Ok(embedder)
    }

    pub fn dimension(&self) -> usize {
        self.dimension
    }

    pub fn fit(&self, documents: &[String]) -> Result<(), String> {
        if documents.is_empty() {
            return Err("no documents provided for training".to_string());
        }

        {
            let mut vectorizer = self.vectorizer.write().unwrap();
            vectorizer.fit(documents)?;
        }

        let vectors = {
            let vectorizer = self.vectorizer.read().unwrap();
            let mut vectors = Vec::new();
            for doc in documents {
                if let Ok(vec) = vectorizer.transform(doc) {
                    vectors.push(vec);
                }
            }
            vectors
        };

        if !vectors.is_empty() && vectors[0].len() >= self.dimension {
            let mut reducer = self.reducer.write().unwrap();
            reducer.fit(&vectors)?;
        }

        if let Some(ref st) = self.storage {
            self.save_vocabulary(st)?;
        }

        *self.fitted.write().unwrap() = true;
        Ok(())
    }

    pub fn fit_incremental(&self, document: &str) -> Result<(), String> {
        if document.is_empty() {
            return Err("empty document provided".to_string());
        }

        {
            let mut vectorizer = self.vectorizer.write().unwrap();
            vectorizer.fit_incremental(document);
        }

        if let Some(ref st) = self.storage {
            self.save_vocabulary(st)?;
        }

        *self.fitted.write().unwrap() = true;
        Ok(())
    }

    pub fn generate(&self, text: &str) -> Result<Vec<f32>, String> {
        if !*self.fitted.read().unwrap() {
            return Ok(vec![0.0; self.dimension]);
        }

        let tfidf_vector = {
            let vectorizer = self.vectorizer.read().unwrap();
            vectorizer.transform(text)?
        };

        let reducer = self.reducer.read().unwrap();

        if tfidf_vector.len() < self.dimension || reducer.source_dimension() == 0 {
            return Ok(self.normalize_to_target(&tfidf_vector));
        }

        let reduced = reducer.transform(&tfidf_vector)?;

        let mut result = Vec::with_capacity(reduced.len());
        for val in reduced {
            result.push(val as f32);
        }

        Ok(result)
    }

    fn normalize_to_target(&self, vector: &[f64]) -> Vec<f32> {
        let mut result = vec![0.0; self.dimension];

        let copy_len = vector.len().min(self.dimension);
        for i in 0..copy_len {
            result[i] = vector[i] as f32;
        }

        result
    }

    fn save_vocabulary(&self, storage: &Arc<dyn Storage>) -> Result<(), String> {
        let vectorizer = self.vectorizer.read().unwrap();

        let vocab_json = serde_json::to_string(&vectorizer.get_vocabulary())
            .map_err(|e| format!("failed to marshal vocabulary: {}", e))?;
        let idf_json = serde_json::to_string(&vectorizer.get_idf())
            .map_err(|e| format!("failed to marshal IDF: {}", e))?;
        let doc_count_json = serde_json::to_string(&vectorizer.get_doc_count())
            .map_err(|e| format!("failed to marshal doc_count: {}", e))?;
        let word_doc_counts_json = serde_json::to_string(&vectorizer.get_word_doc_counts())
            .map_err(|e| format!("failed to marshal word_doc_counts: {}", e))?;

        storage.put("embedding:vocabulary", vocab_json.as_bytes())?;
        storage.put("embedding:idf", idf_json.as_bytes())?;
        storage.put("embedding:doc_count", doc_count_json.as_bytes())?;
        storage.put("embedding:word_doc_counts", word_doc_counts_json.as_bytes())?;

        Ok(())
    }

    fn load_vocabulary(&self) -> Result<(), String> {
        let storage = self.storage.as_ref().ok_or("no storage configured")?;

        let vocab_data = storage
            .get("embedding:vocabulary")?
            .ok_or("vocabulary not found in storage")?;
        let idf_data = storage
            .get("embedding:idf")?
            .ok_or("idf not found in storage")?;
        let doc_count_data = storage
            .get("embedding:doc_count")?
            .ok_or("doc_count not found in storage")?;
        let word_doc_counts_data = storage
            .get("embedding:word_doc_counts")?
            .ok_or("word_doc_counts not found in storage")?;

        let vocab: HashMap<String, usize> = serde_json::from_slice(&vocab_data)
            .map_err(|e| format!("failed to unmarshal vocabulary: {}", e))?;
        let idf: HashMap<String, f64> = serde_json::from_slice(&idf_data)
            .map_err(|e| format!("failed to unmarshal IDF: {}", e))?;
        let doc_count: usize = serde_json::from_slice(&doc_count_data)
            .map_err(|e| format!("failed to unmarshal doc_count: {}", e))?;
        let word_doc_counts: HashMap<String, usize> = serde_json::from_slice(&word_doc_counts_data)
            .map_err(|e| format!("failed to unmarshal word_doc_counts: {}", e))?;

        let mut vectorizer = self.vectorizer.write().unwrap();
        vectorizer.restore_vocabulary(vocab, idf, doc_count, word_doc_counts);

        Ok(())
    }
}

impl Embedder for TfidfEmbedder {
    fn generate(&self, text: &str) -> Result<Vec<f32>, String> {
        self.generate(text)
    }

    fn fit(&self, documents: &[String]) -> Result<(), String> {
        self.fit(documents)
    }

    fn fit_incremental(&self, document: &str) -> Result<(), String> {
        self.fit_incremental(document)
    }

    fn dimension(&self) -> usize {
        self.dimension()
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::sync::RwLock;

    struct InMemoryStorage {
        data: RwLock<HashMap<String, Vec<u8>>>,
    }

    impl InMemoryStorage {
        fn new() -> Self {
            Self {
                data: RwLock::new(HashMap::new()),
            }
        }
    }

    impl Storage for InMemoryStorage {
        fn put(&self, key: &str, value: &[u8]) -> Result<(), String> {
            self.data
                .write()
                .map_err(|e| format!("lock error: {}", e))?
                .insert(key.to_string(), value.to_vec());
            Ok(())
        }

        fn get(&self, key: &str) -> Result<Option<Vec<u8>>, String> {
            Ok(self
                .data
                .read()
                .map_err(|e| format!("lock error: {}", e))?
                .get(key)
                .cloned())
        }

        fn has(&self, key: &str) -> Result<bool, String> {
            Ok(self
                .data
                .read()
                .map_err(|e| format!("lock error: {}", e))?
                .contains_key(key))
        }
    }

    #[test]
    fn test_tfidf_embedder() {
        let storage = Arc::new(InMemoryStorage::new());
        let embedder = TfidfEmbedder::new(Some(storage), 10).unwrap();

        let docs = vec![
            "hello world".to_string(),
            "hello rust".to_string(),
            "world rust".to_string(),
        ];

        embedder.fit(&docs).unwrap();

        let embedding = embedder.generate("hello world").unwrap();
        assert_eq!(embedding.len(), 10);
    }
}
