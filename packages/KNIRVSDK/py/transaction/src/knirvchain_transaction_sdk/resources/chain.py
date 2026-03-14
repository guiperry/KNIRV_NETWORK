# File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

from __future__ import annotations

import httpx

from .._types import NOT_GIVEN, Body, Query, Headers, NotGiven
from .._compat import cached_property
from .._resource import SyncAPIResource, AsyncAPIResource
from .._response import (
    to_raw_response_wrapper,
    to_streamed_response_wrapper,
    async_to_raw_response_wrapper,
    async_to_streamed_response_wrapper,
)
from .._base_client import make_request_options
from ..types.chain_retrieve_response import ChainRetrieveResponse

__all__ = ["ChainResource", "AsyncChainResource"]


class ChainResource(SyncAPIResource):
    @cached_property
    def with_raw_response(self) -> ChainResourceWithRawResponse:
        """
        This property can be used as a prefix for any HTTP method call to return
        the raw response object instead of the parsed content.

        For more information, see https://www.github.com/stainless-sdks/knirvchain-transaction-sdk-python#accessing-raw-response-data-eg-headers
        """
        return ChainResourceWithRawResponse(self)

    @cached_property
    def with_streaming_response(self) -> ChainResourceWithStreamingResponse:
        """
        An alternative to `.with_raw_response` that doesn't eagerly read the response body.

        For more information, see https://www.github.com/stainless-sdks/knirvchain-transaction-sdk-python#with_streaming_response
        """
        return ChainResourceWithStreamingResponse(self)

    def retrieve(
        self,
        *,
        # Use the following arguments if you need to pass additional parameters to the API that aren't available via kwargs.
        # The extra values given here take precedence over values defined on the client or passed to this method.
        extra_headers: Headers | None = None,
        extra_query: Query | None = None,
        extra_body: Body | None = None,
        timeout: float | httpx.Timeout | None | NotGiven = NOT_GIVEN,
    ) -> ChainRetrieveResponse:
        """
        Retrieves the current state of the blockchain including blocks, transaction
        pool, and reflections
        """
        return self._get(
            "/chain",
            options=make_request_options(
                extra_headers=extra_headers, extra_query=extra_query, extra_body=extra_body, timeout=timeout
            ),
            cast_to=ChainRetrieveResponse,
        )


class AsyncChainResource(AsyncAPIResource):
    @cached_property
    def with_raw_response(self) -> AsyncChainResourceWithRawResponse:
        """
        This property can be used as a prefix for any HTTP method call to return
        the raw response object instead of the parsed content.

        For more information, see https://www.github.com/stainless-sdks/knirvchain-transaction-sdk-python#accessing-raw-response-data-eg-headers
        """
        return AsyncChainResourceWithRawResponse(self)

    @cached_property
    def with_streaming_response(self) -> AsyncChainResourceWithStreamingResponse:
        """
        An alternative to `.with_raw_response` that doesn't eagerly read the response body.

        For more information, see https://www.github.com/stainless-sdks/knirvchain-transaction-sdk-python#with_streaming_response
        """
        return AsyncChainResourceWithStreamingResponse(self)

    async def retrieve(
        self,
        *,
        # Use the following arguments if you need to pass additional parameters to the API that aren't available via kwargs.
        # The extra values given here take precedence over values defined on the client or passed to this method.
        extra_headers: Headers | None = None,
        extra_query: Query | None = None,
        extra_body: Body | None = None,
        timeout: float | httpx.Timeout | None | NotGiven = NOT_GIVEN,
    ) -> ChainRetrieveResponse:
        """
        Retrieves the current state of the blockchain including blocks, transaction
        pool, and reflections
        """
        return await self._get(
            "/chain",
            options=make_request_options(
                extra_headers=extra_headers, extra_query=extra_query, extra_body=extra_body, timeout=timeout
            ),
            cast_to=ChainRetrieveResponse,
        )


class ChainResourceWithRawResponse:
    def __init__(self, chain: ChainResource) -> None:
        self._chain = chain

        self.retrieve = to_raw_response_wrapper(
            chain.retrieve,
        )


class AsyncChainResourceWithRawResponse:
    def __init__(self, chain: AsyncChainResource) -> None:
        self._chain = chain

        self.retrieve = async_to_raw_response_wrapper(
            chain.retrieve,
        )


class ChainResourceWithStreamingResponse:
    def __init__(self, chain: ChainResource) -> None:
        self._chain = chain

        self.retrieve = to_streamed_response_wrapper(
            chain.retrieve,
        )


class AsyncChainResourceWithStreamingResponse:
    def __init__(self, chain: AsyncChainResource) -> None:
        self._chain = chain

        self.retrieve = async_to_streamed_response_wrapper(
            chain.retrieve,
        )
