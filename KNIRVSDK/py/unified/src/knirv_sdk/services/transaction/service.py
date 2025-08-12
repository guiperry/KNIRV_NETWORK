"""
KNIRV Transaction Service

This module provides blockchain transaction management services.
"""

from __future__ import annotations
from typing import Any, Dict, List, Optional
import httpx

from ...config import TransactionConfig
from ...exceptions import KNIRVTransactionError, KNIRVValidationError
from ..base import BaseService


class TransactionService(BaseService):
    """
    Service for managing blockchain transactions.
    
    This service provides access to:
    - Transaction creation and submission
    - Transaction status and history
    - Block information
    - Chain information
    - Peer management
    
    Example:
        ```python
        from knirv_sdk.services.transaction import TransactionService
        from knirv_sdk.config import TransactionConfig
        
        config = TransactionConfig(
            base_url="https://transaction.knirv.com",
            api_key="your-api-key"
        )
        service = TransactionService(config)
        
        # Create a transaction
        tx = await service.create_transaction({
            "from": "address1",
            "to": "address2", 
            "amount": 100
        })
        
        # Submit the transaction
        result = await service.submit_transaction(tx["id"])
        ```
    """
    
    def __init__(self, config: Optional[TransactionConfig] = None) -> None:
        """
        Initialize the Transaction service.
        
        Args:
            config: Transaction configuration. If None, uses default configuration.
        """
        self._config = config or TransactionConfig()
        super().__init__(self._config.base_url, self._config)
    
    async def create_transaction(self, transaction_data: Dict[str, Any]) -> Dict[str, Any]:
        """
        Create a new transaction.
        
        Args:
            transaction_data: Transaction data
            
        Returns:
            Created transaction
        """
        if not transaction_data.get("from"):
            raise KNIRVValidationError("From address is required")
        if not transaction_data.get("to"):
            raise KNIRVValidationError("To address is required")
        if not transaction_data.get("amount"):
            raise KNIRVValidationError("Amount is required")
        
        try:
            response = await self._request("POST", "/transaction", json=transaction_data)
            return response.json()
        except httpx.HTTPError as e:
            raise KNIRVTransactionError(f"Failed to create transaction: {e}") from e
    
    async def submit_transaction(self, transaction_id: str) -> Dict[str, Any]:
        """
        Submit a transaction to the blockchain.
        
        Args:
            transaction_id: Transaction identifier
            
        Returns:
            Submission result
        """
        if not transaction_id:
            raise KNIRVValidationError("Transaction ID is required")
        
        try:
            response = await self._request("POST", f"/transaction/{transaction_id}/submit")
            return response.json()
        except httpx.HTTPError as e:
            raise KNIRVTransactionError(f"Failed to submit transaction {transaction_id}: {e}") from e
    
    async def get_transaction(self, transaction_id: str) -> Dict[str, Any]:
        """
        Get transaction details.
        
        Args:
            transaction_id: Transaction identifier
            
        Returns:
            Transaction details
        """
        if not transaction_id:
            raise KNIRVValidationError("Transaction ID is required")
        
        try:
            response = await self._request("GET", f"/transaction/{transaction_id}")
            return response.json()
        except httpx.HTTPError as e:
            raise KNIRVTransactionError(f"Failed to get transaction {transaction_id}: {e}") from e
    
    async def list_transactions(
        self,
        *,
        address: Optional[str] = None,
        status: Optional[str] = None,
        page: int = 1,
        per_page: int = 20,
    ) -> Dict[str, Any]:
        """
        List transactions.
        
        Args:
            address: Filter by address
            status: Filter by status
            page: Page number
            per_page: Items per page
            
        Returns:
            Paginated list of transactions
        """
        params = {"page": page, "per_page": per_page}
        if address:
            params["address"] = address
        if status:
            params["status"] = status
        
        try:
            response = await self._request("GET", "/transaction", params=params)
            return response.json()
        except httpx.HTTPError as e:
            raise KNIRVTransactionError(f"Failed to list transactions: {e}") from e
    
    async def get_transaction_status(self, transaction_id: str) -> Dict[str, Any]:
        """
        Get transaction status.
        
        Args:
            transaction_id: Transaction identifier
            
        Returns:
            Transaction status
        """
        if not transaction_id:
            raise KNIRVValidationError("Transaction ID is required")
        
        try:
            response = await self._request("GET", f"/transaction/{transaction_id}/status")
            return response.json()
        except httpx.HTTPError as e:
            raise KNIRVTransactionError(f"Failed to get transaction status {transaction_id}: {e}") from e
    
    async def get_block(self, block_id: str) -> Dict[str, Any]:
        """
        Get block information.
        
        Args:
            block_id: Block identifier
            
        Returns:
            Block information
        """
        if not block_id:
            raise KNIRVValidationError("Block ID is required")
        
        try:
            response = await self._request("GET", f"/block/{block_id}")
            return response.json()
        except httpx.HTTPError as e:
            raise KNIRVTransactionError(f"Failed to get block {block_id}: {e}") from e
    
    async def get_latest_block(self) -> Dict[str, Any]:
        """
        Get the latest block.
        
        Returns:
            Latest block information
        """
        try:
            response = await self._request("GET", "/block/latest")
            return response.json()
        except httpx.HTTPError as e:
            raise KNIRVTransactionError(f"Failed to get latest block: {e}") from e
    
    async def get_chain_info(self) -> Dict[str, Any]:
        """
        Get blockchain information.
        
        Returns:
            Chain information
        """
        try:
            response = await self._request("GET", "/chain")
            return response.json()
        except httpx.HTTPError as e:
            raise KNIRVTransactionError(f"Failed to get chain info: {e}") from e
    
    async def get_peers(self) -> List[Dict[str, Any]]:
        """
        Get connected peers.
        
        Returns:
            List of connected peers
        """
        try:
            response = await self._request("GET", "/peers")
            return response.json()
        except httpx.HTTPError as e:
            raise KNIRVTransactionError(f"Failed to get peers: {e}") from e
    
    async def ping(self) -> Dict[str, Any]:
        """
        Ping the transaction service.
        
        Returns:
            Ping response
        """
        try:
            response = await self._request("GET", "/ping")
            return response.json()
        except httpx.HTTPError as e:
            raise KNIRVTransactionError(f"Failed to ping transaction service: {e}") from e
