use crate::hnsw::HNSWIndex;
use parking_lot::RwLock;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use uuid::Uuid;

#[derive(Debug, Clone, PartialEq, Eq, Serialize, Deserialize)]
pub enum IndexType {
    BTree,
    GIN,
    HNSW,
    Tag,
}

impl std::fmt::Display for IndexType {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            IndexType::BTree => write!(f, "btree"),
            IndexType::GIN => write!(f, "gin"),
            IndexType::HNSW => write!(f, "hnsw"),
            IndexType::Tag => write!(f, "tag"),
        }
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Index {
    pub name: String,
    pub collection: String,
    pub index_type: IndexType,
    pub fields: Vec<String>,
    pub unique: bool,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub partial_expr: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub options: Option<HashMap<String, serde_json::Value>>,
}

impl Index {
    pub fn new(
        name: String,
        collection: String,
        index_type: IndexType,
        fields: Vec<String>,
        unique: bool,
    ) -> Self {
        Self {
            name,
            collection,
            index_type,
            fields,
            unique,
            partial_expr: None,
            options: None,
        }
    }
}

struct BTreeIndexData {
    data: HashMap<String, Vec<String>>,
}

struct GinIndexData {
    data: HashMap<String, Vec<String>>,
}

struct HnswIndexData {
    index: HNSWIndex,
    vectors: HashMap<String, Vec<f32>>,
}

struct TagBlock {
    id: Uuid,
    timestamp: i64,
    category: String,
    semantic_vector: Option<Vec<f32>>,
    tags: Vec<String>,
}

struct TagIndexData {
    blocks: HashMap<Uuid, TagBlock>,
    tag_index: HashMap<String, Vec<Uuid>>,
}

pub struct IndexManager {
    base_dir: String,
    indexes: RwLock<HashMap<String, Index>>,
    btree_indexes: RwLock<HashMap<String, BTreeIndexData>>,
    gin_indexes: RwLock<HashMap<String, GinIndexData>>,
    hnsw_indexes: RwLock<HashMap<String, HnswIndexData>>,
    tag_indexes: RwLock<HashMap<String, TagIndexData>>,
}

impl IndexManager {
    pub fn new(base_dir: String) -> Self {
        Self {
            base_dir,
            indexes: RwLock::new(HashMap::new()),
            btree_indexes: RwLock::new(HashMap::new()),
            gin_indexes: RwLock::new(HashMap::new()),
            hnsw_indexes: RwLock::new(HashMap::new()),
            tag_indexes: RwLock::new(HashMap::new()),
        }
    }

    pub fn create_index(
        &self,
        collection: &str,
        name: &str,
        index_type: IndexType,
        fields: Vec<String>,
        unique: bool,
        partial_expr: Option<String>,
        options: Option<HashMap<String, serde_json::Value>>,
    ) -> Result<(), String> {
        let key = format!("{}:{}", collection, name);

        if self.indexes.read().contains_key(&key) {
            return Err(format!("index {} already exists", key));
        }

        let index = Index {
            name: name.to_string(),
            collection: collection.to_string(),
            index_type: index_type.clone(),
            fields,
            unique,
            partial_expr,
            options: options.clone(),
        };

        match index_type {
            IndexType::BTree => {
                self.btree_indexes.write().insert(
                    key.clone(),
                    BTreeIndexData {
                        data: HashMap::new(),
                    },
                );
            }
            IndexType::GIN => {
                self.gin_indexes.write().insert(
                    key.clone(),
                    GinIndexData {
                        data: HashMap::new(),
                    },
                );
            }
            IndexType::HNSW => {
                let dimensions = options
                    .as_ref()
                    .and_then(|o| o.get("dimensions"))
                    .and_then(|v| v.as_u64())
                    .unwrap_or(768) as usize;
                let m = options
                    .as_ref()
                    .and_then(|o| o.get("m"))
                    .and_then(|v| v.as_u64())
                    .unwrap_or(16) as usize;
                let ef_construction = options
                    .as_ref()
                    .and_then(|o| o.get("ef_construction"))
                    .and_then(|v| v.as_u64())
                    .unwrap_or(200) as usize;

                self.hnsw_indexes.write().insert(
                    key.clone(),
                    HnswIndexData {
                        index: HNSWIndex::new(dimensions, m, ef_construction),
                        vectors: HashMap::new(),
                    },
                );
            }
            IndexType::Tag => {
                self.tag_indexes.write().insert(
                    key.clone(),
                    TagIndexData {
                        blocks: HashMap::new(),
                        tag_index: HashMap::new(),
                    },
                );
            }
        }

        self.indexes.write().insert(key, index);
        Ok(())
    }

