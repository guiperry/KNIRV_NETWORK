use crate::collection::DistributedCollection;
use crate::types::EntryType;
use std::collections::HashMap;

#[derive(Debug, Clone)]
pub struct Query {
    pub query_type: QueryType,
    pub entry_type: Option<EntryType>,
    pub collection: String,
    pub id: Option<String>,
    pub key: Option<String>,
    pub value: Option<String>,
    pub filters: Vec<Filter>,
    pub similar_to: Vec<f64>,
    pub limit: usize,
    pub index_name: Option<String>,
    pub fields: Vec<String>,
    pub unique: bool,
    pub modality_type: Option<String>,
}

#[derive(Debug, Clone, PartialEq)]
pub enum QueryType {
    Get,
    Set,
    Delete,
    CreateIndex,
    CreateCollection,
    DropIndex,
    DropCollection,
    GetModality,
}

#[derive(Debug, Clone)]
pub struct Filter {
    pub key: String,
    pub operator: String,
    pub value: serde_json::Value,
}

pub struct KNIRVQLParser;

impl KNIRVQLParser {
    pub fn new() -> Self {
        Self
    }

    pub fn parse(&self, query: &str) -> Result<Query, String> {
        let query = query.trim();
        let parts: Vec<&str> = query.split_whitespace().collect();
        
        if parts.is_empty() {
            return Err("empty query".to_string());
        }

        let cmd = parts[0].to_uppercase();
        match cmd.as_str() {
            "GET" => self.parse_get(&parts[1..]),
            "SET" => self.parse_set(&parts[1..]),
            "DELETE" => self.parse_delete(&parts[1..]),
            "CREATE" => self.parse_create(&parts[1..]),
            "DROP" => self.parse_drop(&parts[1..]),
            _ => Err(format!("unknown command: {}", cmd)),
        }
    }

    fn parse_get(&self, parts: &[&str]) -> Result<Query, String> {
        if parts.len() < 2 {
            return Err("invalid GET query".to_string());
        }

        let mut entry_type_str = parts[0].to_uppercase();
        let mut modality_type = None;

        if entry_type_str.starts_with("MEMORY.MODALITY(") && entry_type_str.ends_with(")") {
            let start = "MEMORY.MODALITY(".len();
            let end = entry_type_str.len() - 1;
            modality_type = Some(entry_type_str[start..end].to_string());
            entry_type_str = "MEMORY".to_string();
        }

        let entry_type = match entry_type_str.as_str() {
            "MEMORY" => Some(EntryType::Memory),
            "AUTH" => Some(EntryType::Auth),
            _ => None,
        };

        let mut filters = Vec::new();
        let mut similar_to = Vec::new();
        let mut limit = 0;
        let mut i = 1;

        if i < parts.len() && parts[i].to_uppercase() == "WHERE" {
            i += 1;
            
            while i < parts.len() {
                let next = parts[i].to_uppercase();
                
                if next == "SIMILAR" && i + 2 < parts.len() && parts[i + 1].to_uppercase() == "TO" {
                    i += 2;
                    if i < parts.len() {
                        let vec_str = parts[i].trim_start_matches('[').trim_end_matches(']');
                        for vp in vec_str.split(',') {
                            if let Ok(v) = vp.trim().parse::<f64>() {
                                similar_to.push(v);
                            }
                        }
                    }
                    break;
                } else if next == "WITH" && i + 2 < parts.len() && parts[i + 1].to_uppercase() == "TAGS" {
                    i += 2;
                    if i < parts.len() {
                        let tag_str = parts[i].trim_start_matches('(').trim_end_matches(')');
                        for t in tag_str.split(',') {
                            filters.push(Filter {
                                key: "tags".to_string(),
                                operator: "CONTAINS".to_string(),
                                value: serde_json::json!(t.trim()),
                            });
                        }
                        i += 1;
                    }
                } else if next == "LIMIT" {
                    i += 1;
                    if i < parts.len() {
                        if let Ok(l) = parts[i].parse::<usize>() {
                            limit = l;
                        }
                        i += 1;
                    }
                } else {
                    if i + 2 < parts.len() {
                        let key = parts[i].to_string();
                        let operator = parts[i + 1].to_string();
                        let value_str = parts[i + 2].trim_matches('"');
                        
                        let value: serde_json::Value = value_str.parse::<f64>()
                            .map(serde_json::Value::from)
                            .unwrap_or_else(|_| serde_json::Value::String(value_str.to_string()));

                        filters.push(Filter { key, operator, value });
                        i += 3;
                    } else {
                        break;
                    }
                }
            }
        }

        Ok(Query {
            query_type: QueryType::Get,
            entry_type,
            collection: String::new(),
            id: None,
            key: None,
            value: None,
            filters,
            similar_to,
            limit,
            index_name: None,
            fields: Vec::new(),
            unique: false,
            modality_type,
        })
    }

