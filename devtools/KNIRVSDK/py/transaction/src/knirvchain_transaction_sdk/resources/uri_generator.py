# File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

from __future__ import annotations

from typing import Dict

import httpx

from ..types import uri_generator_create_params
from .._types import NOT_GIVEN, Body, Query, Headers, NotGiven
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
from ..types.uri_generator_create_response import UriGeneratorCreateResponse

__all__ = ["UriGeneratorResource", "AsyncUriGeneratorResource"]


class UriGeneratorResource(SyncAPIResource):
    @cached_property
    def with_raw_response(self) -> UriGeneratorResourceWithRawResponse:
        """
        This property can be used as a prefix for any HTTP method call to return
        the raw response object instead of the parsed content.

        For more information, see https://www.github.com/stainless-sdks/knirvchain-transaction-sdk-python#accessing-raw-response-data-eg-headers
        """
        return UriGeneratorResourceWithRawResponse(self)

    @cached_property
    def with_streaming_response(self) -> UriGeneratorResourceWithStreamingResponse:
        """
        An alternative to `.with_raw_response` that doesn't eagerly read the response body.

        For more information, see https://www.github.com/stainless-sdks/knirvchain-transaction-sdk-python#with_streaming_response
        """
        return UriGeneratorResourceWithStreamingResponse(self)

    def create(
        self,
        *,
        content_hash: str | NotGiven = NOT_GIVEN,
        metadata: Dict[str, object] | NotGiven = NOT_GIVEN,
        owner: str | NotGiven = NOT_GIVEN,
        resource_type: str | NotGiven = NOT_GIVEN,
        # Use the following arguments if you need to pass additional parameters to the API that aren't available via kwargs.
        # The extra values given here take precedence over values defined on the client or passed to this method.
        extra_headers: Headers | None = None,
        extra_query: Query | None = None,
        extra_body: Body | None = None,
        timeout: float | httpx.Timeout | None | NotGiven = NOT_GIVEN,
    ) -> UriGeneratorCreateResponse:
        """
        Generates a new URI and announces it to the network

        Args:
          content_hash: Hash of the content

          metadata: Additional metadata

          owner: Owner address

          resource_type: Type of resource

          extra_headers: Send extra headers

          extra_query: Add additional query parameters to the request

          extra_body: Add additional JSON properties to the request

          timeout: Override the client-level default timeout for this request, in seconds
        """
        return self._post(
            "/uriGenerator",
            body=maybe_transform(
                {
                    "content_hash": content_hash,
                    "metadata": metadata,
                    "owner": owner,
                    "resource_type": resource_type,
                },
                uri_generator_create_params.UriGeneratorCreateParams,
            ),
            options=make_request_options(
                extra_headers=extra_headers, extra_query=extra_query, extra_body=extra_body, timeout=timeout
            ),
            cast_to=UriGeneratorCreateResponse,
        )


class AsyncUriGeneratorResource(AsyncAPIResource):
    @cached_property
    def with_raw_response(self) -> AsyncUriGeneratorResourceWithRawResponse:
        """
        This property can be used as a prefix for any HTTP method call to return
        the raw response object instead of the parsed content.

        For more information, see https://www.github.com/stainless-sdks/knirvchain-transaction-sdk-python#accessing-raw-response-data-eg-headers
        """
        return AsyncUriGeneratorResourceWithRawResponse(self)

    @cached_property
    def with_streaming_response(self) -> AsyncUriGeneratorResourceWithStreamingResponse:
        """
        An alternative to `.with_raw_response` that doesn't eagerly read the response body.

        For more information, see https://www.github.com/stainless-sdks/knirvchain-transaction-sdk-python#with_streaming_response
        """
        return AsyncUriGeneratorResourceWithStreamingResponse(self)

    async def create(
        self,
        *,
        content_hash: str | NotGiven = NOT_GIVEN,
        metadata: Dict[str, object] | NotGiven = NOT_GIVEN,
        owner: str | NotGiven = NOT_GIVEN,
        resource_type: str | NotGiven = NOT_GIVEN,
        # Use the following arguments if you need to pass additional parameters to the API that aren't available via kwargs.
        # The extra values given here take precedence over values defined on the client or passed to this method.
        extra_headers: Headers | None = None,
        extra_query: Query | None = None,
        extra_body: Body | None = None,
        timeout: float | httpx.Timeout | None | NotGiven = NOT_GIVEN,
    ) -> UriGeneratorCreateResponse:
        """
        Generates a new URI and announces it to the network

        Args:
          content_hash: Hash of the content

          metadata: Additional metadata

          owner: Owner address

          resource_type: Type of resource

          extra_headers: Send extra headers

          extra_query: Add additional query parameters to the request

          extra_body: Add additional JSON properties to the request

          timeout: Override the client-level default timeout for this request, in seconds
        """
        return await self._post(
            "/uriGenerator",
            body=await async_maybe_transform(
                {
                    "content_hash": content_hash,
                    "metadata": metadata,
                    "owner": owner,
                    "resource_type": resource_type,
                },
                uri_generator_create_params.UriGeneratorCreateParams,
            ),
            options=make_request_options(
                extra_headers=extra_headers, extra_query=extra_query, extra_body=extra_body, timeout=timeout
            ),
            cast_to=UriGeneratorCreateResponse,
        )


class UriGeneratorResourceWithRawResponse:
    def __init__(self, uri_generator: UriGeneratorResource) -> None:
        self._uri_generator = uri_generator

        self.create = to_raw_response_wrapper(
            uri_generator.create,
        )


class AsyncUriGeneratorResourceWithRawResponse:
    def __init__(self, uri_generator: AsyncUriGeneratorResource) -> None:
        self._uri_generator = uri_generator

        self.create = async_to_raw_response_wrapper(
            uri_generator.create,
        )


class UriGeneratorResourceWithStreamingResponse:
    def __init__(self, uri_generator: UriGeneratorResource) -> None:
        self._uri_generator = uri_generator

        self.create = to_streamed_response_wrapper(
            uri_generator.create,
        )


class AsyncUriGeneratorResourceWithStreamingResponse:
    def __init__(self, uri_generator: AsyncUriGeneratorResource) -> None:
        self._uri_generator = uri_generator

        self.create = async_to_streamed_response_wrapper(
            uri_generator.create,
        )