    pub fn drop_index(&self, collection: &str, name: &str) -> Result<(), String> {
        let key = format!("{}:{}", collection, name);

        if !self.indexes.write().remove(&key).is_some() {
            return Err(format!("index {} does not exist", key));
        }

        self.btree_indexes.write().remove(&key);
        self.gin_indexes.write().remove(&key);
        self.hnsw_indexes.write().remove(&key);
        self.tag_indexes.write().remove(&key);

        Ok(())
    }

    pub fn get_index(&self, collection: &str, name: &str) -> Option<Index> {
        let key = format!("{}:{}", collection, name);
        self.indexes.read().get(&key).cloned()
    }

    pub fn get_indexes_for_collection(&self, collection: &str) -> Vec<Index> {
        self.indexes
            .read()
            .values()
            .filter(|idx| idx.collection == collection)
            .cloned()
            .collect()
    }

    pub fn insert(
        &self,
        collection: &str,
        doc: &HashMap<String, serde_json::Value>,
    ) -> Result<(), String> {
        let indexes = self.get_indexes_for_collection(collection);

        for idx in indexes {
            if !self.matches_partial(&idx, doc) {
                continue;
            }

            let doc_id = doc
                .get("id")
                .and_then(|v| v.as_str())
                .ok_or("document must have an 'id' field")?
                .to_string();

            match idx.index_type {
                IndexType::BTree => self.insert_btree(&idx, &doc_id, doc)?,
                IndexType::GIN => self.insert_gin(&idx, &doc_id, doc)?,
                IndexType::HNSW => self.insert_hnsw(&idx, &doc_id, doc)?,
                IndexType::Tag => self.insert_tag(&idx, &doc_id, doc)?,
            }
        }

        Ok(())
    }

    pub fn delete(&self, collection: &str, doc_id: &str) -> Result<(), String> {
        let indexes = self.get_indexes_for_collection(collection);

        for idx in indexes {
            match idx.index_type {
                IndexType::BTree => self.delete_btree(&idx, doc_id)?,
                IndexType::GIN => self.delete_gin(&idx, doc_id)?,
                IndexType::HNSW => self.delete_hnsw(&idx, doc_id)?,
                IndexType::Tag => self.delete_tag(&idx, doc_id)?,
            }
        }

        Ok(())
    }

    pub fn query_index(
        &self,
        collection: &str,
        index_name: &str,
        query: &HashMap<String, serde_json::Value>,
    ) -> Result<Vec<String>, String> {
        let idx = self
            .get_index(collection, index_name)
            .ok_or_else(|| format!("index {}:{} not found", collection, index_name))?;

        match idx.index_type {
            IndexType::BTree => self.query_btree(&idx, query),
            IndexType::GIN => self.query_gin(&idx, query),
            IndexType::HNSW => self.query_hnsw(&idx, query),
            IndexType::Tag => self.query_tag(&idx, query),
        }
    }

    fn matches_partial(&self, idx: &Index, doc: &HashMap<String, serde_json::Value>) -> bool {
        if idx.partial_expr.is_none() {
            return true;
        }

        let expr = idx.partial_expr.as_ref().unwrap();
        let parts: Vec<&str> = expr.split('=').collect();
        if parts.len() != 2 {
            return true;
        }

        let field = parts[0].trim();
        let expected = parts[1].trim();

        if let Some(val) = doc.get(field) {
            return val.to_string().trim() == expected;
        }

        false
    }

    fn insert_btree(
        &self,
        idx: &Index,
        doc_id: &str,
        doc: &HashMap<String, serde_json::Value>,
    ) -> Result<(), String> {
        let key = format!("{}:{}", idx.collection, idx.name);
        let mut indexes = self.btree_indexes.write();

        if let Some(btree) = indexes.get_mut(&key) {
            let key_value = self.build_composite_key(&idx.fields, doc);

            if idx.unique {
                if let Some(existing) = btree.data.get(&key_value) {
                    if !existing.is_empty() {
                        return Ok(());
                    }
                }
            }

            btree
                .data
                .entry(key_value)
                .or_insert_with(Vec::new)
                .push(doc_id.to_string());
        }

        Ok(())
    }

