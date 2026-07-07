"""
Economics service example for the KNIRV Gateway SDK
"""

import asyncio
import os
from knirv_gateway_sdk import KNIRVGateway
from knirv_gateway_sdk.types import LLMRequest, ValidationRequest


async def economics_example():
    """Demonstrate Economics service capabilities."""
    client = KNIRVGateway(
        base_url=os.getenv("ECONOMICS_SERVICE_URL", "http://localhost:8090"),
        api_key=os.getenv("KNIRV_GATEWAY_API_KEY", "your-api-key")
    )
    
    try:
        print("=== Economics Service Example ===\n")
        
        # 1. Skills Management
        print("1. Skills Management")
        print("-" * 20)
        
        # List all skills
        skills = await client.economics.skills.list()
        print(f"Total skills available: {len(skills)}")
        
        # List skills by category
        ai_skills = await client.economics.skills.list(category="ai")
        print(f"AI skills: {len(ai_skills)}")
        
        if skills:
            skill = skills[0]
            print(f"Example skill: {skill.name}")
            print(f"  Category: {skill.category}")
            print(f"  Cost: {skill.cost}")
            print(f"  Provider: {skill.provider}")
            
            # Get detailed skill info
            detailed_skill = await client.economics.skills.get(skill.id)
            print(f"  Description: {detailed_skill.description}")
            
            # Invoke a skill (example)
            try:
                invocation = await client.economics.skills.invoke(
                    skill_id=skill.id,
                    user_id="user-123",
                    amount=skill.cost,
                    metadata={"source": "example"}
                )
                print(f"  Invocation ID: {invocation.id}")
                print(f"  Status: {invocation.status}")
            except Exception as e:
                print(f"  Skill invocation error: {e}")
        
        print()
        
        # 2. LLM Service
        print("2. LLM Service")
        print("-" * 15)
        
        llm_request = LLMRequest(
            prompt="Explain the concept of Proof of Audit in blockchain systems",
            model="gpt-3.5-turbo",
            max_tokens=150,
            temperature=0.7
        )
        
        try:
            llm_response = await client.economics.llm.generate(llm_request)
            print(f"Prompt: {llm_request.prompt}")
            print(f"Response: {llm_response.response}")
            print(f"Model: {llm_response.model}")
            print(f"Tokens used: {llm_response.tokens_used}")
            print(f"Cost: {llm_response.cost}")
        except Exception as e:
            print(f"LLM error: {e}")
        
        print()
        
        # 3. Validation Service
        print("3. Validation Service")
        print("-" * 20)
        
        # Valid data example
        valid_request = ValidationRequest(
            data={
                "username": "alice",
                "email": "alice@example.com",
                "age": 25
            },
            rules=["required", "email_format", "min_age:18"]
        )
        
        try:
            result = await client.economics.validation.validate(valid_request)
            print(f"Valid data result: {result.valid}")
            if result.errors:
                print(f"Errors: {result.errors}")
            if result.warnings:
                print(f"Warnings: {result.warnings}")
        except Exception as e:
            print(f"Validation error: {e}")
        
        # Invalid data example
        invalid_request = ValidationRequest(
            data={
                "username": "",
                "email": "invalid-email",
                "age": 15
            },
            rules=["required", "email_format", "min_age:18"]
        )
        
        try:
            result = await client.economics.validation.validate(invalid_request)
            print(f"Invalid data result: {result.valid}")
            if result.errors:
                print(f"Errors: {result.errors}")
        except Exception as e:
            print(f"Validation error: {e}")
        
        print()
        
        # 4. Fees Management
        print("4. Fees Management")
        print("-" * 18)
        
        try:
            # Get fee structure for a service
            fee_structure = await client.economics.fees.get_structure("skill_invocation")
            print(f"Fee structure for skill_invocation:")
            print(f"  Base fee: {fee_structure.base_fee}")
            print(f"  Variable fee: {fee_structure.variable_fee}")
            print(f"  Fee type: {fee_structure.fee_type}")
            
            # Calculate fees
            fee_calc = await client.economics.fees.calculate(
                service="skill_invocation",
                amount="100",
                parameters={"skill_category": "ai"}
            )
            print(f"Fee calculation for 100 tokens: {fee_calc}")
        except Exception as e:
            print(f"Fees error: {e}")
        
        print()
        
        # 5. Metrics
        print("5. Metrics")
        print("-" * 10)
        
        try:
            # Query metrics
            metrics = await client.economics.metrics.query(
                metric_name="skill_invocations",
                labels={"category": "ai"}
            )
            print(f"Found {len(metrics)} metric data points")
            
            if metrics:
                latest = metrics[-1]
                print(f"Latest metric: {latest.value} at {latest.timestamp}")
        except Exception as e:
            print(f"Metrics error: {e}")
        
        print()
        
        # 6. Transactions
        print("6. Transactions")
        print("-" * 14)
        
        try:
            # List recent transactions
            transactions = await client.economics.transactions.list(per_page=5)
            print(f"Recent transactions: {len(transactions)}")
            
            for tx in transactions:
                print(f"  {tx.id}: {tx.amount} ({tx.status})")
        except Exception as e:
            print(f"Transactions error: {e}")
        
        print()
        
        # 7. Rules Management
        print("7. Rules Management")
        print("-" * 18)
        
        try:
            # List rules
            rules = await client.economics.rules.list()
            print(f"Active rules: {len(rules)}")
            
            for rule in rules[:3]:  # Show first 3
                status = "enabled" if rule.enabled else "disabled"
                print(f"  {rule.name}: {rule.description} ({status})")
        except Exception as e:
            print(f"Rules error: {e}")
        
        print("\n=== Economics Example Completed ===")
        
    finally:
        await client.close()


if __name__ == "__main__":
    asyncio.run(economics_example())
