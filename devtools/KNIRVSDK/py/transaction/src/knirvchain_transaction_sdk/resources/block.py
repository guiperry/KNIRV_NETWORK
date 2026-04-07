# File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

from __future__ import annotations

from typing import Union, Iterable

import httpx

from ..types import block_submit_params
from .._types import NOT_GIVEN, Body, Query, Headers, NotGiven, Base64FileInput
from .._utils import maybe_transform, async_maybe_transform
from .._compat import cached_property
from .._resource import SyncAPIResource, AsyncAPIResource
from .._response import (
    to_raw_response_wrapper,
    to_streamed_response_wrapper,
    async_to_raw_response_wrapper,
    async_to_streamed_response_wrapper,
)
from .._base_client import make_request_options
from ..types.transaction_param import TransactionParam
from ..types.block_submit_response import BlockSubmitResponse

__all__ = ["BlockResource", "AsyncBlockResource"]


class BlockResource(SyncAPIResource):
    @cached_property
    def with_raw_response(self) -> BlockResourceWithRawResponse:
        """
        This property can be used as a prefix for any HTTP method call to return
        the raw response object instead of the parsed content.

        For more information, see https://www.github.com/stainless-sdks/knirvchain-transaction-sdk-python#accessing-raw-response-data-eg-headers
        """
        return BlockResourceWithRawResponse(self)

    @cached_property
    def with_streaming_response(self) -> BlockResourceWithStreamingResponse:
        """
        An alternative to `.with_raw_response` that doesn't eagerly read the response body.

        For more information, see https://www.github.com/stainless-sdks/knirvchain-transaction-sdk-python#with_streaming_response
        """
        return BlockResourceWithStreamingResponse(self)

    def submit(
        self,
        *,
        block_number: int | NotGiven = NOT_GIVEN,
        hash: Union[str, Base64FileInput] | NotGiven = NOT_GIVEN,
        nonce: int | NotGiven = NOT_GIVEN,
        prev_hash: Union[str, Base64FileInput] | NotGiven = NOT_GIVEN,
        proposer_address: str | NotGiven = NOT_GIVEN,
        timestamp: int | NotGiven = NOT_GIVEN,
        transactions: Iterable[TransactionParam] | NotGiven = NOT_GIVEN,
        # Use the following arguments if you need to pass additional parameters to the API that aren't available via kwargs.
        # The extra values given here take precedence over values defined on the client or passed to this method.
        extra_headers: Headers | None = None,
        extra_query: Query | None = None,
        extra_body: Body | None = None,
        timeout: float | httpx.Timeout | None | NotGiven = NOT_GIVEN,
    ) -> BlockSubmitResponse:
        """
        Submits a new block to the blockchain

        Args:
          block_number: Block number in the chain

          hash: Hash of the block

          nonce: Nonce used for mining

          prev_hash: Hash of the previous block

          proposer_address: Address of the block proposer (miner/validator)

          timestamp: Unix timestamp when the block was created

          transactions: Transactions included in this block

          extra_headers: Send extra headers

          extra_query: Add additional query parameters to the request

          extra_body: Add additional JSON properties to the request

          timeout: Override the client-level default timeout for this request, in seconds
        """
        return self._post(
            "/block",
            body=maybe_transform(
                {
                    "block_number": block_number,
                    "hash": hash,
                    "nonce": nonce,
                    "prev_hash": prev_hash,
                    "proposer_address": proposer_address,
                    "timestamp": timestamp,
                    "transactions": transactions,
                },
                block_submit_params.BlockSubmitParams,
            ),
            options=make_request_options(
                extra_headers=extra_headers, extra_query=extra_query, extra_body=extra_body, timeout=timeout
            ),
            cast_to=BlockSubmitResponse,
        )


class AsyncBlockResource(AsyncAPIResource):
    @cached_property
    def with_raw_response(self) -> AsyncBlockResourceWithRawResponse:
        """
        This property can be used as a prefix for any HTTP method call to return
        the raw response object instead of the parsed content.

        For more information, see https://www.github.com/stainless-sdks/knirvchain-transaction-sdk-python#accessing-raw-response-data-eg-headers
        """
        return AsyncBlockResourceWithRawResponse(self)

    @cached_property
    def with_streaming_response(self) -> AsyncBlockResourceWithStreamingResponse:
        """
        An alternative to `.with_raw_response` that doesn't eagerly read the response body.

        For more information, see https://www.github.com/stainless-sdks/knirvchain-transaction-sdk-python#with_streaming_response
        """
        return AsyncBlockResourceWithStreamingResponse(self)

    async def submit(
        self,
        *,
        block_number: int | NotGiven = NOT_GIVEN,
        hash: Union[str, Base64FileInput] | NotGiven = NOT_GIVEN,
        nonce: int | NotGiven = NOT_GIVEN,
        prev_hash: Union[str, Base64FileInput] | NotGiven = NOT_GIVEN,
        proposer_address: str | NotGiven = NOT_GIVEN,
        timestamp: int | NotGiven = NOT_GIVEN,
        transactions: Iterable[TransactionParam] | NotGiven = NOT_GIVEN,
        # Use the following arguments if you need to pass additional parameters to the API that aren't available via kwargs.
        # The extra values given here take precedence over values defined on the client or passed to this method.
        extra_headers: Headers | None = None,
        extra_query: Query | None = None,
        extra_body: Body | None = None,
        timeout: float | httpx.Timeout | None | NotGiven = NOT_GIVEN,
    ) -> BlockSubmitResponse:
        """
        Submits a new block to the blockchain

        Args:
          block_number: Block number in the chain

          hash: Hash of the block

          nonce: Nonce used for mining

          prev_hash: Hash of the previous block

          proposer_address: Address of the block proposer (miner/validator)

          timestamp: Unix timestamp when the block was created

          transactions: Transactions included in this block

          extra_headers: Send extra headers

          extra_query: Add additional query parameters to the request

          extra_body: Add additional JSON properties to the request

          timeout: Override the client-level default timeout for this request, in seconds
        """
        return await self._post(
            "/block",
            body=await async_maybe_transform(
                {
                    "block_number": block_number,
                    "hash": hash,
                    "nonce": nonce,
                    "prev_hash": prev_hash,
                    "proposer_address": proposer_address,
                    "timestamp": timestamp,
                    "transactions": transactions,
                },
                block_submit_params.BlockSubmitParams,
            ),
            options=make_request_options(
                extra_headers=extra_headers, extra_query=extra_query, extra_body=extra_body, timeout=timeout
            ),
            cast_to=BlockSubmitResponse,
        )


class BlockResourceWithRawResponse:
    def __init__(self, block: BlockResource) -> None:
        self._block = block

        self.submit = to_raw_response_wrapper(
            block.submit,
        )


class AsyncBlockResourceWithRawResponse:
    def __init__(self, block: AsyncBlockResource) -> None:
        self._block = block

        self.submit = async_to_raw_response_wrapper(
            block.submit,
        )


class BlockResourceWithStreamingResponse:
    def __init__(self, block: BlockResource) -> None:
        self._block = block

        self.submit = to_streamed_response_wrapper(
            block.submit,
        )


class AsyncBlockResourceWithStreamingResponse:
    def __init__(self, block: AsyncBlockResource) -> None:
        self._block = block

        self.submit = async_to_streamed_response_wrapper(
            block.submit,
        )
