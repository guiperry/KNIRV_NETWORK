#!/bin/bash

# go back
cd ../
# remove the file
rm -rf 5001

# Define the file path
file_path="constants/constants.go"

# Use sed to search and replace the value
sed -i 's/\(BLOCKCHAIN_DB_PATH\s*=\s*"\)[^\/]*\/knirvchain_peer_5001_db"/\15001\/knirvchain_peer_5001_db"/' "$file_path"

# run the file
go run start.go chain -port 5001 -miners_address knirvchain4c5756faf0c45cc4d1a32e47def1485d0a87f0bb9b8052 -remote_node http://127.0.0.1:5001