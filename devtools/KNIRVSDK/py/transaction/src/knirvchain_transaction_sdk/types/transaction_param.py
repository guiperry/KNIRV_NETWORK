# File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

from __future__ import annotations

from typing import Dict, Union, Optional
from typing_extensions import Required, Annotated, TypedDict

from .._types import Base64FileInput
from .._utils import PropertyInfo
from .._models import set_pydantic_config

__all__ = ["TransactionParam"]

_TransactionParamReservedKeywords = TypedDict(
    "_TransactionParamReservedKeywords",
    {
        "from": str,
    },
    total=False,
)


class TransactionParam(_TransactionParamReservedKeywords, total=False):
    id: Required[str]
    """Transaction hash/ID."""

    fee: Required[str]
    """NRN token gas fee paid for this transaction"""

    public_key: Required[str]
    """Public key of the sender"""

    signature: Required[Annotated[Union[str, Base64FileInput], PropertyInfo(format="base64")]]
    """Cryptographic signature"""

    timestamp: Required[int]
    """Unix timestamp (nanoseconds) when the transaction was created"""

    type: Required[str]
    """Transaction type for MCP transactions"""

    version: Required[int]

    data: Dict[str, object]
    """Transaction payload, structure depends on transaction type.

    For MCP ops, this will be JSON of MCPRegisterCapabilityData,
    MCPInvokeCapabilityData, etc.
    """

    status: str
    """Transaction status (PENDING, SUCCESS, FAILED)"""

    to: Optional[str]
    """Recipient address"""

    transaction_hash: str
    """Hash of the transaction"""

    value: str
    """Amount of NRN transferred."""


set_pydantic_config(TransactionParam, {"arbitrary_types_allowed": True})
