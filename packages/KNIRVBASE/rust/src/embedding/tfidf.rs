use serde::{Deserialize, Serialize};
use std::collections::HashMap;

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct TfidfVectorizer {
    #[serde(skip)]
    vocabulary: HashMap<String, usize>,
    #[serde(skip)]
    idf: HashMap<String, f64>,
    doc_count: usize,
    #[serde(skip)]
    word_doc_counts: HashMap<String, usize>,
}

impl TfidfVectorizer {
    pub fn new() -> Self {
        Self {
            vocabulary: HashMap::new(),
            idf: HashMap::new(),
            doc_count: 0,
            word_doc_counts: HashMap::new(),
        }
    }

    pub fn fit(&mut self, documents: &[String]) -> Result<(), String> {
        if documents.is_empty() {
            return Err("no documents provided".to_string());
        }

        self.vocabulary.clear();
        self.idf.clear();
        self.word_doc_counts.clear();
        self.doc_count = documents.len();

        for doc in documents {
            let tokens = tokenize(doc);
            let mut unique_words: HashMap<String, bool> = HashMap::new();

            for token in tokens {
                unique_words.insert(token, true);
            }

            for word in unique_words.keys() {
                *self.word_doc_counts.entry(word.clone()).or_insert(0) += 1;
            }
        }

        let mut idx = 0usize;
        for (word, doc_freq) in &self.word_doc_counts {
            self.vocabulary.insert(word.clone(), idx);
            self.idf.insert(
                word.clone(),
                ((self.doc_count as f64) / (*doc_freq as f64)).ln(),
            );
            idx += 1;
        }

        Ok(())
    }

    pub fn fit_incremental(&mut self, document: &str) {
        let tokens = tokenize(document);
        let mut unique_words: HashMap<String, bool> = HashMap::new();

        for token in tokens {
            unique_words.insert(token, true);
        }

        self.doc_count += 1;

        for word in unique_words.keys() {
            *self.word_doc_counts.entry(word.clone()).or_insert(0) += 1;

            if !self.vocabulary.contains_key(word) {
                self.vocabulary.insert(word.clone(), self.vocabulary.len());
            }

            self.idf.insert(
                word.clone(),
                ((self.doc_count as f64) / (*self.word_doc_counts.get(word).unwrap() as f64)).ln(),
            );
        }

        for word in self.vocabulary.keys() {
            if !unique_words.contains_key(word) {
                self.idf.insert(
                    word.clone(),
                    ((self.doc_count as f64)
                        / (*self.word_doc_counts.get(word).unwrap_or(&1) as f64))
                        .ln(),
                );
            }
        }
    }

    pub fn transform(&self, document: &str) -> Result<Vec<f64>, String> {
        if self.vocabulary.is_empty() {
            return Err("vectorizer not fitted".to_string());
        }

        let tokens = tokenize(document);
        let mut tf: HashMap<String, f64> = HashMap::new();

        for token in &tokens {
            *tf.entry(token.clone()).or_insert(0.0) += 1.0;
        }

        let doc_length = tokens.len() as f64;
        if doc_length > 0.0 {
            for val in tf.values_mut() {
                *val /= doc_length;
            }
        }

        let mut vector = vec![0.0; self.vocabulary.len()];
        for (word, freq) in tf {
            if let Some(&idx) = self.vocabulary.get(&word) {
                if let Some(&idf_val) = self.idf.get(&word) {
                    vector[idx] = freq * idf_val;
                }
            }
        }

        Ok(vector)
    }

    pub fn fit_transform(&mut self, documents: &[String]) -> Result<Vec<Vec<f64>>, String> {
        self.fit(documents)?;

        let mut vectors = Vec::new();
        for doc in documents {
            if let Ok(vec) = self.transform(doc) {
                vectors.push(vec);
            }
        }

        Ok(vectors)
    }

    pub fn vocabulary_size(&self) -> usize {
        self.vocabulary.len()
    }

    pub fn get_vocabulary(&self) -> HashMap<String, usize> {
        self.vocabulary.clone()
    }
}

impl Default for TfidfVectorizer {
    fn default() -> Self {
        Self::new()
    }
}

fn tokenize(text: &str) -> Vec<String> {
    let text_lower = text.to_lowercase();
    let mut tokens = Vec::new();
    let mut current_token = String::new();

    let stop_words = [
        "is", "a", "an", "the", "and", "or", "but", "in", "on", "at", "to", "for", "of", "with",
        "by",
    ];

    for ch in text_lower.chars() {
        if ch.is_alphanumeric() {
            current_token.push(ch);
        } else {
            if current_token.len() > 1 && !stop_words.contains(&current_token.as_str()) {
                tokens.push(current_token.clone());
            }
            current_token.clear();
        }
    }

    if current_token.len() > 1 && !stop_words.contains(&current_token.as_str()) {
        tokens.push(current_token);
    }

    tokens
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_tokenize() {
        let tokens = tokenize("Hello World! This is a test.");
        assert_eq!(tokens, vec!["hello", "world", "this", "test"]);
    }

    #[test]
    fn test_tfidf_fit_transform() {
        let mut vectorizer = TfidfVectorizer::new();
        let docs = vec![
            "hello world".to_string(),
            "hello rust".to_string(),
            "world rust".to_string(),
        ];

        let vectors = vectorizer.fit_transform(&docs).unwrap();
        assert_eq!(vectors.len(), 3);
        assert!(vectorizer.vocabulary_size() > 0);
    }
}
