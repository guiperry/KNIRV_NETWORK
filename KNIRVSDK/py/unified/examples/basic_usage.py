#!/usr/bin/env python3
"""
Basic usage example for the KNIRV Unified SDK.

This example demonstrates how to use the unified SDK to interact with
all KNIRV Network services through a single client interface.
"""

import asyncio
import os
from knirv_sdk import KNIRVClient, ClientConfig


async def main():
    """Main example function."""
    print("🚀 KNIRV Unified SDK Example")
    print("=" * 40)
    
    # Create client with environment-based configuration
    config = ClientConfig.for_development()
    
    async with KNIRVClient(config) as client:
        print(f"✅ Client created: {client}")
        print(f"📊 Environment: {client.config.environment}")
        print()
        
        # Example 1: Gateway Service - Economics
        print("🏦 Gateway Service - Economics")
        print("-" * 30)
        
        try:
            # Note: These would normally make real API calls
            # For demo purposes, they'll fail with connection errors
            print("📋 Listing skills...")
            # skills = await client.gateway.economics.skills.list(per_page=5)
            # print(f"Found {len(skills.get('data', []))} skills")
            print("   (Would list available skills)")
            
            print("💰 Getting LLM usage...")
            # usage = await client.gateway.economics.llm.get_usage()
            # print(f"Current usage: {usage}")
            print("   (Would show LLM usage statistics)")
            
        except Exception as e:
            print(f"   ⚠️  Demo mode - API calls would work with real endpoints")
        
        print()
        
        # Example 2: Transaction Service
        print("⛓️  Transaction Service")
        print("-" * 30)
        
        try:
            print("📊 Getting chain info...")
            # chain_info = await client.transaction.get_chain_info()
            # print(f"Chain height: {chain_info.get('height', 'N/A')}")
            print("   (Would show blockchain information)")
            
            print("🔗 Getting latest block...")
            # latest_block = await client.transaction.get_latest_block()
            # print(f"Latest block: {latest_block.get('hash', 'N/A')}")
            print("   (Would show latest block information)")
            
        except Exception as e:
            print(f"   ⚠️  Demo mode - API calls would work with real endpoints")
        
        print()
        
        # Example 3: Transmission Service
        print("📡 Transmission Service")
        print("-" * 30)
        
        try:
            # URI parsing works without API calls
            print("🔗 Parsing KNIRV URI...")
            uri = client.transmission.parse_uri("knirv://skill123/execute?param=value")
            print(f"   ID: {uri.id}")
            print(f"   Resource Type: {uri.resource_type}")
            print(f"   Path: {uri.path}")
            print(f"   Query Param: {uri.get_query_param('param')}")
            
            print("🚀 Generating URI...")
            # new_uri = await client.transmission.generate_uri({
            #     "resource_type": "skill",
            #     "id": "test-skill",
            #     "action": "execute"
            # })
            # print(f"Generated: {new_uri}")
            print("   (Would generate a new KNIRV URI)")
            
        except Exception as e:
            print(f"   ⚠️  Demo mode - some API calls would work with real endpoints")
        
        print()
        
        # Example 4: Configuration
        print("⚙️  Configuration")
        print("-" * 30)
        print(f"🌐 Gateway URL: {client.config.gateway.base_url}")
        print(f"⛓️  Transaction URL: {client.config.transaction.base_url}")
        print(f"📡 Transmission URL: {client.config.transmission.base_url}")
        print(f"🔑 API Key: {'***' if client.config.api_key else 'Not set'}")
        print(f"⏱️  Timeout: {client.config.timeout}s")
        print(f"🔄 Retries: {client.config.retries}")
        
        print()
        print("✅ Example completed successfully!")


def sync_example():
    """Example of synchronous usage (not recommended for production)."""
    print("\n🔄 Synchronous Usage Example")
    print("-" * 30)
    
    # Create client for testing
    config = ClientConfig.for_testing()
    
    with KNIRVClient(config) as client:
        print(f"📱 Client: {client}")
        print("   Note: Use async context manager for production code")
        
        # Parse URI (synchronous operation)
        uri = client.transmission.parse_uri("knirv://test123/action")
        print(f"🔗 Parsed URI: {uri.id} -> {uri.resource_type}")


if __name__ == "__main__":
    # Run async example
    asyncio.run(main())
    
    # Run sync example
    sync_example()
    
    print("\n🎉 All examples completed!")
    print("\n💡 Tips:")
    print("   - Set KNIRV_API_KEY environment variable for authentication")
    print("   - Set KNIRV_ENVIRONMENT=development for development mode")
    print("   - Use async context managers for proper resource cleanup")
    print("   - Check the README.md for more detailed usage examples")
