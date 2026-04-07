"""
KNIRV PoAuD (Proof of Audit) Service

This module provides proof of audit services for the KNIRV network.
"""

from __future__ import annotations
from typing import Any, Dict, List, Optional
import httpx

from ...exceptions import KNIRVGatewayError, KNIRVValidationError


class PoAuDService:
    """Proof of Audit service."""
    
    def __init__(self, http_client: httpx.AsyncClient, config: Any) -> None:
        self._client = http_client
        self._config = config
    
    async def create_audit(self, audit_data: Dict[str, Any]) -> Dict[str, Any]:
        """
        Create a new audit record.
        
        Args:
            audit_data: Audit data to record
            
        Returns:
            Created audit record
        """
        if not audit_data.get("entity_id"):
            raise KNIRVValidationError("Entity ID is required")
        if not audit_data.get("audit_type"):
            raise KNIRVValidationError("Audit type is required")
        
        try:
            response = await self._client.post("/poaud/audits", json=audit_data)
            return response.json()
        except httpx.HTTPError as e:
            raise KNIRVGatewayError(f"Failed to create audit: {e}") from e
    
    async def get_audit(self, audit_id: str) -> Dict[str, Any]:
        """
        Get an audit record by ID.
        
        Args:
            audit_id: Audit identifier
            
        Returns:
            Audit record
        """
        if not audit_id:
            raise KNIRVValidationError("Audit ID is required")
        
        try:
            response = await self._client.get(f"/poaud/audits/{audit_id}")
            return response.json()
        except httpx.HTTPError as e:
            raise KNIRVGatewayError(f"Failed to get audit {audit_id}: {e}") from e
    
    async def list_audits(
        self,
        *,
        entity_id: Optional[str] = None,
        audit_type: Optional[str] = None,
        page: int = 1,
        per_page: int = 20,
    ) -> Dict[str, Any]:
        """
        List audit records.
        
        Args:
            entity_id: Filter by entity ID
            audit_type: Filter by audit type
            page: Page number
            per_page: Items per page
            
        Returns:
            Paginated list of audit records
        """
        params = {"page": page, "per_page": per_page}
        if entity_id:
            params["entity_id"] = entity_id
        if audit_type:
            params["audit_type"] = audit_type
        
        try:
            response = await self._client.get("/poaud/audits", params=params)
            return response.json()
        except httpx.HTTPError as e:
            raise KNIRVGatewayError(f"Failed to list audits: {e}") from e
    
    async def verify_audit(self, audit_id: str) -> Dict[str, Any]:
        """
        Verify an audit record.
        
        Args:
            audit_id: Audit identifier
            
        Returns:
            Verification result
        """
        if not audit_id:
            raise KNIRVValidationError("Audit ID is required")
        
        try:
            response = await self._client.post(f"/poaud/audits/{audit_id}/verify")
            return response.json()
        except httpx.HTTPError as e:
            raise KNIRVGatewayError(f"Failed to verify audit {audit_id}: {e}") from e
    
    async def get_proof(self, audit_id: str) -> Dict[str, Any]:
        """
        Get cryptographic proof for an audit.
        
        Args:
            audit_id: Audit identifier
            
        Returns:
            Cryptographic proof
        """
        if not audit_id:
            raise KNIRVValidationError("Audit ID is required")
        
        try:
            response = await self._client.get(f"/poaud/audits/{audit_id}/proof")
            return response.json()
        except httpx.HTTPError as e:
            raise KNIRVGatewayError(f"Failed to get proof for audit {audit_id}: {e}") from e