    fn parse_set(&self, parts: &[&str]) -> Result<Query, String> {
        if parts.len() < 3 || parts[1] != "=" {
            return Err("invalid SET query".to_string());
        }

        let key = parts[0].to_string();
        let value = parts[2..].join(" ").trim_matches('"').to_string();

        Ok(Query {
            query_type: QueryType::Set,
            entry_type: Some(EntryType::Auth),
            collection: String::new(),
            id: None,
            key: Some(key),
            value: Some(value),
            filters: Vec::new(),
            similar_to: Vec::new(),
            limit: 0,
            index_name: None,
            fields: Vec::new(),
            unique: false,
            modality_type: None,
        })
    }

    fn parse_delete(&self, parts: &[&str]) -> Result<Query, String> {
        if parts.len() < 4 || parts[0].to_uppercase() != "WHERE" 
            || parts[1].to_uppercase() != "ID" || parts[2] != "=" {
            return Err("invalid DELETE query".to_string());
        }

        Ok(Query {
            query_type: QueryType::Delete,
            entry_type: None,
            collection: String::new(),
            id: Some(parts[3].to_string()),
            key: None,
            value: None,
            filters: Vec::new(),
            similar_to: Vec::new(),
            limit: 0,
            index_name: None,
            fields: Vec::new(),
            unique: false,
            modality_type: None,
        })
    }

    fn parse_create(&self, parts: &[&str]) -> Result<Query, String> {
        if parts.is_empty() {
            return Err("invalid CREATE command".to_string());
        }

        let sub_cmd = parts[0].to_uppercase();
        match sub_cmd.as_str() {
            "INDEX" => self.parse_create_index(&parts[1..]),
            "COLLECTION" => self.parse_create_collection(&parts[1..]),
            _ => Err(format!("unknown CREATE command: {}", sub_cmd)),
        }
    }

    fn parse_create_index(&self, parts: &[&str]) -> Result<Query, String> {
        if parts.len() < 3 {
            return Err("invalid CREATE INDEX command".to_string());
        }

        let index_ref = parts[0];
        let index_parts: Vec<&str> = index_ref.split(':').collect();
        
        if index_parts.len() != 2 {
            return Err("invalid index reference, expected collection:index".to_string());
        }

        let collection = index_parts[0].to_string();
        let index_name = index_parts[1].to_string();

        if parts[1] != "ON" {
            return Err("expected ON after index name".to_string());
        }

        if parts[2] != collection {
            return Err("collection mismatch in index definition".to_string());
        }

        let mut fields = Vec::new();
        let mut i = 3;
        let mut unique = false;

        if i < parts.len() && parts[i].starts_with('(') {
            let field_str = parts[i].trim_start_matches('(').trim_end_matches(')');
            if !field_str.is_empty() {
                for f in field_str.split(',') {
                    let field = f.trim().to_string();
                    if !field.is_empty() {
                        fields.push(field);
                    }
                }
            }
            i += 1;
        }

        if i < parts.len() && parts[i].to_uppercase() == "UNIQUE" {
            unique = true;
        }

        Ok(Query {
            query_type: QueryType::CreateIndex,
            entry_type: None,
            collection,
            id: None,
            key: None,
            value: None,
            filters: Vec::new(),
            similar_to: Vec::new(),
            limit: 0,
            index_name: Some(index_name),
            fields,
            unique,
            modality_type: None,
        })
    }

