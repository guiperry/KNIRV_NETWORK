# File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

from typing import Optional

from .._models import BaseModel

__all__ = ["InfoRetrieveResponse"]


class InfoRetrieveResponse(BaseModel):
    chain_height: Optional[int] = None
    """Current blockchain height"""

    is_mining: Optional[bool] = None
    """Whether the node is currently mining"""

    node_id: Optional[str] = None
    """Unique identifier for the node"""

    peer_count: Optional[int] = None
    """Number of connected peers"""

    uptime: Optional[int] = None
    """Node uptime in seconds"""

    version: Optional[str] = None
    """Software version"""

    wallet_address: Optional[str] = None
    """Node's wallet address"""
