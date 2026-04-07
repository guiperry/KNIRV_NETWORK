"""
Type definitions for the KNIRV Gateway SDK
"""

from __future__ import annotations

from typing import Any, Dict, List, Optional, Union
from typing_extensions import TypedDict, Literal
from pydantic import BaseModel


# Base types
class BaseResponse(BaseModel):
    """Base response model for all API responses."""
    success: bool
    message: Optional[str] = None
    timestamp: Optional[str] = None


# Economics Service Types
class SkillDefinition(BaseModel):
    """Skill definition model."""
    id: str
    name: str
    description: str
    category: str
    cost: str
    provider: str
    metadata: Optional[Dict[str, Any]] = None


class SkillInvocation(BaseModel):
    """Skill invocation model."""
    id: str
    skill_id: str
    user_id: str
    amount: str
    status: Literal["pending", "completed", "failed"]
    result: Optional[Any] = None
    timestamp: str


class LLMRequest(BaseModel):
    """LLM request model."""
    prompt: str
    model: Optional[str] = None
    max_tokens: Optional[int] = None
    temperature: Optional[float] = None
    parameters: Optional[Dict[str, Any]] = None


class LLMResponse(BaseResponse):
    """LLM response model."""
    response: str
    model: str
    tokens_used: int
    cost: str


class ValidationRequest(BaseModel):
    """Validation request model."""
    data: Dict[str, Any]
    validation_schema: Optional[Dict[str, Any]] = None
    rules: Optional[List[str]] = None


class ValidationResponse(BaseResponse):
    """Validation response model."""
    valid: bool
    errors: Optional[List[str]] = None
    warnings: Optional[List[str]] = None


class FeeStructure(BaseModel):
    """Fee structure model."""
    base_fee: str
    variable_fee: str
    fee_type: Literal["fixed", "percentage", "tiered"]
    parameters: Optional[Dict[str, Any]] = None


class MetricsData(BaseModel):
    """Metrics data model."""
    metric_name: str
    value: Union[int, float, str]
    timestamp: str
    labels: Optional[Dict[str, str]] = None


class TransactionRecord(BaseModel):
    """Transaction record model."""
    id: str
    from_address: str
    to_address: str
    amount: str
    fee: str
    status: Literal["pending", "confirmed", "failed"]
    timestamp: str
    block_height: Optional[int] = None


class BurnRecord(BaseModel):
    """Burn record model."""
    id: str
    amount: str
    reason: str
    timestamp: str
    transaction_id: str


class Rule(BaseModel):
    """Rule model."""
    id: str
    name: str
    description: str
    condition: str
    action: str
    enabled: bool
    created_at: str
    updated_at: str


# Gateway Service Types
class Route(BaseModel):
    """Route model."""
    path: str
    method: str
    handler: str
    middleware: Optional[List[str]] = None
    enabled: bool


class HealthStatus(BaseModel):
    """Health status model."""
    service: str
    status: Literal["healthy", "unhealthy", "degraded"]
    uptime: int
    last_check: str
    details: Optional[Dict[str, Any]] = None


class ServiceStatus(BaseModel):
    """Service status model."""
    name: str
    version: str
    status: Literal["running", "stopped", "error"]
    dependencies: Optional[List[str]] = None


# Integration Service Types
class Integration(BaseModel):
    """Integration model."""
    id: str
    name: str
    type: str
    config: Dict[str, Any]
    enabled: bool
    created_at: str
    updated_at: str


# PoAuD Service Types
class AuditRecord(BaseModel):
    """Audit record model."""
    id: str
    entity_id: str
    entity_type: str
    action: str
    details: Dict[str, Any]
    auditor: str
    timestamp: str
    signature: str


class ProofOfAudit(BaseModel):
    """Proof of Audit model."""
    audit_id: str
    proof_hash: str
    merkle_root: str
    timestamp: str
    validator: str
    signature: str


# Request/Response wrapper types
class PaginatedResponse(BaseResponse):
    """Paginated response model."""
    data: List[Any]
    page: int
    per_page: int
    total: int
    total_pages: int


# Client configuration types
class ClientConfig(TypedDict, total=False):
    """Client configuration options."""
    base_url: str
    api_key: Optional[str]
    timeout: int
    max_retries: int
    debug: bool
