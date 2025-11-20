# Regenerate your Go Protobuf files

cd /home/gperry/Documents/GitHub/cloud-equities/_GO_ROOT_MCP

# Remove old generated files first (good practice)

rm -f *.pb.go # Or more specific paths if they are in subdirectories
rm -f proto/*.pb.go # If you have them in a proto subdirectory as well

# Regenerate (adjust if your .proto files are in different locations or you have more)

protoc --proto_path=. --go_out=. --go_opt=paths=source_relative ./proto/*.proto

# Another version enabling grpc
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       ./proto/*.proto
