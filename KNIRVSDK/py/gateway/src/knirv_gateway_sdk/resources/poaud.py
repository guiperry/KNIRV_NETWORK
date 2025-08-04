"""
PoAuD (Proof of Audit) service resource for the KNIRV Gateway SDK
"""

from __future__ import annotations

from typing import Any, Dict, List, Optional

from ._base import BaseResource
from ..types import AuditRecord, ProofOfAudit


class PoAuDResource(BaseResource):
    """PoAuD (Proof of Audit) service resource."""
    
    async def create_audit(
        self,
        *,
        entity_id: str,
        entity_type: str,
        action: str,
        details: Dict[str, Any],
        auditor: str,
    ) -> AuditRecord:
        """
        Create a new audit record.
        
        Args:
            entity_id: Entity ID being audited
            entity_type: Type of entity
            action: Action being audited
            details: Audit details
            auditor: Auditor identifier
            
        Returns:
            Created audit record
        """
        data = {
            "entity_id": entity_id,
            "entity_type": entity_type,
            "action": action,
            "details": details,
            "auditor": auditor,
        }
        response = await self._post("/poaud/audit", json_data=data)
        return self._parse_response(response, AuditRecord)
    
    async def get_audit(self, audit_id: str) -> AuditRecord:
        """
        Get a specific audit record.
        
        Args:
            audit_id: Audit ID
            
        Returns:
            Audit record
        """
        response = await self._get(f"/poaud/audit/{audit_id}")
        return self._parse_response(response, AuditRecord)
    
    async def list_audits(
        self,
        *,
        page: int = 1,
        per_page: int = 20,
        entity_id: Optional[str] = None,
        entity_type: Optional[str] = None,
        auditor: Optional[str] = None,
        start_time: Optional[str] = None,
        end_time: Optional[str] = None,
    ) -> List[AuditRecord]:
        """
        List audit records.
        
        Args:
            page: Page number
            per_page: Items per page
            entity_id: Filter by entity ID
            entity_type: Filter by entity type
            auditor: Filter by auditor
            start_time: Filter by start time (ISO format)
            end_time: Filter by end time (ISO format)
            
        Returns:
            List of audit records
        """
        params = {"page": page, "per_page": per_page}
        if entity_id:
            params["entity_id"] = entity_id
        if entity_type:
            params["entity_type"] = entity_type
        if auditor:
            params["auditor"] = auditor
        if start_time:
            params["start_time"] = start_time
        if end_time:
            params["end_time"] = end_time
        
        response = await self._get("/poaud/audit", params=params)
        return self._parse_response_list(response, AuditRecord)
    
    async def create_proof(
        self,
        audit_id: str,
        *,
        validator: str,
    ) -> ProofOfAudit:
        """
        Create a proof of audit.
        
        Args:
            audit_id: Audit ID
            validator: Validator identifier
            
        Returns:
            Proof of audit
        """
        data = {
            "audit_id": audit_id,
            "validator": validator,
        }
        response = await self._post("/poaud/proof", json_data=data)
        return self._parse_response(response, ProofOfAudit)
    
    async def get_proof(self, audit_id: str) -> ProofOfAudit:
        """
        Get proof of audit for a specific audit.
        
        Args:
            audit_id: Audit ID
            
        Returns:
            Proof of audit
        """
        response = await self._get(f"/poaud/proof/{audit_id}")
        return self._parse_response(response, ProofOfAudit)
    
    async def verify_proof(
        self,
        audit_id: str,
        *,
        proof_hash: str,
    ) -> Dict[str, Any]:
        """
        Verify a proof of audit.
        
        Args:
            audit_id: Audit ID
            proof_hash: Proof hash to verify
            
        Returns:
            Verification result
        """
        data = {"proof_hash": proof_hash}
        response = await self._post(f"/poaud/proof/{audit_id}/verify", json_data=data)
        return response.json()
    
    async def get_merkle_tree(self, audit_id: str) -> Dict[str, Any]:
        """
        Get Merkle tree for an audit.
        
        Args:
            audit_id: Audit ID
            
        Returns:
            Merkle tree data
        """
        response = await self._get(f"/poaud/merkle/{audit_id}")
        return response.json()
    
    async def validate_chain(
        self,
        *,
        start_audit_id: str,
        end_audit_id: str,
    ) -> Dict[str, Any]:
        """
        Validate audit chain integrity.
        
        Args:
            start_audit_id: Starting audit ID
            end_audit_id: Ending audit ID
            
        Returns:
            Chain validation result
        """
        params = {
            "start_audit_id": start_audit_id,
            "end_audit_id": end_audit_id,
        }
        response = await self._get("/poaud/validate-chain", params=params)
        return response.json()
