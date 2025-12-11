use std::env;
use std::path::PathBuf;

fn main() -> Result<(), Box<dyn std::error::Error>> {
    let out_dir = PathBuf::from(env::var("OUT_DIR").unwrap());
    
    prost_build::Config::new()
        .out_dir(&out_dir)
        .type_attribute(".", "#[derive(serde::Serialize, serde::Deserialize)]")
        .compile_protos(&["proto/cortex.proto"], &["proto/"])?;
    
    println!("cargo:rerun-if-changed=proto/cortex.proto");
    
    Ok(())
}
