# File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

from typing import Dict, Optional

from pydantic import Field as FieldInfo

from .._models import BaseModel

__all__ = ["Transaction"]


class Transaction(BaseModel):
    id: str
    """Transaction hash/ID."""

    fee: str
    """NRN token gas fee paid for this transaction"""

    from_: str = FieldInfo(alias="from")
    """Sender address"""

    public_key: str
    """Public key of the sender"""

    signature: str
    """Cryptographic signature"""

    timestamp: int
    """Unix timestamp (nanoseconds) when the transaction was created"""

    type: str
    """Transaction type for MCP transactions"""

    version: int

    data: Optional[Dict[str, object]] = None
    """Transaction payload, structure depends on transaction type.

    For MCP ops, this will be JSON of MCPRegisterCapabilityData,
    MCPInvokeCapabilityData, etc.
    """

    status: Optional[str] = None
    """Transaction status (PENDING, SUCCESS, FAILED)"""

    to: Optional[str] = None
    """Recipient address"""

    transaction_hash: Optional[str] = None
    """Hash of the transaction"""

    value: Optional[str] = None
    """Amount of NRN transferred."""
