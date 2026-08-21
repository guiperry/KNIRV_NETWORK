//! Command-line entrypoint for inspecting or materializing embedded modules.

use std::{env, path::Path};

use knirv_sdk::{materialize_wasm_module, WasmModule};

fn main() {
    if let Err(message) = run(env::args().skip(1)) {
        eprintln!("{message}");
        std::process::exit(2);
    }
}

fn run(mut args: impl Iterator<Item = String>) -> Result<(), String> {
    match args.next().as_deref() {
        Some("list") => {
            for module in WasmModule::ALL {
                println!(
                    "{}\t{} bytes\t{}\t{}",
                    module.id(),
                    module.bytes().len(),
                    module.sha256(),
                    module.description()
                );
            }
            Ok(())
        }
        Some("extract") => {
            let module = args.next().ok_or_else(usage)?;
            let destination = args.next().ok_or_else(usage)?;
            if args.next().is_some() {
                return Err(usage());
            }
            materialize_wasm_module(&module, Path::new(&destination))
                .map_err(|error| format!("could not extract {module} to {destination}: {error}"))
        }
        Some("verify") => {
            let module = args.next().ok_or_else(usage)?;
            let path = args.next().ok_or_else(usage)?;
            if args.next().is_some() {
                return Err(usage());
            }
            let module = WasmModule::parse(&module)
                .ok_or_else(|| format!("unknown KNIRV WASM module: {module}"))?;
            let bytes =
                std::fs::read(&path).map_err(|error| format!("could not read {path}: {error}"))?;
            module
                .verify(&bytes)
                .map_err(|error| format!("module verification failed: {error}"))?;
            println!("verified {}", module.id());
            Ok(())
        }
        _ => Err(usage()),
    }
}

fn usage() -> String {
    "Usage: knirv-sdk list | knirv-sdk extract <module> <destination> | knirv-sdk verify <module> <path>".into()
}