    fn delete_btree(&self, idx: &Index, doc_id: &str) -> Result<(), String> {
        let key = format!("{}:{}", idx.collection, idx.name);
        let mut indexes = self.btree_indexes.write();

        if let Some(btree) = indexes.get_mut(&key) {
            let mut keys_to_remove = Vec::new();

            for (k, doc_ids) in btree.data.iter_mut() {
                if let Some(pos) = doc_ids.iter().position(|id| id == doc_id) {
                    doc_ids.remove(pos);
                    if doc_ids.is_empty() {
                        keys_to_remove.push(k.clone());
                    }
                }
            }

            for k in keys_to_remove {
                btree.data.remove(&k);
            }
        }

        Ok(())
    }

    fn query_btree(
        &self,
        idx: &Index,
        query: &HashMap<String, serde_json::Value>,
    ) -> Result<Vec<String>, String> {
        let key = format!("{}:{}", idx.collection, idx.name);
        let indexes = self.btree_indexes.read();

        if let Some(btree) = indexes.get(&key) {
            if let Some(value) = query.get("value") {
                let key_str = value.to_string();
                if let Some(doc_ids) = btree.data.get(&key_str) {
                    return Ok(doc_ids.clone());
                }
            }
        }

        Ok(Vec::new())
    }

    fn insert_gin(
        &self,
        idx: &Index,
        doc_id: &str,
        doc: &HashMap<String, serde_json::Value>,
    ) -> Result<(), String> {
        let key = format!("{}:{}", idx.collection, idx.name);
        let mut indexes = self.gin_indexes.write();

        if let Some(gin) = indexes.get_mut(&key) {
            let tokens = self.tokenize_json(doc);
            for token in tokens {
                gin.data
                    .entry(token)
                    .or_insert_with(Vec::new)
                    .push(doc_id.to_string());
            }
        }

        Ok(())
    }

    fn delete_gin(&self, idx: &Index, doc_id: &str) -> Result<(), String> {
        let key = format!("{}:{}", idx.collection, idx.name);
        let mut indexes = self.gin_indexes.write();

        if let Some(gin) = indexes.get_mut(&key) {
            let mut keys_to_remove = Vec::new();

            for (token, doc_ids) in gin.data.iter_mut() {
                if let Some(pos) = doc_ids.iter().position(|id| id == doc_id) {
                    doc_ids.remove(pos);
                    if doc_ids.is_empty() {
                        keys_to_remove.push(token.clone());
                    }
                }
            }

            for k in keys_to_remove {
                gin.data.remove(&k);
            }
        }

        Ok(())
    }

    fn query_gin(
        &self,
        idx: &Index,
        query: &HashMap<String, serde_json::Value>,
    ) -> Result<Vec<String>, String> {
        let key = format!("{}:{}", idx.collection, idx.name);
        let indexes = self.gin_indexes.read();

        if let Some(gin) = indexes.get(&key) {
            if let Some(token) = query.get("token").and_then(|v| v.as_str()) {
                if let Some(doc_ids) = gin.data.get(token) {
                    return Ok(doc_ids.clone());
                }
            }
        }

        Ok(Vec::new())
    }

    fn insert_hnsw(
        &self,
        idx: &Index,
        doc_id: &str,
        doc: &HashMap<String, serde_json::Value>,
    ) -> Result<(), String> {
        let key = format!("{}:{}", idx.collection, idx.name);
        let mut indexes = self.hnsw_indexes.write();

        if let Some(hnsw) = indexes.get_mut(&key) {
            if let Some(payload) = doc.get("payload").and_then(|v| v.as_object()) {
                if let Some(vector) = payload.get("vector").and_then(|v| v.as_array()) {
                    let vec: Vec<f32> = vector
                        .iter()
                        .filter_map(|v| v.as_f64().map(|f| f as f32))
                        .collect();

                    if !vec.is_empty() {
                        hnsw.vectors.insert(doc_id.to_string(), vec.clone());
                        let uuid = Uuid::parse_str(doc_id).unwrap_or_else(|_| Uuid::new_v4());
                        let _ = hnsw.index.add(uuid, vec);
                    }
                }
            }
        }

        Ok(())
    }

