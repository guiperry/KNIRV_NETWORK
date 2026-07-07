#!/usr/bin/env python3
"""
Example script for fetching a resource using the KNIRV Client SDK.
"""

import asyncio
import sys
from knirv_uri_sdk import KnirvClient, parse_knirv_uri, KnirvURIError, KnirvClientError


async def main():
    # Parse command line arguments
    if len(sys.argv) < 2:
        print("Usage: fetch_resource.py <knirv-uri>")
        print("Example: fetch_resource.py knirv://mychain-alpha.chain/block?number=123")
        return 1
    
    uri_string = sys.argv[1]
    
    # Parse the URI to validate it
    try:
        parsed_uri = parse_knirv_uri(uri_string)
        
        print("Parsed URI:")
        print(f"  ID:           {parsed_uri.id}")
        print(f"  ResourceType: {parsed_uri.resource_type}")
        print(f"  Path:         {parsed_uri.path}")
        print(f"  Raw URI:      {parsed_uri.raw}")
        print("  Query Parameters:")
        for key, values in parsed_uri.query.items():
            for value in values:
                print(f"    {key}: {value}")
    except KnirvURIError as e:
        print(f"Failed to parse URI: {e}")
        return 1
    
    # Create a client with default configuration
    client = KnirvClient(
        bootstrap_peers=[
            "/ip4/104.131.131.82/tcp/4001/p2p/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
            "/ip4/104.236.179.241/tcp/4001/p2p/QmSoLPppuBtQSGwKDZT2M73ULpjvfd3aZ6ha4oFGL1KrGM",
            "/ip4/128.199.219.111/tcp/4001/p2p/QmSoLSafTMBsPKadTEgaXctDQVcqN88CNLHXMkTNwMKPnu",
            "/ip4/104.236.76.40/tcp/4001/p2p/QmSoLV4Bbm51jM9C4gDYZQ9Cy3U6aXMJDAbzgu2fzaDs64",
        ],
        log_enabled=True
    )
    
    try:
        # Bootstrap the client
        print("Bootstrapping client...")
        await client.bootstrap()
        
        # Fetch the resource
        print(f"Fetching resource: {uri_string}")
        resource = await client.fetch_resource(uri_string)
        
        # Print the resource data
        print("Resource data:")
        print(str(resource))
        
        return 0
    except KnirvClientError as e:
        print(f"Failed to fetch resource: {e}")
        return 1
    finally:
        # Close the client
        await client.close()


if __name__ == "__main__":
    sys.exit(asyncio.run(main()))