    fn parse_create_collection(&self, parts: &[&str]) -> Result<Query, String> {
        if parts.is_empty() {
            return Err("invalid CREATE COLLECTION command".to_string());
        }

        Ok(Query {
            query_type: QueryType::CreateCollection,
            entry_type: None,
            collection: parts[0].to_string(),
            id: None,
            key: None,
            value: None,
            filters: Vec::new(),
            similar_to: Vec::new(),
            limit: 0,
            index_name: None,
            fields: Vec::new(),
            unique: false,
            modality_type: None,
        })
    }

    fn parse_drop(&self, parts: &[&str]) -> Result<Query, String> {
        if parts.len() < 2 {
            return Err("invalid DROP command".to_string());
        }

        let sub_cmd = parts[0].to_uppercase();
        match sub_cmd.as_str() {
            "INDEX" => self.parse_drop_index(&parts[1..]),
            "COLLECTION" => self.parse_drop_collection(&parts[1..]),
            _ => Err(format!("unknown DROP command: {}", sub_cmd)),
        }
    }

    fn parse_drop_index(&self, parts: &[&str]) -> Result<Query, String> {
        if parts.is_empty() {
            return Err("invalid DROP INDEX command".to_string());
        }

        let index_ref = parts[0];
        let index_parts: Vec<&str> = index_ref.split(':').collect();
        
        if index_parts.len() != 2 {
            return Err("invalid index reference, expected collection:index".to_string());
        }

        Ok(Query {
            query_type: QueryType::DropIndex,
            entry_type: None,
            collection: index_parts[0].to_string(),
            id: None,
            key: None,
            value: None,
            filters: Vec::new(),
            similar_to: Vec::new(),
            limit: 0,
            index_name: Some(index_parts[1].to_string()),
            fields: Vec::new(),
            unique: false,
            modality_type: None,
        })
    }

    fn parse_drop_collection(&self, parts: &[&str]) -> Result<Query, String> {
        if parts.is_empty() {
            return Err("invalid DROP COLLECTION command".to_string());
        }

        Ok(Query {
            query_type: QueryType::DropCollection,
            entry_type: None,
            collection: parts[0].to_string(),
            id: None,
            key: None,
            value: None,
            filters: Vec::new(),
            similar_to: Vec::new(),
            limit: 0,
            index_name: None,
            fields: Vec::new(),
            unique: false,
            modality_type: None,
        })
    }
}

impl Query {
    pub async fn execute(
        &self,
        collection: &DistributedCollection,
    ) -> Result<serde_json::Value, String> {
        match self.query_type {
            QueryType::Get => self.execute_get(collection).await,
            QueryType::Set => {
                let doc = serde_json::json!({
                    "id": self.key.as_ref().unwrap(),
                    "entryType": "AUTH",
                    "payload": {
                        "key": self.key.as_ref().unwrap(),
                        "value": self.value.as_ref().unwrap(),
                    }
                });
                let doc_map = serde_json::from_value(doc).map_err(|e| e.to_string())?;
                collection.insert("", doc_map).await
                    .map(|_| serde_json::Value::Null)
                    .map_err(|e| e.to_string())
            }
            QueryType::Delete => {
                if let Some(id) = &self.id {
                    collection.delete(id).await
                        .map_err(|e| e.to_string())?;
                }
                Ok(serde_json::Value::Null)
            }
            _ => Err("unsupported query type".to_string()),
        }
    }

