# File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

from __future__ import annotations

from typing import Union, Iterable
from typing_extensions import Annotated, TypedDict

from .._types import Base64FileInput
from .._utils import PropertyInfo
from .transaction_param import TransactionParam

__all__ = ["BlockSubmitParams"]


class BlockSubmitParams(TypedDict, total=False):
    block_number: int
    """Block number in the chain"""

    hash: Annotated[Union[str, Base64FileInput], PropertyInfo(format="base64")]
    """Hash of the block"""

    nonce: int
    """Nonce used for mining"""

    prev_hash: Annotated[Union[str, Base64FileInput], PropertyInfo(alias="prevHash", format="base64")]
    """Hash of the previous block"""

    proposer_address: str
    """Address of the block proposer (miner/validator)"""

    timestamp: int
    """Unix timestamp when the block was created"""

    transactions: Iterable[TransactionParam]
    """Transactions included in this block"""
