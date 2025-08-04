# File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

from typing import Dict, List, Optional

from .block import Block
from .._models import BaseModel
from .transaction import Transaction

__all__ = ["ChainRetrieveResponse"]


class ChainRetrieveResponse(BaseModel):
    blocks: Optional[List[Block]] = None

    chain_address: Optional[str] = None
    """The blockchain's address"""

    chain_id: Optional[str] = None
    """Unique identifier for the blockchain"""

    reflections: Optional[Dict[str, bool]] = None
    """Map of reflection URLs"""

    transaction_pool: Optional[List[Transaction]] = None
