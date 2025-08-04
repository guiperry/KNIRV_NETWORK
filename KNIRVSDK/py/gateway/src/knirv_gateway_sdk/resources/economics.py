"""
Economics service resource for the KNIRV Gateway SDK
"""

from __future__ import annotations

from typing import Any, Dict, List, Optional
import httpx

from ._base import BaseResource
from ..types import (
    SkillDefinition,
    SkillInvocation,
    LLMRequest,
    LLMResponse,
    ValidationRequest,
    ValidationResponse,
    FeeStructure,
    MetricsData,
    TransactionRecord,
    BurnRecord,
    Rule,
    PaginatedResponse,
)


class SkillsResource(BaseResource):
    """Skills management resource."""
    
    async def list(
        self,
        *,
        page: int = 1,
        per_page: int = 20,
        category: Optional[str] = None,
    ) -> List[SkillDefinition]:
        """
        List available skills.
        
        Args:
            page: Page number
            per_page: Items per page
            category: Filter by category
            
        Returns:
            List of skill definitions
        """
        params = {"page": page, "per_page": per_page}
        if category:
            params["category"] = category
        
        response = await self._get("/economics/skills", params=params)
        return self._parse_response_list(response, SkillDefinition)
    
    async def get(self, skill_id: str) -> SkillDefinition:
        """
        Get a specific skill by ID.
        
        Args:
            skill_id: Skill ID
            
        Returns:
            Skill definition
        """
        response = await self._get(f"/economics/skills/{skill_id}")
        return self._parse_response(response, SkillDefinition)
    
    async def invoke(
        self,
        skill_id: str,
        *,
        user_id: str,
        amount: str,
        metadata: Optional[Dict[str, Any]] = None,
    ) -> SkillInvocation:
        """
        Invoke a skill.
        
        Args:
            skill_id: Skill ID
            user_id: User ID
            amount: Amount to pay
            metadata: Additional metadata
            
        Returns:
            Skill invocation record
        """
        data = {
            "user_id": user_id,
            "amount": amount,
        }
        if metadata:
            data["metadata"] = metadata
        
        response = await self._post(f"/economics/skills/{skill_id}/invoke", json_data=data)
        return self._parse_response(response, SkillInvocation)


class LLMResource(BaseResource):
    """LLM service resource."""
    
    async def generate(self, request: LLMRequest) -> LLMResponse:
        """
        Generate text using LLM.
        
        Args:
            request: LLM request
            
        Returns:
            LLM response
        """
        response = await self._post("/economics/llm/generate", json_data=request.model_dump())
        return self._parse_response(response, LLMResponse)


class ValidationResource(BaseResource):
    """Validation service resource."""
    
    async def validate(self, request: ValidationRequest) -> ValidationResponse:
        """
        Validate data.
        
        Args:
            request: Validation request
            
        Returns:
            Validation response
        """
        response = await self._post("/economics/validation/validate", json_data=request.model_dump())
        return self._parse_response(response, ValidationResponse)


