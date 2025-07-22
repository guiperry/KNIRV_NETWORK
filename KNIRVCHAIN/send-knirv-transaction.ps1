# Configuration

#If the environment variable KNIRVCHAIN_RPC_ENDPOINT is not set, set it to a default value.
$rpcEndpoint = if ($env:KNIRVCHAIN_RPC_ENDPOINT) { $env:KNIRVCHAIN_RPC_ENDPOINT } else { "http://localhost:8080" }


# Create a transaction object as a hashtable
$transaction = @{
    data = "test transaction from powershell";
    signature = "mocked signature from powershell with a different signature";
}

# Convert hashtable to JSON
$jsonPayload = $transaction | ConvertTo-Json

# Set up headers for the HTTP request.
$headers = @{
    "Content-Type" = "application/json"
}

# Send the HTTP POST request using Invoke-WebRequest.
try {
    $response = Invoke-WebRequest -Uri "$rpcEndpoint/send_txn" -Method Post -Body $jsonPayload -Headers $headers -UseBasicParsing
    Write-Host "Request to server successful."
   # Convert the JSON response to a Powershell object, and parse the message field
    $responseObject = $response.Content | ConvertFrom-Json;
    # Display the hash
    Write-Host "Transaction Hash: $($responseObject.transaction_hash)"
    Write-Host "Response Message: $($responseObject.message)"

} catch {
   Write-Host "Error sending request:"
   Write-Host $_.Exception.Message

}