// File generated from our OpenAPI spec by Stainless. See CONTRIBUTING.md for details.

import KnirvchainTransactionSDK from 'knirvchain-transaction-sdk';

const client = new KnirvchainTransactionSDK({
  apiKey: 'My API Key',
  baseURL: process.env['TEST_API_BASE_URL'] ?? 'http://127.0.0.1:4010',
});

describe('resource transaction', () => {
  // skipped: tests are disabled for the time being
  test.skip('submit: only required params', async () => {
    const responsePromise = client.transaction.submit({
      id: '0xabcdef123456...',
      fee: 'fee',
      from: 'from',
      public_key: 'public_key',
      signature: 'U3RhaW5sZXNzIHJvY2tz',
      timestamp: 0,
      type: 'type',
      version: 1,
    });
    const rawResponse = await responsePromise.asResponse();
    expect(rawResponse).toBeInstanceOf(Response);
    const response = await responsePromise;
    expect(response).not.toBeInstanceOf(Response);
    const dataAndResponse = await responsePromise.withResponse();
    expect(dataAndResponse.data).toBe(response);
    expect(dataAndResponse.response).toBe(rawResponse);
  });

  // skipped: tests are disabled for the time being
  test.skip('submit: required and optional params', async () => {
    const response = await client.transaction.submit({
      id: '0xabcdef123456...',
      fee: 'fee',
      from: 'from',
      public_key: 'public_key',
      signature: 'U3RhaW5sZXNzIHJvY2tz',
      timestamp: 0,
      type: 'type',
      version: 1,
      data: { foo: 'bar' },
      status: 'status',
      to: 'to',
      transaction_hash: 'transaction_hash',
      value: '1000000000000000000',
    });
  });
});