    fn delete_hnsw(&self, idx: &Index, doc_id: &str) -> Result<(), String> {
        let key = format!("{}:{}", idx.collection, idx.name);
        let mut indexes = self.hnsw_indexes.write();

        if let Some(hnsw) = indexes.get_mut(&key) {
            hnsw.vectors.remove(doc_id);
            let uuid = Uuid::parse_str(doc_id).unwrap_or_else(|_| Uuid::new_v4());
            let _ = hnsw.index.remove(uuid);
        }

        Ok(())
    }

    fn query_hnsw(
        &self,
        idx: &Index,
        query: &HashMap<String, serde_json::Value>,
    ) -> Result<Vec<String>, String> {
        let key = format!("{}:{}", idx.collection, idx.name);
        let indexes = self.hnsw_indexes.read();

        if let Some(hnsw) = indexes.get(&key) {
            if let Some(vector_query) = query.get("vector").and_then(|v| v.as_array()) {
                let query_vec: Vec<f32> = vector_query
                    .iter()
                    .filter_map(|v| v.as_f64().map(|f| f as f32))
                    .collect();

                let limit = query.get("limit").and_then(|v| v.as_u64()).unwrap_or(10) as usize;

                let results = hnsw.index.search(&query_vec, limit).unwrap_or_default();
                return Ok(results.iter().map(|u| u.to_string()).collect());
            }
        }

        Ok(Vec::new())
    }

    fn insert_tag(
        &self,
        idx: &Index,
        doc_id: &str,
        doc: &HashMap<String, serde_json::Value>,
    ) -> Result<(), String> {
        let key = format!("{}:{}", idx.collection, idx.name);
        let mut indexes = self.tag_indexes.write();

        if let Some(tag_idx) = indexes.get_mut(&key) {
            let uuid = Uuid::parse_str(doc_id).unwrap_or_else(|_| Uuid::new_v4());

            let timestamp = doc
                .get("_timestamp")
                .and_then(|v| v.as_i64())
                .unwrap_or_else(|| chrono::Utc::now().timestamp());

            let category = doc
                .get("category")
                .and_then(|v| v.as_str())
                .unwrap_or("GENERIC")
                .to_string();

            let semantic_vector = doc
                .get("payload")
                .and_then(|v| v.as_object())
                .and_then(|p| p.get("vector"))
                .and_then(|v| v.as_array())
                .map(|arr| {
                    arr.iter()
                        .filter_map(|v| v.as_f64().map(|f| f as f32))
                        .collect()
                });

            let tags: Vec<String> = doc
                .get("payload")
                .and_then(|v| v.as_object())
                .and_then(|p| p.get("tags"))
                .and_then(|v| v.as_array())
                .map(|arr| {
                    arr.iter()
                        .filter_map(|v| v.as_str().map(String::from))
                        .collect()
                })
                .unwrap_or_default();

            let block = TagBlock {
                id: uuid,
                timestamp,
                category,
                semantic_vector,
                tags: tags.clone(),
            };

            tag_idx.blocks.insert(uuid, block);

            for tag in tags {
                tag_idx
                    .tag_index
                    .entry(tag)
                    .or_insert_with(Vec::new)
                    .push(uuid);
            }
        }

        Ok(())
    }

    fn delete_tag(&self, idx: &Index, doc_id: &str) -> Result<(), String> {
        let key = format!("{}:{}", idx.collection, idx.name);
        let mut indexes = self.tag_indexes.write();

        if let Some(tag_idx) = indexes.get_mut(&key) {
            let uuid = Uuid::parse_str(doc_id).unwrap_or_else(|_| Uuid::new_v4());

            if let Some(block) = tag_idx.blocks.remove(&uuid) {
                for tag in &block.tags {
                    if let Some(ids) = tag_idx.tag_index.get_mut(tag) {
                        ids.retain(|&id| id != uuid);
                    }
                }
            }
        }

        Ok(())
    }

    fn query_tag(
        &self,
        idx: &Index,
        query: &HashMap<String, serde_json::Value>,
    ) -> Result<Vec<String>, String> {
        let key = format!("{}:{}", idx.collection, idx.name);
        let indexes = self.tag_indexes.read();

        if let Some(tag_idx) = indexes.get(&key) {
            let tags = query
                .get("tags")
                .and_then(|v| v.as_array())
                .map(|arr| {
                    arr.iter()
                        .filter_map(|v| v.as_str().map(String::from))
                        .collect()
                })
                .or_else(|| {
                    query
                        .get("tag")
                        .and_then(|v| v.as_str())
                        .map(|t| vec![t.to_string()])
                })
                .unwrap_or_default();

            if tags.is_empty() {
                return Err("invalid query for tag index".to_string());
            }

            let mut result_ids: Vec<Uuid> = Vec::new();

            for tag in &tags {
                if let Some(ids) = tag_idx.tag_index.get(tag) {
                    if result_ids.is_empty() {
                        result_ids = ids.clone();
                    } else {
                        result_ids.retain(|id| ids.contains(id));
                    }
                } else {
                    return Ok(Vec::new());
                }
            }

            return Ok(result_ids.iter().map(|u| u.to_string()).collect());
        }

        Ok(Vec::new())
    }