    async fn execute_get(&self, collection: &DistributedCollection) -> Result<serde_json::Value, String> {
        let docs = collection.find_all().await
            .map_err(|e| e.to_string())?;

        let mut results: Vec<serde_json::Value> = Vec::new();
        for doc in docs.into_iter() {
            if self.matches_filters(&doc) {
                results.push(serde_json::Value::Object(doc.into_iter().collect()));
            }
        }

        if self.limit > 0 && results.len() > self.limit {
            results.truncate(self.limit);
        }

        Ok(serde_json::to_value(results).map_err(|e| e.to_string())?)
    }

    fn matches_filters(&self, doc: &HashMap<String, serde_json::Value>) -> bool {
        let payload = match doc.get("payload") {
            Some(serde_json::Value::Object(p)) => p,
            _ => return true,
        };

        for filter in &self.filters {
            if let Some(val) = payload.get(&filter.key) {
                if !self.matches_filter(val, filter) {
                    return false;
                }
            } else {
                return false;
            }
        }

        true
    }

    fn matches_filter(&self, value: &serde_json::Value, filter: &Filter) -> bool {
        match filter.operator.as_str() {
            "=" => {
                let val_str = value.to_string();
                let filter_str = filter.value.to_string();
                val_str.trim_matches('"') == filter_str.trim_matches('"')
            }
            "!=" => {
                let val_str = value.to_string();
                let filter_str = filter.value.to_string();
                val_str.trim_matches('"') != filter_str.trim_matches('"')
            }
            ">" | "<" | ">=" | "<=" => {
                let val_num = value.as_f64().or_else(|| value.as_i64().map(|v| v as f64));
                let filter_num = filter.value.as_f64().or_else(|| filter.value.as_i64().map(|v| v as f64));
                
                if let (Some(val_num), Some(filter_num)) = (val_num, filter_num) {
                    match filter.operator.as_str() {
                        ">" => val_num > filter_num,
                        "<" => val_num < filter_num,
                        ">=" => val_num >= filter_num,
                        "<=" => val_num <= filter_num,
                        _ => false,
                    }
                } else {
                    false
                }
            }
            "CONTAINS" => {
                if let (Some(val_str), Some(filter_str)) = (value.as_str(), filter.value.as_str()) {
                    val_str.contains(filter_str)
                } else {
                    false
                }
            }
            "STARTS_WITH" => {
                if let (Some(val_str), Some(filter_str)) = (value.as_str(), filter.value.as_str()) {
                    val_str.starts_with(filter_str)
                } else {
                    false
                }
            }
            _ => false,
        }
    }
}

impl Default for KNIRVQLParser {
    fn default() -> Self {
        Self::new()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_parse_get_query() {
        let parser = KNIRVQLParser::new();
        
        let query = parser.parse("GET MEMORY WHERE source = \"web-scrape\" LIMIT 10").unwrap();
        assert_eq!(query.query_type, QueryType::Get);
        assert_eq!(query.limit, 10);
        assert_eq!(query.filters.len(), 1);
    }

    #[test]
    fn test_parse_set_query() {
        let parser = KNIRVQLParser::new();
        
        let query = parser.parse("SET mykey = \"myvalue\"").unwrap();
        assert_eq!(query.query_type, QueryType::Set);
        assert_eq!(query.key, Some("mykey".to_string()));
        assert_eq!(query.value, Some("myvalue".to_string()));
    }

    #[test]
    fn test_parse_create_index() {
        let parser = KNIRVQLParser::new();
        
        let query = parser.parse("CREATE INDEX memory:by_email ON memory (email) UNIQUE").unwrap();
        assert_eq!(query.query_type, QueryType::CreateIndex);
        assert_eq!(query.collection, "memory");
        assert_eq!(query.index_name, Some("by_email".to_string()));
        assert!(query.unique);
    }

    #[test]
    fn test_parse_delete_query() {
        let parser = KNIRVQLParser::new();
        
        let query = parser.parse("DELETE WHERE id = doc123").unwrap();
        assert_eq!(query.query_type, QueryType::Delete);
        assert_eq!(query.id, Some("doc123".to_string()));
    }
}
