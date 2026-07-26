use knirvbase::prelude::*;
use std::collections::HashMap;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error + Send + Sync>> {
    println!("KNIRVBASE Distributed Database Starting...");

    let app_data_dir = std::env::var("XDG_DATA_HOME")
        .unwrap_or_else(|_| {
            let home = std::env::var("HOME").unwrap();
            format!("{}/.local/share/knirvbase", home)
        });

    std::fs::create_dir_all(&app_data_dir)?;

    let opts = knirvbase::Options {
        data_dir: app_data_dir,
        distributed_enabled: true,
        distributed_network_id: "".to_string(),
        distributed_bootstrap_peers: vec![],
    };

    let db = knirvbase::DB::new(opts).await?;

    let network_id = db.create_network(NetworkConfig {
        network_id: "consortium-1".to_string(),
        name: "Consortium 1".to_string(),
        collections: HashMap::new(),
        bootstrap_peers: vec![],
        default_posting_network: "".to_string(),
        auto_post_classifications: vec![],
        private_by_default: true,
        encryption: Default::default(),
        replication: Default::default(),
        discovery: Default::default(),
    }).await?;

    let mut auth_coll = db.collection("auth").await?;
    let mut memory_coll = db.collection("memory").await?;

    auth_coll.attach_to_network(&network_id).await?;
    memory_coll.attach_to_network(&network_id).await?;

    println!("KNIRVBASE Distributed Database Started");

    let mut doc = HashMap::new();
    doc.insert("id".to_string(), serde_json::json!("mem1"));
    doc.insert("entryType".to_string(), serde_json::json!("MEMORY"));
    let mut payload = HashMap::new();
    payload.insert("source".to_string(), serde_json::json!("web-scrape"));
    payload.insert("data".to_string(), serde_json::json!("some data"));
    payload.insert("vector".to_string(), serde_json::json!([0.45, 0.12]));
    doc.insert("payload".to_string(), serde_json::json!(payload));

    memory_coll.insert(doc).await?;
    println!("Inserted memory entry");

    let all_docs = memory_coll.find_all().await?;
    println!("Memory results: {:?}", all_docs);

    println!("KNIRVBASE running. Press Ctrl+C to exit.");

    tokio::signal::ctrl_c().await?;
    db.shutdown().await?;

    Ok(())
}