    fn build_composite_key(
        &self,
        fields: &[String],
        doc: &HashMap<String, serde_json::Value>,
    ) -> String {
        let parts: Vec<String> = fields
            .iter()
            .filter_map(|field| doc.get(field).map(|v| v.to_string()))
            .collect();
        parts.join("|")
    }

    fn tokenize_json(&self, doc: &HashMap<String, serde_json::Value>) -> Vec<String> {
        let mut tokens = Vec::new();
        let doc_value =
            serde_json::Value::Object(doc.iter().map(|(k, v)| (k.clone(), v.clone())).collect());
        self.tokenize_value(&doc_value, &mut tokens);
        tokens
    }

    fn tokenize_value(&self, value: &serde_json::Value, tokens: &mut Vec<String>) {
        match value {
            serde_json::Value::String(s) => {
                for word in s.to_lowercase().split_whitespace() {
                    if word.len() > 1 {
                        tokens.push(word.to_string());
                    }
                }
            }
            serde_json::Value::Object(map) => {
                for val in map.values() {
                    self.tokenize_value(val, tokens);
                }
            }
            serde_json::Value::Array(arr) => {
                for item in arr {
                    self.tokenize_value(item, tokens);
                }
            }
            _ => {}
        }
    }

    pub fn cosine_similarity(a: &[f64], b: &[f64]) -> f64 {
        if a.len() != b.len() {
            return 0.0;
        }

        let mut dot_product = 0.0_f64;
        let mut norm_a = 0.0_f64;
        let mut norm_b = 0.0_f64;

        for i in 0..a.len() {
            dot_product += a[i] * b[i];
            norm_a += a[i] * a[i];
            norm_b += b[i] * b[i];
        }

        if norm_a == 0.0 || norm_b == 0.0 {
            return 0.0;
        }

        dot_product / (norm_a * norm_b).sqrt()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_create_and_query_btree_index() {
        let manager = IndexManager::new("/tmp/test".to_string());

        manager
            .create_index(
                "test_collection",
                "by_email",
                IndexType::BTree,
                vec!["email".to_string()],
                true,
                None,
                None,
            )
            .unwrap();

        let mut doc = HashMap::new();
        doc.insert("id".to_string(), serde_json::json!("doc1"));
        doc.insert("email".to_string(), serde_json::json!("test@example.com"));

        manager.insert("test_collection", &doc).unwrap();

        let mut query = HashMap::new();
        query.insert("value".to_string(), serde_json::json!("test@example.com"));

        let results = manager
            .query_index("test_collection", "by_email", &query)
            .unwrap();

        assert_eq!(results, vec!["doc1"]);
    }

    #[test]
    fn test_hnsw_index() {
        let manager = IndexManager::new("/tmp/test".to_string());

        let mut options = HashMap::new();
        options.insert("dimensions".to_string(), serde_json::json!(4));
        options.insert("m".to_string(), serde_json::json!(16));

        manager
            .create_index(
                "test_collection",
                "vector",
                IndexType::HNSW,
                vec!["vector".to_string()],
                false,
                None,
                Some(options),
            )
            .unwrap();

        let mut doc = HashMap::new();
        doc.insert("id".to_string(), serde_json::json!("vec1"));
        let payload = serde_json::json!({
            "vector": vec![1.0, 0.0, 0.0, 0.0]
        });
        doc.insert("payload".to_string(), payload);

        manager.insert("test_collection", &doc).unwrap();

        let mut query = HashMap::new();
        query.insert(
            "vector".to_string(),
            serde_json::json!(vec![1.0, 0.0, 0.0, 0.0]),
        );
        query.insert("limit".to_string(), serde_json::json!(1));

        let results = manager
            .query_index("test_collection", "vector", &query)
            .unwrap();

        assert!(!results.is_empty());
    }
}
