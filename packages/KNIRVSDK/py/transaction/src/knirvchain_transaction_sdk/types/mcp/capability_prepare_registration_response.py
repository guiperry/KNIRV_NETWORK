# File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

from typing import Optional

from pydantic import Field as FieldInfo

from ..._models import BaseModel

__all__ = ["CapabilityPrepareRegistrationResponse", "UnsignedTransaction"]


class UnsignedTransaction(BaseModel):
    data: Optional[str] = None
    """Base64 encoded JSON bytes of MCPRegisterCapabilityData."""

    fee: Optional[str] = None

    from_: Optional[str] = FieldInfo(alias="from", default=None)

    timestamp: Optional[int] = None

    to: Optional[str] = None

    type: Optional[str] = None

    unsigned_transaction_payload_hash: Optional[str] = FieldInfo(alias="unsignedTransactionPayloadHash", default=None)
    """Hash of the payload that needs to be signed."""

    value: Optional[str] = None


class CapabilityPrepareRegistrationResponse(BaseModel):
    pending_transaction_hash: str = FieldInfo(alias="pendingTransactionHash")
    """A server-side hash identifying this pending registration preparation."""

    unsigned_transaction: UnsignedTransaction = FieldInfo(alias="unsignedTransaction")
    """
    A structured representation of the unsigned transaction the client is expected
    to form and sign.
    """
