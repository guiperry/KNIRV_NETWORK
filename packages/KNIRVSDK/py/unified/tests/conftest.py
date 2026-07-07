"""
Pytest configuration and fixtures for KNIRV SDK tests.
"""

import pytest
import asyncio
from unittest.mock import AsyncMock, MagicMock
from typing import AsyncGenerator, Generator

from knirv_sdk import KNIRVClient, ClientConfig
from knirv_sdk.config import GatewayConfig, TransactionConfig, TransmissionConfig


@pytest.fixture(scope="session")
def event_loop() -> Generator[asyncio.AbstractEventLoop, None, None]:
    """Create an instance of the default event loop for the test session."""
    loop = asyncio.get_event_loop_policy().new_event_loop()
    yield loop
    loop.close()


@pytest.fixture
def mock_config() -> ClientConfig:
    """Create a test configuration."""
    return ClientConfig.for_testing(
        api_key="test-api-key"
    )


@pytest.fixture
def gateway_config() -> GatewayConfig:
    """Create a test gateway configuration."""
    return GatewayConfig(
        base_url="https://test-gateway.knirv.com",
        api_key="test-api-key",
        timeout=5,
        retries=1
    )


@pytest.fixture
def transaction_config() -> TransactionConfig:
    """Create a test transaction configuration."""
    return TransactionConfig(
        base_url="https://test-transaction.knirv.com",
        api_key="test-api-key",
        timeout=5,
        retries=1
    )


@pytest.fixture
def transmission_config() -> TransmissionConfig:
    """Create a test transmission configuration."""
    return TransmissionConfig(
        base_url="https://test-transmission.knirv.com",
        api_key="test-api-key",
        timeout=5,
        retries=1
    )


@pytest.fixture
async def mock_client(mock_config: ClientConfig) -> AsyncGenerator[KNIRVClient, None]:
    """Create a mock KNIRV client for testing."""
    client = KNIRVClient(mock_config)
    
    # Mock the HTTP clients for each service
    mock_http_client = AsyncMock()
    
    # Mock successful responses
    mock_response = MagicMock()
    mock_response.json.return_value = {"data": "test"}
    mock_response.status_code = 200
    mock_http_client.get.return_value = mock_response
    mock_http_client.post.return_value = mock_response
    mock_http_client.put.return_value = mock_response
    mock_http_client.delete.return_value = mock_response
    
    # Replace the HTTP clients
    if hasattr(client, '_gateway') and client._gateway:
        client._gateway._http_client = mock_http_client
    if hasattr(client, '_transaction') and client._transaction:
        client._transaction._http_client = mock_http_client
    if hasattr(client, '_transmission') and client._transmission:
        client._transmission._http_client = mock_http_client
    
    yield client
    
    await client.close()


@pytest.fixture
def mock_http_response():
    """Create a mock HTTP response."""
    response = MagicMock()
    response.status_code = 200
    response.json.return_value = {"data": "test", "status": "success"}
    response.text = "test response"
    return response


@pytest.fixture
def mock_skills_response():
    """Create a mock skills list response."""
    return {
        "data": [
            {
                "id": "skill-1",
                "name": "Test Skill 1",
                "description": "A test skill",
                "cost": 100,
                "category": "test"
            },
            {
                "id": "skill-2", 
                "name": "Test Skill 2",
                "description": "Another test skill",
                "cost": 200,
                "category": "test"
            }
        ],
        "pagination": {
            "page": 1,
            "per_page": 20,
            "total": 2,
            "pages": 1
        }
    }


@pytest.fixture
def mock_skill_data():
    """Create mock skill data for testing."""
    return {
        "name": "Test Skill",
        "description": "A test skill for unit testing",
        "cost": 100,
        "category": "test",
        "capabilities": ["test", "validation"]
    }


@pytest.fixture
def mock_transaction_data():
    """Create mock transaction data for testing."""
    return {
        "from": "test-address-1",
        "to": "test-address-2",
        "amount": 100,
        "data": "test transaction data"
    }


@pytest.fixture
def mock_uri_data():
    """Create mock URI data for testing."""
    return {
        "resource_type": "skill",
        "id": "test-skill-123",
        "action": "execute",
        "parameters": {"param1": "value1"}
    }
