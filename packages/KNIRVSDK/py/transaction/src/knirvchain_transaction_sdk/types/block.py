# File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

from typing import List, Optional

from pydantic import Field as FieldInfo

from .._models import BaseModel
from .transaction import Transaction

__all__ = ["Block"]


class Block(BaseModel):
    block_number: Optional[int] = None
    """Block number in the chain"""

    hash: Optional[str] = None
    """Hash of the block"""

    nonce: Optional[int] = None
    """Nonce used for mining"""

    prev_hash: Optional[str] = FieldInfo(alias="prevHash", default=None)
    """Hash of the previous block"""

    proposer_address: Optional[str] = None
    """Address of the block proposer (miner/validator)"""

    timestamp: Optional[int] = None
    """Unix timestamp when the block was created"""

    transactions: Optional[List[Transaction]] = None
    """Transactions included in this block"""
