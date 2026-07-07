# File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

from __future__ import annotations

import os
from typing import Any, cast

import pytest

from tests.utils import assert_matches_type
from knirvchain_transaction_sdk import KnirvchainTransactionSDK, AsyncKnirvchainTransactionSDK
from knirvchain_transaction_sdk.types import TxnPoolRetrieveResponse

base_url = os.environ.get("TEST_API_BASE_URL", "http://127.0.0.1:4010")


class TestTxnPool:
    parametrize = pytest.mark.parametrize("client", [False, True], indirect=True, ids=["loose", "strict"])

    @pytest.mark.skip()
    @parametrize
    def test_method_retrieve(self, client: KnirvchainTransactionSDK) -> None:
        txn_pool = client.txn_pool.retrieve()
        assert_matches_type(TxnPoolRetrieveResponse, txn_pool, path=["response"])

    @pytest.mark.skip()
    @parametrize
    def test_raw_response_retrieve(self, client: KnirvchainTransactionSDK) -> None:
        response = client.txn_pool.with_raw_response.retrieve()

        assert response.is_closed is True
        assert response.http_request.headers.get("X-Stainless-Lang") == "python"
        txn_pool = response.parse()
        assert_matches_type(TxnPoolRetrieveResponse, txn_pool, path=["response"])

    @pytest.mark.skip()
    @parametrize
    def test_streaming_response_retrieve(self, client: KnirvchainTransactionSDK) -> None:
        with client.txn_pool.with_streaming_response.retrieve() as response:
            assert not response.is_closed
            assert response.http_request.headers.get("X-Stainless-Lang") == "python"

            txn_pool = response.parse()
            assert_matches_type(TxnPoolRetrieveResponse, txn_pool, path=["response"])

        assert cast(Any, response.is_closed) is True


class TestAsyncTxnPool:
    parametrize = pytest.mark.parametrize("async_client", [False, True], indirect=True, ids=["loose", "strict"])

    @pytest.mark.skip()
    @parametrize
    async def test_method_retrieve(self, async_client: AsyncKnirvchainTransactionSDK) -> None:
        txn_pool = await async_client.txn_pool.retrieve()
        assert_matches_type(TxnPoolRetrieveResponse, txn_pool, path=["response"])

    @pytest.mark.skip()
    @parametrize
    async def test_raw_response_retrieve(self, async_client: AsyncKnirvchainTransactionSDK) -> None:
        response = await async_client.txn_pool.with_raw_response.retrieve()

        assert response.is_closed is True
        assert response.http_request.headers.get("X-Stainless-Lang") == "python"
        txn_pool = await response.parse()
        assert_matches_type(TxnPoolRetrieveResponse, txn_pool, path=["response"])

    @pytest.mark.skip()
    @parametrize
    async def test_streaming_response_retrieve(self, async_client: AsyncKnirvchainTransactionSDK) -> None:
        async with async_client.txn_pool.with_streaming_response.retrieve() as response:
            assert not response.is_closed
            assert response.http_request.headers.get("X-Stainless-Lang") == "python"

            txn_pool = await response.parse()
            assert_matches_type(TxnPoolRetrieveResponse, txn_pool, path=["response"])

        assert cast(Any, response.is_closed) is True