class FeesResource(BaseResource):
    """Fees management resource."""
    
    async def get_structure(self, service: str) -> FeeStructure:
        """
        Get fee structure for a service.
        
        Args:
            service: Service name
            
        Returns:
            Fee structure
        """
        response = await self._get(f"/economics/fees/{service}")
        return self._parse_response(response, FeeStructure)
    
    async def calculate(
        self,
        service: str,
        *,
        amount: str,
        parameters: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """
        Calculate fees for a service.
        
        Args:
            service: Service name
            amount: Transaction amount
            parameters: Additional parameters
            
        Returns:
            Fee calculation result
        """
        data = {"amount": amount}
        if parameters:
            data["parameters"] = parameters
        
        response = await self._post(f"/economics/fees/{service}/calculate", json_data=data)
        return response.json()


class MetricsResource(BaseResource):
    """Metrics collection resource."""
    
    async def submit(self, metrics: List[MetricsData]) -> Dict[str, Any]:
        """
        Submit metrics data.
        
        Args:
            metrics: List of metrics data
            
        Returns:
            Submission result
        """
        data = {"metrics": [metric.model_dump() for metric in metrics]}
        response = await self._post("/economics/metrics", json_data=data)
        return response.json()
    
    async def query(
        self,
        metric_name: str,
        *,
        start_time: Optional[str] = None,
        end_time: Optional[str] = None,
        labels: Optional[Dict[str, str]] = None,
    ) -> List[MetricsData]:
        """
        Query metrics data.
        
        Args:
            metric_name: Metric name
            start_time: Start time (ISO format)
            end_time: End time (ISO format)
            labels: Label filters
            
        Returns:
            List of metrics data
        """
        params = {"metric_name": metric_name}
        if start_time:
            params["start_time"] = start_time
        if end_time:
            params["end_time"] = end_time
        if labels:
            params.update(labels)
        
        response = await self._get("/economics/metrics/query", params=params)
        return self._parse_response_list(response, MetricsData)


class TransactionsResource(BaseResource):
    """Transactions management resource."""
    
    async def list(
        self,
        *,
        page: int = 1,
        per_page: int = 20,
        status: Optional[str] = None,
    ) -> List[TransactionRecord]:
        """
        List transactions.
        
        Args:
            page: Page number
            per_page: Items per page
            status: Filter by status
            
        Returns:
            List of transaction records
        """
        params = {"page": page, "per_page": per_page}
        if status:
            params["status"] = status
        
        response = await self._get("/economics/transactions", params=params)
        return self._parse_response_list(response, TransactionRecord)
    
    async def get(self, transaction_id: str) -> TransactionRecord:
        """
        Get a specific transaction.
        
        Args:
            transaction_id: Transaction ID
            
        Returns:
            Transaction record
        """
        response = await self._get(f"/economics/transactions/{transaction_id}")
        return self._parse_response(response, TransactionRecord)


class BurnResource(BaseResource):
    """Token burn management resource."""
    
    async def burn(
        self,
        *,
        amount: str,
        reason: str,
    ) -> BurnRecord:
        """
        Burn tokens.
        
        Args:
            amount: Amount to burn
            reason: Reason for burning
            
        Returns:
            Burn record
        """
        data = {"amount": amount, "reason": reason}
        response = await self._post("/economics/burn", json_data=data)
        return self._parse_response(response, BurnRecord)
    
    async def list(
        self,
        *,
        page: int = 1,
        per_page: int = 20,
    ) -> List[BurnRecord]:
        """
        List burn records.
        
        Args:
            page: Page number
            per_page: Items per page
            
        Returns:
            List of burn records
        """
        params = {"page": page, "per_page": per_page}
        response = await self._get("/economics/burn", params=params)
        return self._parse_response_list(response, BurnRecord)


class RulesResource(BaseResource):
    """Rules management resource."""
    
    async def list(self) -> List[Rule]:
        """
        List all rules.
        
        Returns:
            List of rules
        """
        response = await self._get("/economics/rules")
        return self._parse_response_list(response, Rule)
    
    async def get(self, rule_id: str) -> Rule:
        """
        Get a specific rule.
        
        Args:
            rule_id: Rule ID
            
        Returns:
            Rule
        """
        response = await self._get(f"/economics/rules/{rule_id}")
        return self._parse_response(response, Rule)
    
    async def create(
        self,
        *,
        name: str,
        description: str,
        condition: str,
        action: str,
        enabled: bool = True,
    ) -> Rule:
        """
        Create a new rule.
        
        Args:
            name: Rule name
            description: Rule description
            condition: Rule condition
            action: Rule action
            enabled: Whether the rule is enabled
            
        Returns:
            Created rule
        """
        data = {
            "name": name,
            "description": description,
            "condition": condition,
            "action": action,
            "enabled": enabled,
        }
        response = await self._post("/economics/rules", json_data=data)
        return self._parse_response(response, Rule)


class EconomicsResource(BaseResource):
    """Economics service resource."""
    
    def __init__(self, client) -> None:
        super().__init__(client)
        self.skills = SkillsResource(client)
        self.llm = LLMResource(client)
        self.validation = ValidationResource(client)
        self.fees = FeesResource(client)
        self.metrics = MetricsResource(client)
        self.transactions = TransactionsResource(client)
        self.burn = BurnResource(client)
        self.rules = RulesResource(client)
