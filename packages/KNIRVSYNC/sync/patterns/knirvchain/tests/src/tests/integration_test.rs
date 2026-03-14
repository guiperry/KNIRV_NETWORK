// tests/integration_test.rs
use knirvchain::nrn_token::Address; // Import necessary types
use num_bigint::BigInt;
use reqwest;
use serde_json::json;
use std::env;
use std::process::Child;
use std::thread;
use std::time::Duration;

// Helper to start your blockchain in a separate process for testing.
fn start_blockchain() -> Child {
    std::process::Command::new("cargo")
        .arg("run")
        .spawn()
        .expect("Failed to start blockchain process")
}

#[tokio::test]
async fn test_api_calls() {
    // Set necessary environment variables for the test (if needed).
    //env::set_var("NRN_OWNER_PRIVATE_KEY", "..."); // Set private key here
    env::set_var("RUST_LOG", "info,knirvchain=debug");
    // Start the blockchain in a separate thread/process
    let mut child = start_blockchain();

    // Give the server some time to start up
    thread::sleep(Duration::from_secs(3));

    // Base URL for your API
    let base_url = "http://127.0.0.1:8000"; // Adjust if needed
    let client = reqwest::Client::new();

    // Example: Generate Key API Request
    let owner_private_key = format!("{:x}", SigningKey::random(&mut OsRng).to_bytes());
    let owner_address = get_address_from_private_key(&owner_private_key);
    // Example: Mint Token API request
    let mint_request = json!({
        "to_address": format!("{}",owner_address.unwrap()), // Replace with a valid address string
        "amount": "500",
        "owner_private_key": owner_private_key.clone(),
    });
    let mint_response = client
        .post(format!("{}/mint_token", base_url))
        .json(&mint_request)
        .send()
        .await
        .expect("Failed to send mint request");
    assert!(mint_response.status().is_success());

    // Example: Transfer Token API request
    let transfer_request = json!({
        "to_address": "0x...", // Replace with a valid recipient address
        "amount": "100",
        "from_private_key": owner_private_key,
    });
    let transfer_response = client
        .post(format!("{}/transfer_token", base_url))
        .json(&transfer_request)
        .send()
        .await
        .expect("Failed to send transfer request");
    // ... Assertions ...

    //Stop the blockchain (kill the child process when done)
    child.kill().expect("Failed to kill test process");
}

// Copy your get_address_from_private_key() function from nrn_token.rs here to reuse the test code in `main()`
fn get_address_from_private_key(private_key: &str) -> Result<Address> {
    let private_key_bytes =
        hex::decode(private_key).map_err(|e| anyhow!("Failed to decode private key: {}", e))?;
    let signing_key = SigningKey::from_bytes(private_key_bytes.as_slice().into())
        .map_err(|e| anyhow!("Failed to parse private key: {}", e))?;
    let public_key = signing_key.verifying_key();
    let public_key_bytes = public_key.to_encoded_point(false);
    Ok(get_address_from_public_key(&public_key_bytes)) // Pass the EncodedPoint
}

//Copy the get_address_from_public_key() as well.
fn get_address_from_public_key(public_key_bytes: &EncodedPoint<k256::Secp256k1>) -> Address {
    // Specify the generic type
    let mut hasher = Keccak256::new();
    hasher.update(public_key_bytes.as_bytes());
    let hash = hasher.finalize();
    let mut address_bytes = [0u8; 20];
    address_bytes.copy_from_slice(&hash[12..]);
    Address(address_bytes)
}
