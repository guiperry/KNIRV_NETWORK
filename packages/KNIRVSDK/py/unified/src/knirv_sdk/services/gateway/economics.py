"""
KNIRV Economics Service

This module provides access to KNIRV Economics services including:
- Skills: Skill management and execution
- LLM: Language model services
- Validation: Data validation services
"""

from __future__ import annotations
from typing import Any, Dict, List, Optional, Union
import httpx

from ...exceptions import KNIRVValidationError, KNIRVGatewayError


class SkillsService:
    """Skills management service."""
    
    def __init__(self, http_client: httpx.AsyncClient, config: Any) -> None:
        self._client = http_client
        self._config = config
    
    async def list(
        self,
        *,
        page: int = 1,
        per_page: int = 20,
        category: Optional[str] = None,
        query: Optional[str] = None,
    ) -> Dict[str, Any]:
        """
        List available skills.
        
        Args:
            page: Page number
            per_page: Items per page
            category: Filter by category
            query: Search query
            
        Returns:
            Paginated list of skills
        """
        params = {"page": page, "per_page": per_page}
        if category:
            params["category"] = category
        if query:
            params["query"] = query
        
        try:
            response = await self._client.get("/economics/skills", params=params)
            return response.json()
        except httpx.HTTPError as e:
            raise KNIRVGatewayError(f"Failed to list skills: {e}") from e
    
    async def get(self, skill_id: str) -> Dict[str, Any]:
        """
        Get a specific skill by ID.
        
        Args:
            skill_id: Skill identifier
            
        Returns:
            Skill definition
        """
        if not skill_id:
            raise KNIRVValidationError("Skill ID is required")
        
        try:
            response = await self._client.get(f"/economics/skills/{skill_id}")
            return response.json()
        except httpx.HTTPError as e:
            raise KNIRVGatewayError(f"Failed to get skill {skill_id}: {e}") from e
    
    async def create(self, skill_data: Dict[str, Any]) -> Dict[str, Any]:
        """
        Create a new skill.
        
        Args:
            skill_data: Skill definition data
            
        Returns:
            Created skill
        """
        if not skill_data.get("name"):
            raise KNIRVValidationError("Name is required")
        if not skill_data.get("description"):
            raise KNIRVValidationError("Description is required")
        if skill_data.get("cost") is None:
            raise KNIRVValidationError("Cost is required")
        if skill_data.get("cost", 0) < 0:
            raise KNIRVValidationError("Cost must be positive")
        
        try:
            response = await self._client.post("/economics/skills", json=skill_data)
            return response.json()
        except httpx.HTTPError as e:
            raise KNIRVGatewayError(f"Failed to create skill: {e}") from e
    
    async def update(self, skill_id: str, skill_data: Dict[str, Any]) -> Dict[str, Any]:
        """
        Update an existing skill.
        
        Args:
            skill_id: Skill identifier
            skill_data: Updated skill data
            
        Returns:
            Updated skill
        """
        if not skill_id:
            raise KNIRVValidationError("Skill ID is required")
        if skill_data.get("cost") is not None and skill_data.get("cost") < 0:
            raise KNIRVValidationError("Cost must be positive")
        
        try:
            response = await self._client.put(f"/economics/skills/{skill_id}", json=skill_data)
            return response.json()
        except httpx.HTTPError as e:
            raise KNIRVGatewayError(f"Failed to update skill {skill_id}: {e}") from e
    
    async def delete(self, skill_id: str) -> None:
        """
        Delete a skill.
        
        Args:
            skill_id: Skill identifier
        """
        if not skill_id:
            raise KNIRVValidationError("Skill ID is required")
        
        try:
            await self._client.delete(f"/economics/skills/{skill_id}")
        except httpx.HTTPError as e:
            raise KNIRVGatewayError(f"Failed to delete skill {skill_id}: {e}") from e
    
    async def search(self, query: str, **options: Any) -> Dict[str, Any]:
        """
        Search for skills.
        
        Args:
            query: Search query
            **options: Additional search options
            
        Returns:
            Search results
        """
        if not query:
            raise KNIRVValidationError("Search query is required")
        
        params = {"query": query, **options}
        
        try:
            response = await self._client.get("/economics/skills/search", params=params)
            return response.json()
        except httpx.HTTPError as e:
            raise KNIRVGatewayError(f"Failed to search skills: {e}") from e


class LLMService:
    """Language model service."""
    
    def __init__(self, http_client: httpx.AsyncClient, config: Any) -> None:
        self._client = http_client
        self._config = config
    
    async def estimate_cost(self, request_data: Dict[str, Any]) -> Dict[str, Any]:
        """
        Estimate the cost of an LLM request.
        
        Args:
            request_data: LLM request data
            
        Returns:
            Cost estimate
        """
        if not request_data.get("text"):
            raise KNIRVValidationError("Text is required")
        if not request_data.get("model"):
            raise KNIRVValidationError("Model is required")
        
        try:
            response = await self._client.post("/economics/llm/estimate", json=request_data)
            return response.json()
        except httpx.HTTPError as e:
            raise KNIRVGatewayError(f"Failed to estimate LLM cost: {e}") from e
    
    async def get_usage(self, **options: Any) -> Dict[str, Any]:
        """
        Get LLM usage statistics.
        
        Args:
            **options: Usage query options
            
        Returns:
            Usage statistics
        """
        params = options if options else {}
        
        try:
            response = await self._client.get("/economics/llm/usage", params=params if params else None)
            return response.json()
        except httpx.HTTPError as e:
            raise KNIRVGatewayError(f"Failed to get LLM usage: {e}") from e


class ValidationService:
    """Data validation service."""
    
    def __init__(self, http_client: httpx.AsyncClient, config: Any) -> None:
        self._client = http_client
        self._config = config
    
    async def validate(self, request_data: Dict[str, Any]) -> Dict[str, Any]:
        """
        Validate data using a skill.
        
        Args:
            request_data: Validation request data
            
        Returns:
            Validation result
        """
        if not request_data.get("skill_id"):
            raise KNIRVValidationError("Skill ID is required")
        if not request_data.get("data"):
            raise KNIRVValidationError("Data is required")
        
        try:
            response = await self._client.post("/economics/validation/validate", json=request_data)
            return response.json()
        except httpx.HTTPError as e:
            raise KNIRVGatewayError(f"Failed to validate data: {e}") from e
    
    async def list_rules(self, **options: Any) -> Dict[str, Any]:
        """
        List validation rules.
        
        Args:
            **options: Query options
            
        Returns:
            List of validation rules
        """
        params = options if options else {}
        
        try:
            response = await self._client.get("/economics/validation/rules", params=params if params else None)
            return response.json()
        except httpx.HTTPError as e:
            raise KNIRVGatewayError(f"Failed to list validation rules: {e}") from e


class EconomicsService:
    """Main economics service that provides access to all economics sub-services."""
    
    def __init__(self, http_client: httpx.AsyncClient, config: Any) -> None:
        self._client = http_client
        self._config = config
        
        # Initialize sub-services
        self.skills = SkillsService(http_client, config)
        self.llm = LLMService(http_client, config)
        self.validation = ValidationService(http_client, config)
