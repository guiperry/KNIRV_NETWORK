"""
Tests for Gateway services.
"""

import pytest
from unittest.mock import AsyncMock, MagicMock
import httpx

from knirv_sdk.services.gateway import GatewayService, EconomicsService, SkillsService
from knirv_sdk.exceptions import KNIRVValidationError, KNIRVGatewayError


class TestSkillsService:
    """Test cases for SkillsService."""
    
    @pytest.fixture
    def mock_http_client(self):
        """Create a mock HTTP client."""
        return AsyncMock(spec=httpx.AsyncClient)
    
    @pytest.fixture
    def skills_service(self, mock_http_client, gateway_config):
        """Create a SkillsService instance with mock HTTP client."""
        return SkillsService(mock_http_client, gateway_config)
    
    @pytest.mark.asyncio
    async def test_list_skills(self, skills_service, mock_http_client, mock_skills_response):
        """Test listing skills."""
        mock_response = MagicMock()
        mock_response.json.return_value = mock_skills_response
        mock_http_client.get.return_value = mock_response
        
        result = await skills_service.list()
        
        mock_http_client.get.assert_called_once_with(
            "/economics/skills",
            params={"page": 1, "per_page": 20}
        )
        assert result == mock_skills_response
    
    @pytest.mark.asyncio
    async def test_list_skills_with_filters(self, skills_service, mock_http_client):
        """Test listing skills with filters."""
        mock_response = MagicMock()
        mock_response.json.return_value = {"data": []}
        mock_http_client.get.return_value = mock_response
        
        await skills_service.list(
            page=2,
            per_page=10,
            category="test",
            query="search"
        )
        
        mock_http_client.get.assert_called_once_with(
            "/economics/skills",
            params={
                "page": 2,
                "per_page": 10,
                "category": "test",
                "query": "search"
            }
        )
    
    @pytest.mark.asyncio
    async def test_get_skill(self, skills_service, mock_http_client):
        """Test getting a specific skill."""
        skill_data = {"id": "skill-1", "name": "Test Skill"}
        mock_response = MagicMock()
        mock_response.json.return_value = skill_data
        mock_http_client.get.return_value = mock_response
        
        result = await skills_service.get("skill-1")
        
        mock_http_client.get.assert_called_once_with("/economics/skills/skill-1")
        assert result == skill_data
    
    @pytest.mark.asyncio
    async def test_get_skill_empty_id(self, skills_service):
        """Test getting skill with empty ID raises validation error."""
        with pytest.raises(KNIRVValidationError, match="Skill ID is required"):
            await skills_service.get("")
    
    @pytest.mark.asyncio
    async def test_create_skill(self, skills_service, mock_http_client, mock_skill_data):
        """Test creating a new skill."""
        created_skill = {**mock_skill_data, "id": "skill-123"}
        mock_response = MagicMock()
        mock_response.json.return_value = created_skill
        mock_http_client.post.return_value = mock_response
        
        result = await skills_service.create(mock_skill_data)
        
        mock_http_client.post.assert_called_once_with(
            "/economics/skills",
            json=mock_skill_data
        )
        assert result == created_skill
    
    @pytest.mark.asyncio
    async def test_create_skill_validation_errors(self, skills_service):
        """Test skill creation validation errors."""
        # Missing name
        with pytest.raises(KNIRVValidationError, match="Name is required"):
            await skills_service.create({"description": "test", "cost": 100})
        
        # Missing description
        with pytest.raises(KNIRVValidationError, match="Description is required"):
            await skills_service.create({"name": "test", "cost": 100})
        
        # Missing cost
        with pytest.raises(KNIRVValidationError, match="Cost is required"):
            await skills_service.create({"name": "test", "description": "test"})
        
        # Negative cost
        with pytest.raises(KNIRVValidationError, match="Cost must be positive"):
            await skills_service.create({
                "name": "test",
                "description": "test", 
                "cost": -10
            })
    
    @pytest.mark.asyncio
    async def test_update_skill(self, skills_service, mock_http_client):
        """Test updating a skill."""
        update_data = {"name": "Updated Skill"}
        updated_skill = {"id": "skill-1", **update_data}
        mock_response = MagicMock()
        mock_response.json.return_value = updated_skill
        mock_http_client.put.return_value = mock_response
        
        result = await skills_service.update("skill-1", update_data)
        
        mock_http_client.put.assert_called_once_with(
            "/economics/skills/skill-1",
            json=update_data
        )
        assert result == updated_skill
    
    @pytest.mark.asyncio
    async def test_update_skill_validation_errors(self, skills_service):
        """Test skill update validation errors."""
        # Empty ID
        with pytest.raises(KNIRVValidationError, match="Skill ID is required"):
            await skills_service.update("", {"name": "test"})
        
        # Negative cost
        with pytest.raises(KNIRVValidationError, match="Cost must be positive"):
            await skills_service.update("skill-1", {"cost": -10})
    
    @pytest.mark.asyncio
    async def test_delete_skill(self, skills_service, mock_http_client):
        """Test deleting a skill."""
        mock_http_client.delete.return_value = MagicMock()
        
        await skills_service.delete("skill-1")
        
        mock_http_client.delete.assert_called_once_with("/economics/skills/skill-1")
    
    @pytest.mark.asyncio
    async def test_delete_skill_empty_id(self, skills_service):
        """Test deleting skill with empty ID raises validation error."""
        with pytest.raises(KNIRVValidationError, match="Skill ID is required"):
            await skills_service.delete("")
    
    @pytest.mark.asyncio
    async def test_search_skills(self, skills_service, mock_http_client):
        """Test searching skills."""
        search_results = {"data": [], "total": 0}
        mock_response = MagicMock()
        mock_response.json.return_value = search_results
        mock_http_client.get.return_value = mock_response
        
        result = await skills_service.search("test query", category="test")
        
        mock_http_client.get.assert_called_once_with(
            "/economics/skills/search",
            params={"query": "test query", "category": "test"}
        )
        assert result == search_results
    
    @pytest.mark.asyncio
    async def test_search_skills_empty_query(self, skills_service):
        """Test searching skills with empty query raises validation error."""
        with pytest.raises(KNIRVValidationError, match="Search query is required"):
            await skills_service.search("")
    
    @pytest.mark.asyncio
    async def test_http_error_handling(self, skills_service, mock_http_client):
        """Test HTTP error handling."""
        mock_http_client.get.side_effect = httpx.HTTPError("Connection failed")
        
        with pytest.raises(KNIRVGatewayError, match="Failed to list skills"):
            await skills_service.list()


class TestGatewayService:
    """Test cases for GatewayService."""
    
    @pytest.mark.asyncio
    async def test_gateway_service_initialization(self, gateway_config):
        """Test GatewayService initialization."""
        service = GatewayService(gateway_config)
        
        assert service._config == gateway_config
        assert service.economics is not None
        assert service.health is not None
        assert service.integration is not None
        assert service.poaud is not None
    
    @pytest.mark.asyncio
    async def test_gateway_service_sub_services(self, gateway_config):
        """Test that sub-services are properly initialized."""
        service = GatewayService(gateway_config)
        
        # Test that accessing sub-services returns the same instance
        economics1 = service.economics
        economics2 = service.economics
        assert economics1 is economics2
        
        # Test that sub-services have the correct type
        assert isinstance(service.economics, EconomicsService)
        assert hasattr(service.economics, 'skills')
        assert isinstance(service.economics.skills, SkillsService)
