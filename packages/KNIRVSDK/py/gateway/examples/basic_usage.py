"""
Basic usage example for the KNIRV Gateway SDK
"""

import asyncio
import os
from knirv_gateway_sdk import KNIRVGateway
from knirv_gateway_sdk.types import LLMRequest, ValidationRequest


async def main():
    """Main example function."""
    # Initialize the client
    client = KNIRVGateway(
        base_url=os.getenv("KNIRV_GATEWAY_BASE_URL", "https://gateway.knirv.com"),
        api_key=os.getenv("KNIRV_GATEWAY_API_KEY", "your-api-key")
    )
    
    try:
        print("=== KNIRV Gateway SDK Example ===\n")
        
        # Health check
        print("1. Checking service health...")
        health = await client.health.check()
        print(f"   Service: {health.service}")
        print(f"   Status: {health.status}")
        print(f"   Uptime: {health.uptime} seconds\n")
        
        # Economics - List skills
        print("2. Listing available skills...")
        skills = await client.economics.skills.list(per_page=5)
        print(f"   Found {len(skills)} skills:")
        for skill in skills:
            print(f"   - {skill.name} ({skill.category}): {skill.cost}")
        print()
        
        # Economics - LLM example
        print("3. Using LLM service...")
        llm_request = LLMRequest(
            prompt="What is the KNIRV network?",
            model="gpt-3.5-turbo",
            max_tokens=100,
            temperature=0.7
        )
        try:
            llm_response = await client.economics.llm.generate(llm_request)
            print(f"   Response: {llm_response.response}")
            print(f"   Tokens used: {llm_response.tokens_used}")
            print(f"   Cost: {llm_response.cost}\n")
        except Exception as e:
            print(f"   LLM service error: {e}\n")
        
        # Economics - Validation example
        print("4. Using validation service...")
        validation_request = ValidationRequest(
            data={"username": "alice", "email": "alice@example.com"},
            rules=["required", "email_format"]
        )
        try:
            validation_result = await client.economics.validation.validate(validation_request)
            print(f"   Valid: {validation_result.valid}")
            if validation_result.errors:
                print(f"   Errors: {validation_result.errors}")
            print()
        except Exception as e:
            print(f"   Validation service error: {e}\n")
        
        # Gateway - List routes
        print("5. Listing gateway routes...")
        try:
            routes = await client.gateway.routes.list()
            print(f"   Found {len(routes)} routes:")
            for route in routes[:3]:  # Show first 3
                print(f"   - {route.method} {route.path} -> {route.handler}")
            print()
        except Exception as e:
            print(f"   Gateway routes error: {e}\n")
        
        # Integration - List integrations
        print("6. Listing integrations...")
        try:
            integrations = await client.integration.list(per_page=5)
            print(f"   Found {len(integrations)} integrations:")
            for integration in integrations:
                status = "enabled" if integration.enabled else "disabled"
                print(f"   - {integration.name} ({integration.type}): {status}")
            print()
        except Exception as e:
            print(f"   Integration service error: {e}\n")
        
        # PoAuD - List recent audits
        print("7. Listing recent audit records...")
        try:
            audits = await client.poaud.list_audits(per_page=5)
            print(f"   Found {len(audits)} audit records:")
            for audit in audits:
                print(f"   - {audit.entity_type} {audit.entity_id}: {audit.action}")
            print()
        except Exception as e:
            print(f"   PoAuD service error: {e}\n")
        
        print("=== Example completed successfully! ===")
        
    except Exception as e:
        print(f"Error: {e}")
    
    finally:
        # Always close the client
        await client.close()


if __name__ == "__main__":
    asyncio.run(main())
