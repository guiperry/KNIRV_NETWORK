

---

**Source**: KNIRVROOT/docs/TestCompletions/Test_Log.md

e: 1, NumTx: 1
2025/05/20 11:25:02 [INFO] ProtoBlock fields for block #2:
2025/05/20 11:25:02 [INFO] - BlockNumber: 2
2025/05/20 11:25:02 [INFO] - PrevHash: 082f72b943b2cc209053fcfbdcb039e8c40e63490ae040242b73e8968b678053
2025/05/20 11:25:02 [INFO] - Nonce: 1
2025/05/20 11:25:02 [INFO] - Timestamp: 2025-05-20 15:25:02 +0000 UTC
2025/05/20 11:25:02 [INFO] - Transactions count: 1
2025/05/20 11:25:02 [INFO] - ProposerAddress: 
2025/05/20 11:25:02 [INFO] Block.Hash: Block #2 being converted to proto. Timestamp: 1747754702, Nonce: 2, NumTx: 1
2025/05/20 11:25:02 [INFO] ProtoBlock fields for block #2:
2025/05/20 11:25:02 [INFO] - BlockNumber: 2
2025/05/20 11:25:02 [INFO] - PrevHash: 082f72b943b2cc209053fcfbdcb039e8c40e63490ae040242b73e8968b678053
2025/05/20 11:25:02 [INFO] - Nonce: 2
2025/05/20 11:25:02 [INFO] - Timestamp: 2025-05-20 15:25:02 +0000 UTC
2025/05/20 11:25:02 [INFO] - Transactions count: 1
2025/05/20 11:25:02 [INFO] - ProposerAddress: 
2025/05/20 11:25:02 [INFO] Block.Hash: Block #2 being converted to proto. Timestamp: 1747754702, Nonce: 3, NumTx: 1
2025/05/20 11:25:02 [INFO] ProtoBlock fields for block #2:
2025/05/20 11:25:02 [INFO] - BlockNumber: 2
2025/05/20 11:25:02 [INFO] - PrevHash: 082f72b943b2cc209053fcfbdcb039e8c40e63490ae040242b73e8968b678053
2025/05/20 11:25:02 [INFO] - Nonce: 3
2025/05/20 11:25:02 [INFO] - Timestamp: 2025-05-20 15:25:02 +0000 UTC
2025/05/20 11:25:02 [INFO] - Transactions count: 1
2025/05/20 11:25:02 [INFO] - ProposerAddress: 
2025/05/20 11:25:02 [INFO] Block.Hash: Block #2 being converted to proto. Timestamp: 1747754702, Nonce: 4, NumTx: 1
2025/05/20 11:25:02 [INFO] ProtoBlock fields for block #2:
2025/05/20 11:25:02 [INFO] - BlockNumber: 2
2025/05/20 11:25:02 [INFO] - PrevHash: 082f72b943b2cc209053fcfbdcb039e8c40e63490ae040242b73e8968b678053
2025/05/20 11:25:02 [INFO] - Nonce: 4
2025/05/20 11:25:02 [INFO] - Timestamp: 2025-05-20 15:25:02 +0000 UTC
2025/05/20 11:25:02 [INFO] - Transactions count: 1
2025/05/20 11:25:02 [INFO] - ProposerAddress: 
2025/05/20 11:25:02 [INFO] Block.Hash: Block #2 being converted to proto. Timestamp: 1747754702, Nonce: 5, NumTx: 1
2025/05/20 11:25:02 [INFO] ProtoBlock fields for block #2:
2025/05/20 11:25:02 [INFO] - BlockNumber: 2
2025/05/20 11:25:02 [INFO] - PrevHash: 082f72b943b2cc209053fcfbdcb039e8c40e63490ae040242b73e8968b678053
2025/05/20 11:25:02 [INFO] - Nonce: 5
2025/05/20 11:25:02 [INFO] - Timestamp: 2025-05-20 15:25:02 +0000 UTC
2025/05/20 11:25:02 [INFO] - Transactions count: 1
2025/05/20 11:25:02 [INFO] - ProposerAddress: 
2025/05/20 11:25:02 [INFO] Attempting to add block number 2 (Hash: 0423fb4ce8f36cc159a56a249810314525a10665350a19f75a9a51d4972f65c9)
2025/05/20 11:25:02 [INFO] AddBlock: Current chain length before add: 2
2025/05/20 11:25:02 [INFO] Block.Hash: Block #2 being converted to proto. Timestamp: 1747754702, Nonce: 5, NumTx: 1
2025/05/20 11:25:02 [INFO] ProtoBlock fields for block #2:
2025/05/20 11:25:02 [INFO] - BlockNumber: 2
2025/05/20 11:25:02 [INFO] - PrevHash: 082f72b943b2cc209053fcfbdcb039e8c40e63490ae040242b73e8968b678053
2025/05/20 11:25:02 [INFO] - Nonce: 5
2025/05/20 11:25:02 [INFO] - Timestamp: 2025-05-20 15:25:02 +0000 UTC
2025/05/20 11:25:02 [INFO] - Transactions count: 1
2025/05/20 11:25:02 [INFO] - ProposerAddress: KNIRVCHAINb53c1e30b8a578c091dd40612bfd1433991b4e09
2025/05/20 11:25:02 [ERROR] VerifyBlock failed for block 2: Calculated hash 6943ba4d52664fc284ce591bbab7defc3d7ef30ab967bb514aa061971f06b77c does not match stored BlockHash 0423fb4ce8f36cc159a56a249810314525a10665350a19f75a9a51d4972f65c9: <nil>
2025/05/20 11:25:02 [ERROR] Block 2 verification failed, not adding.: <nil>
2025/05/20 11:25:02 [DEBUG] GetCapabilityByID: Attempting to get capability with key: mcp:capability:resource-123
2025/05/20 11:25:02 [DEBUG] GetCapabilityByID: Key mcp:capability:resource-123 not found in DB.
    mcp_blockchain_test.go:98: Failed to get capability from database: failed to get capability: key not found: leveldb: not found
--- FAIL: TestMCPTransactionProcessing (1.13s)
=== RUN   TestMCPCapabilityUpdate
2025/05/20 11:25:02 Successfully opened LevelDB at path: testdb_1747754702604918483
2025/05/20 11:25:02 [INFO] No existing blockchain data found for key 'test_chain_1747754702888109223'. Creating new blockchain.
2025/05/20 11:25:02 [INFO] Creating new blockchain for test_chain_1747754702888109223 using provided Genesis block.
2025/05/20 11:25:02 [INFO] Successfully created and stored new blockchain for key 'test_chain_1747754702888109223'.
2025/05/20 11:25:02 Verification hash input: 
2KNIRVCHAINdd14e2626e1cc72fc6b5c391cbc11ad171848501"�{"capabilityDescriptor":{"id":"resource-123","capabilityType":"RESOURCE","name":"Test Resource","owner":"KNIRVCHAINdd14e2626e1cc72fc6b5c391cbc11ad171848501","version":"1.0.0","description":"Test resource for unit testing","gasFeeNRN":100,"timestamp":1747754702,"customMetadata":{"key1":"value1"},"resourceType":"FILE","contentHash":"sha256:1234567890abcdef","schema":{"summary":"This is resource 2 (API)","locationHints":["http://api.example.com/res2"]}}}ν��0d:register_capability_txn
2025/05/20 11:25:02 Verification hash bytes (hex): 0a326b6e697276636861696e6464313465323632366531636337326663366235633339316362633131616431373138343835303122c7037b226361706162696c69747944657363726970746f72223a7b226964223a227265736f757263652d313233222c226361706162696c69747954797065223a225245534f55524345222c226e616d65223a2254657374205265736f75726365222c226f776e6572223a226b6e697276636861696e64643134653236323665316363373266633662356333393163626331316164313731383438353031222c2276657273696f6e223a22312e302e30222c226465736372697074696f6e223a2254657374207265736f7572636520666f7220756e69742074657374696e67222c226761734665654e524e223a3130302c2274696d657374616d70223a313734373735343730322c22637573746f6d4d65746164617461223a7b226b657931223a2276616c756531227d2c227265736f7572636554797065223a2246494c45222c22636f6e74656e7448617368223a227368613235363a31323334353637383930616263646566222c22736368656d61223a7b2273756d6d617279223a2254686973206973207265736f757263652032202841504929222c226c6f636174696f6e48696e7473223a5b22687474703a2f2f6170692e6578616d706c652e636f6d2f72657332225d7d7d7d2a0608cebdb2c10630643a1772656769737465725f6361706162696c6974795f74786e
2025/05/20 11:25:02 Signature bytes (hex): 3046022100d044d9c6c429f51f3f0081f06eefd19fe2dfc3bba1a22172066505feb5530b40022100f2d0d361238ca4902cd5f92205a9f60e0a94147f8ff66620028f015076b56dee
2025/05/20 11:25:02 Verification hash input: 
2KNIRVCHAINdd14e2626e1cc72fc6b5c391cbc11ad171848501"�{"capabilityDescriptor":{"id":"resource-123","capabilityType":"RESOURCE","name":"Test Resource","owner":"KNIRVCHAINdd14e2626e1cc72fc6b5c391cbc11ad171848501","version":"1.0.0","description":"Test resource for unit testing","gasFeeNRN":100,"timestamp":1747754702,"customMetadata":{"key1":"value1"},"resourceType":"FILE","contentHash":"sha256:1234567890abcdef","schema":{"summary":"This is resource 2 (API)","locationHints":["http://api.example.com/res2"]}}}ν��0d:register_capability_txn
2025/05/20 11:25:02 Verification hash bytes (hex): 0a326b6e697276636861696e6464313465323632366531636337326663366235633339316362633131616431373138343835303122c7037b226361706162696c69747944657363726970746f72223a7b226964223a227265736f757263652d313233222c226361706162696c69747954797065223a225245534f55524345222c226e616d65223a2254657374205265736f75726365222c226f776e6572223a226b6e697276636861696e64643134653236323665316363373266633662356333393163626331316164313731383438353031222c2276657273696f6e223a22312e302e30222c226465736372697074696f6e223a2254657374207265736f7572636520666f7220756e69742074657374696e67222c226761734665654e524e223a3130302c2274696d657374616d70223a313734373735343730322c22637573746f6d4d65746164617461223a7b226b657931223a2276616c756531227d2c227265736f7572636554797065223a2246494c45222c22636f6e74656e7448617368223a227368613235363a31323334353637383930616263646566222c22736368656d61223a7b2273756d6d617279223a2254686973206973207265736f757263652032202841504929222c226c6f636174696f6e48696e7473223a5b22687474703a2f2f6170692e6578616d706c652e636f6d2f72657332225d7d7d7d2a0608cebdb2c10630643a1772656769737465725f6361706162696c6974795f74786e
2025/05/20 11:25:02 Signature bytes (hex): 3046022100d044d9c6c429f51f3f0081f06eefd19fe2dfc3bba1a22172066505feb5530b40022100f2d0d361238ca4902cd5f92205a9f60e0a94147f8ff66620028f015076b56dee
2025/05/20 11:25:03 [DEBUG] GetCapabilityByID: Attempting to get capability with key: mcp:capability:resource-123
2025/05/20 11:25:03 [DEBUG] GetCapabilityByID: Key mcp:capability:resource-123 not found in DB.
2025/05/20 11:25:03 [INFO] calculateTotalCryptoLocked for KNIRVCHAINdd14e2626e1cc72fc6b5c391cbc11ad171848501: Iterating 1 blocks...
2025/05/20 11:25:03 [INFO]   Block 0: Processing...
2025/05/20 11:25:03 [INFO]   Block 0: Contains 0 transactions.
2025/05/20 11:25:03 [INFO] calculateTotalCryptoLocked for KNIRVCHAINdd14e2626e1cc72fc6b5c391cbc11ad171848501 finished. Final calculated balance: 0
2025/05/20 11:25:03 [INFO] Simulated Balance Check for KNIRVCHAINdd14e2626e1cc72fc6b5c391cbc11ad171848501 ->  (Value: 0, Fee: 100): Initial on-chain balance for KNIRVCHAINdd14e2626e1cc72fc6b5c391cbc11ad171848501 = 0
2025/05/20 11:25:03 [INFO]   = Final Check: Final Simulated Balance (0) >= New Txn Total Cost (Value: 0 + Fee: 100 = 100) -> false
2025/05/20 11:25:03 [INFO] Block.Hash: Block #0 being converted to proto. Timestamp: 1747754702, Nonce: 0, NumTx: 0
2025/05/20 11:25:03 [INFO] ProtoBlock fields for block #0:
2025/05/20 11:25:03 [INFO] - BlockNumber: 0
2025/05/20 11:25:03 [INFO] - PrevHash: 
2025/05/20 11:25:03 [INFO] - Nonce: 0
2025/05/20 11:25:03 [INFO] - Timestamp: 2025-05-20 15:25:02 +0000 UTC
2025/05/20 11:25:03 [INFO] - Transactions count: 0
2025/05/20 11:25:03 [INFO] - ProposerAddress: 
2025/05/20 11:25:03 [INFO] Block.Hash: Block #1 being converted to proto. Timestamp: 1747754703, Nonce: 1, NumTx: 1
2025/05/20 11:25:03 [INFO] ProtoBlock fields for block #1:
2025/05/20 11:25:03 [INFO] - BlockNumber: 1
2025/05/20 11:25:03 [INFO] - PrevHash: 91ef7e71bce1ff1e6b74c8da6c23675ea1a5cef89e47a289e531d80f73dd03ad
2025/05/20 11:25:03 [INFO] - Nonce: 1
2025/05/20 11:25:03 [INFO] - Timestamp: 2025-05-20 15:25:03 +0000 UTC
2025/05/20 11:25:03 [INFO] - Transactions count: 1
2025/05/20 11:25:03 [INFO] - ProposerAddress: 
2025/05/20 11:25:03 [INFO] Attempting to add block number 1 (Hash: 08475f34266570ec3b947f3608c253a1cb7f6af3e098255a944faba403bc0f25)
2025/05/20 11:25:03 [INFO] AddBlock: Current chain length before add: 1
2025/05/20 11:25:03 [INFO] Block.Hash: Block #1 being converted to proto. Timestamp: 1747754703, Nonce: 1, NumTx: 1
2025/05/20 11:25:03 [INFO] ProtoBlock fields for block #1:
2025/05/20 11:25:03 [INFO] - BlockNumber: 1
2025/05/20 11:25:03 [INFO] - PrevHash: 91ef7e71bce1ff1e6b74c8da6c23675ea1a5cef89e47a289e531d80f73dd03ad
2025/05/20 11:25:03 [INFO] - Nonce: 1
2025/05/20 11:25:03 [INFO] - Timestamp: 2025-05-20 15:25:03 +0000 UTC
2025/05/20 11:25:03 [INFO] - Transactions count: 1
2025/05/20 11:25:03 [INFO] - ProposerAddress: 
2025/05/20 11:25:03 Verification hash input: 
2KNIRVCHAINdd14e2626e1cc72fc6b5c391cbc11ad171848501"�{"capabilityDescriptor":{"id":"resource-123","capabilityType":"RESOURCE","name":"Test Resource","owner":"KNIRVCHAINdd14e2626e1cc72fc6b5c391cbc11ad171848501","version":"1.0.0","description":"Test resource for unit testing","gasFeeNRN":100,"timestamp":1747754702,"customMetadata":{"key1":"value1"},"resourceType":"FILE","contentHash":"sha256:1234567890abcdef","schema":{"summary":"This is resource 2 (API)","locationHints":["http://api.example.com/res2"]}}}ν��0d:register_capability_txn
2025/05/20 11:25:03 Verification hash bytes (hex): 0a326b6e697276636861696e6464313465323632366531636337326663366235633339316362633131616431373138343835303122c7037b226361706162696c69747944657363726970746f72223a7b226964223a227265736f757263652d313233222c226361706162696c69747954797065223a225245534f55524345222c226e616d65223a2254657374205265736f75726365222c226f776e6572223a226b6e697276636861696e64643134653236323665316363373266633662356333393163626331316164313731383438353031222c2276657273696f6e223a22312e302e30222c226465736372697074696f6e223a2254657374207265736f7572636520666f7220756e69742074657374696e67222c226761734665654e524e223a3130302c2274696d657374616d70223a313734373735343730322c22637573746f6d4d65746164617461223a7b226b657931223a2276616c756531227d2c227265736f7572636554797065223a2246494c45222c22636f6e74656e7448617368223a227368613235363a31323334353637383930616263646566222c22736368656d61223a7b2273756d6d617279223a2254686973206973207265736f757263652032202841504929222c226c6f636174696f6e48696e7473223a5b22687474703a2f2f6170692e6578616d706c652e636f6d2f72657332225d7d7d7d2a0608cebdb2c10630643a1772656769737465725f6361706162696c6974795f74786e
2025/05/20 11:25:03 Signature bytes (hex): 3046022100d044d9c6c429f51f3f0081f06eefd19fe2dfc3bba1a22172066505feb5530b40022100f2d0d361238ca4902cd5f92205a9f60e0a94147f8ff66620028f015076b56dee
2025/05/20 11:25:03 [INFO] Block.Hash: Block #0 being converted to proto. Timestamp: 1747754702, Nonce: 0, NumTx: 0
2025/05/20 11:25:03 [INFO] ProtoBlock fields for block #0:
2025/05/20 11:25:03 [INFO] - BlockNumber: 0
2025/05/20 11:25:03 [INFO] - PrevHash: 
2025/05/20 11:25:03 [INFO] - Nonce: 0
2025/05/20 11:25:03 [INFO] - Timestamp: 2025-05-20 15:25:02 +0000 UTC
2025/05/20 11:25:03 [INFO] - Transactions count: 0
2025/05/20 11:25:03 [INFO] - ProposerAddress: 
2025/05/20 11:25:03 [INFO] Checks passed for block 1. Proceeding to process transactions and update state.
2025/05/20 11:25:03 [INFO] Pre-loaded balances for 1 affected accounts for block 1.
2025/05/20 11:25:03 [INFO] Processing tx 1/1 (Hash: 0xa3ff63b2adfdb49c5af77922958aa25a54a9045501c76b114b7f7171ea48a339) in block 1
2025/05/20 11:25:03 [ERROR] Invalid transaction 0xa3ff63b2adfdb49c5af77922958aa25a54a9045501c76b114b7f7171ea48a339 in block 1 during AddBlock: insufficient balance for network fee for tx 0xa3ff63b2adfdb49c5af77922958aa25a54a9045501c76b114b7f7171ea48a339. From: KNIRVCHAINdd14e2626e1cc72fc6b5c391cbc11ad171848501, Balance: 0, Fee: 100: <nil>
2025/05/20 11:25:03 Verification hash input: 
2KNIRVCHAINdd14e2626e1cc72fc6b5c391cbc11ad171848501"�{"capabilityDescriptor":{"id":"resource-123","capabilityType":"RESOURCE","name":"Updated Test Resource","owner":"KNIRVCHAINdd14e2626e1cc72fc6b5c391cbc11ad171848501","version":"1.1.0","description":"Updated test resource for unit testing","gasFeeNRN":200,"timestamp":1747754702,"customMetadata":{"key1":"updated-value1","key2":"value2"},"resourceType":"FILE","contentHash":"sha256:1234567890abcdef","schema":{"summary":"This is resource 2 (API)","locationHints":["http://api.example.com/res2"]}},"capabilityID":"resource-123","capabilityType":"RESOURCE"}Ͻ��0d:update_capability_txn
2025/05/20 11:25:03 Verification hash bytes (hex): 0a326b6e697276636861696e6464313465323632366531636337326663366235633339316362633131616431373138343835303122a9047b226361706162696c69747944657363726970746f72223a7b226964223a227265736f757263652d313233222c226361706162696c69747954797065223a225245534f55524345222c226e616d65223a22557064617465642054657374205265736f75726365222c226f776e6572223a226b6e697276636861696e64643134653236323665316363373266633662356333393163626331316164313731383438353031222c2276657273696f6e223a22312e312e30222c226465736372697074696f6e223a22557064617465642074657374207265736f7572636520666f7220756e69742074657374696e67222c226761734665654e524e223a3230302c2274696d657374616d70223a313734373735343730322c22637573746f6d4d65746164617461223a7b226b657931223a22757064617465642d76616c756531222c226b657932223a2276616c756532227d2c227265736f7572636554797065223a2246494c45222c22636f6e74656e7448617368223a227368613235363a31323334353637383930616263646566222c22736368656d61223a7b2273756d6d617279223a2254686973206973207265736f757263652032202841504929222c226c6f636174696f6e48696e7473223a5b22687474703a2f2f6170692e6578616d706c652e636f6d2f72657332225d7d7d2c226361706162696c6974794944223a227265736f757263652d313233222c226361706162696c69747954797065223a225245534f55524345227d2a0608cfbdb2c10630643a157570646174655f6361706162696c6974795f74786e
2025/05/20 11:25:03 Signature bytes (hex): 304502202c92feb8aa767be7fa6b879477250160b2983ad7b2c62462769e335a64a1a570022100e5250bbb43e760ed4822173693871f37d8381aa0fc16ba6a2fac127ff2524ed8
2025/05/20 11:25:03 Verification hash input: 
2KNIRVCHAINdd14e2626e1cc72fc6b5c391cbc11ad171848501"�{"capabilityDescriptor":{"id":"resource-123","capabilityType":"RESOURCE","name":"Updated Test Resource","owner":"KNIRVCHAINdd14e2626e1cc72fc6b5c391cbc11ad171848501","version":"1.1.0","description":"Updated test resource for unit testing","gasFeeNRN":200,"timestamp":1747754702,"customMetadata":{"key1":"updated-value1","key2":"value2"},"resourceType":"FILE","contentHash":"sha256:1234567890abcdef","schema":{"summary":"This is resource 2 (API)","locationHints":["http://api.example.com/res2"]}},"capabilityID":"resource-123","capabilityType":"RESOURCE"}Ͻ��0d:update_capability_txn
2025/05/20 11:25:03 Verification hash bytes (hex): 0a326b6e697276636861696e6464313465323632366531636337326663366235633339316362633131616431373138343835303122a9047b226361706162696c69747944657363726970746f72223a7b226964223a227265736f757263652d313233222c226361706162696c69747954797065223a225245534f55524345222c226e616d65223a22557064617465642054657374205265736f75726365222c226f776e6572223a226b6e697276636861696e64643134653236323665316363373266633662356333393163626331316164313731383438353031222c2276657273696f6e223a22312e312e30222c226465736372697074696f6e223a22557064617465642074657374207265736f7572636520666f7220756e69742074657374696e67222c226761734665654e524e223a3230302c2274696d657374616d70223a313734373735343730322c22637573746f6d4d65746164617461223a7b226b657931223a22757064617465642d76616c756531222c226b657932223a2276616c756532227d2c227265736f7572636554797065223a2246494c45222c22636f6e74656e7448617368223a227368613235363a31323334353637383930616263646566222c22736368656d61223a7b2273756d6d617279223a2254686973206973207265736f757263652032202841504929222c226c6f636174696f6e48696e7473223a5b22687474703a2f2f6170692e6578616d706c652e636f6d2f72657332225d7d7d2c226361706162696c6974794944223a227265736f757263652d313233222c226361706162696c69747954797065223a225245534f55524345227d2a0608cfbdb2c10630643a157570646174655f6361706162696c6974795f74786e
2025/05/20 11:25:03 Signature bytes (hex): 304502202c92feb8aa767be7fa6b879477250160b2983ad7b2c62462769e335a64a1a570022100e5250bbb43e760ed4822173693871f37d8381aa0fc16ba6a2fac127ff2524ed8
2025/05/20 11:25:03 [DEBUG] GetCapabilityByID: Attempting to get capability with key: mcp:capability:resource-123
2025/05/20 11:25:03 [DEBUG] GetCapabilityByID: Key mcp:capability:resource-123 not found in DB.
2025/05/20 11:25:03 [ERROR] Failed to get capability:: failed to get capability: failed to get capability: key not found: leveldb: not found
2025/05/20 11:25:03 [INFO] Block.Hash: Block #0 being converted to proto. Timestamp: 1747754702, Nonce: 0, NumTx: 0
2025/05/20 11:25:03 [INFO] ProtoBlock fields for block #0:
2025/05/20 11:25:03 [INFO] - BlockNumber: 0
2025/05/20 11:25:03 [INFO] - PrevHash: 
2025/05/20 11:25:03 [INFO] - Nonce: 0
2025/05/20 11:25:03 [INFO] - Timestamp: 2025-05-20 15:25:02 +0000 UTC
2025/05/20 11:25:03 [INFO] - Transactions count: 0
2025/05/20 11:25:03 [INFO] - ProposerAddress: 
2025/05/20 11:25:03 [INFO] Block.Hash: Block #1 being converted to proto. Timestamp: 1747754703, Nonce: 1, NumTx: 1
2025/05/20 11:25:03 [INFO] ProtoBlock fields for block #1:
2025/05/20 11:25:03 [INFO] - BlockNumber: 1
2025/05/20 11:25:03 [INFO] - PrevHash: 91ef7e71bce1ff1e6b74c8da6c23675ea1a5cef89e47a289e531d80f73dd03ad
2025/05/20 11:25:03 [INFO] - Nonce: 1
2025/05/20 11:25:03 [INFO] - Timestamp: 2025-05-20 15:25:03 +0000 UTC
2025/05/20 11:25:03 [INFO] - Transactions count: 1
2025/05/20 11:25:03 [INFO] - ProposerAddress: 
2025/05/20 11:25:03 [INFO] Block.Hash: Block #1 being converted to proto. Timestamp: 1747754703, Nonce: 1, NumTx: 1
2025/05/20 11:25:03 [INFO] ProtoBlock fields for block #1:
2025/05/20 11:25:03 [INFO] - BlockNumber: 1
2025/05/20 11:25:03 [INFO] - PrevHash: 91ef7e71bce1ff1e6b74c8da6c23675ea1a5cef89e47a289e531d80f73dd03ad
2025/05/20 11:25:03 [INFO] - Nonce: 1
2025/05/20 11:25:03 [INFO] - Timestamp: 2025-05-20 15:25:03 +0000 UTC
2025/05/20 11:25:03 [INFO] - Transactions count: 1
2025/05/20 11:25:03 [INFO] - ProposerAddress: 
2025/05/20 11:25:03 [INFO] Block.Hash: Block #1 being converted to proto. Timestamp: 1747754703, Nonce: 2, NumTx: 1
2025/05/20 11:25:03 [INFO] ProtoBlock fields for block #1:
2025/05/20 11:25:03 [INFO] - BlockNumber: 1
2025/05/20 11:25:03 [INFO] - PrevHash: 91ef7e71bce1ff1e6b74c8da6c23675ea1a5cef89e47a289e531d80f73dd03ad
2025/05/20 11:25:03 [INFO] - Nonce: 2
2025/05/20 11:25:03 [INFO] - Timestamp: 2025-05-20 15:25:03 +0000 UTC
2025/05/20 11:25:03 [INFO] - Transactions count: 1
2025/05/20 11:25:03 [INFO] - ProposerAddress: 
2025/05/20 11:25:03 [INFO] Block.Hash: Block #1 being converted to proto. Timestamp: 1747754703, Nonce: 3, NumTx: 1
2025/05/20 11:25:03 [INFO] ProtoBlock fields for block #1:
2025/05/20 11:25:03 [INFO] - BlockNumber: 1
2025/05/20 11:25:03 [INFO] - PrevHash: 91ef7e71bce1ff1e6b74c8da6c23675ea1a5cef89e47a289e531d80f73dd03ad
2025/05/20 11:25:03 [INFO] - Nonce: 3
2025/05/20 11:25:03 [INFO] - Timestamp: 2025-05-20 15:25:03 +0000 UTC
2025/05/20 11:25:03 [INFO] - Transactions count: 1
2025/05/20 11:25:03 [INFO] - ProposerAddress: 
2025/05/20 11:25:03 [INFO] Block.Hash: Block #1 being converted to proto. Timestamp: 1747754703, Nonce: 4, NumTx: 1
2025/05/20 11:25:03 [INFO] ProtoBlock fields for block #1:
2025/05/20 11:25:03 [INFO] - BlockNumber: 1
2025/05/20 11:25:03 [INFO] - PrevHash: 91ef7e71bce1ff1e6b74c8da6c23675ea1a5cef89e47a289e531d80f73dd03ad
2025/05/20 11:25:03 [INFO] - Nonce: 4
2025/05/20 11:25:03 [INFO] - Timestamp: 2025-05-20 15:25:03 +0000 UTC
2025/05/20 11:25:03 [INFO] - Transactions count: 1
2025/05/20 11:25:03 [INFO] - ProposerAddress: 
2025/05/20 11:25:03 [INFO] Block.Hash: Block #1 being converted to proto. Timestamp: 1747754703, Nonce: 5, NumTx: 1
2025/05/20 11:25:03 [INFO] ProtoBlock fields for block #1:
2025/05/20 11:25:03 [INFO] - BlockNumber: 1
2025/05/20 11:25:03 [INFO] - PrevHash: 91ef7e71bce1ff1e6b74c8da6c23675ea1a5cef89e47a289e531d80f73dd03ad
2025/05/20 11:25:03 [INFO] - Nonce: 5
2025/05/20 11:25:03 [INFO] - Timestamp: 2025-05-20 15:25:03 +0000 UTC
2025/05/20 11:25:03 [INFO] - Transactions count: 1
2025/05/20 11:25:03 [INFO] - ProposerAddress: 
2025/05/20 11:25:03 [INFO] Block.Hash: Block #1 being converted to proto. Timestamp: 1747754703, Nonce: 6, NumTx: 1
2025/05/20 11:25:03 [INFO] ProtoBlock fields for block #1:
2025/05/20 11:25:03 [INFO] - BlockNumber: 1
2025/05/20 11:25:03 [INFO] - PrevHash: 91ef7e71bce1ff1e6b74c8da6c23675ea1a5cef89e47a289e531d80f73dd03ad
2025/05/20 11:25:03 [INFO] - Nonce: 6
2025/05/20 11:25:03 [INFO] - Timestamp: 2025-05-20 15:25:03 +0000 UTC
2025/05/20 11:25:03 [INFO] - Transactions count: 1
2025/05/20 11:25:03 [INFO] - ProposerAddress: 
2025/05/20 11:25:03 [INFO] Attempting to add block number 1 (Hash: 065c32f66684583bbbe065dc5fa9fe32bac77142380d89db99161f0f7cc7fe8d)
2025/05/20 11:25:03 [INFO] AddBlock: Current chain length before add: 1
2025/05/20 11:25:03 [INFO] Block.Hash: Block #1 being converted to proto. Timestamp: 1747754703, Nonce: 6, NumTx: 1
2025/05/20 11:25:03 [INFO] ProtoBlock fields for block #1:
2025/05/20 11:25:03 [INFO] - BlockNumber: 1
2025/05/20 11:25:03 [INFO] - PrevHash: 91ef7e71bce1ff1e6b74c8da6c23675ea1a5cef89e47a289e531d80f73dd03ad
2025/05/20 11:25:03 [INFO] - Nonce: 6
2025/05/20 11:25:03 [INFO] - Timestamp: 2025-05-20 15:25:03 +0000 UTC
2025/05/20 11:25:03 [INFO] - Transactions count: 1
2025/05/20 11:25:03 [INFO] - ProposerAddress: 
2025/05/20 11:25:03 Verification hash input: 
2KNIRVCHAINdd14e2626e1cc72fc6b5c391cbc11ad171848501"�{"capabilityDescriptor":{"id":"resource-123","capabilityType":"RESOURCE","name":"Updated Test Resource","owner":"KNIRVCHAINdd14e2626e1cc72fc6b5c391cbc11ad171848501","version":"1.1.0","description":"Updated test resource for unit testing","gasFeeNRN":200,"timestamp":1747754702,"customMetadata":{"key1":"updated-value1","key2":"value2"},"resourceType":"FILE","contentHash":"sha256:1234567890abcdef","schema":{"summary":"This is resource 2 (API)","locationHints":["http://api.example.com/res2"]}},"capabilityID":"resource-123","capabilityType":"RESOURCE"}Ͻ��0d:update_capability_txn
2025/05/20 11:25:03 Verification hash bytes (hex): 0a326b6e697276636861696e6464313465323632366531636337326663366235633339316362633131616431373138343835303122a9047b226361706162696c69747944657363726970746f72223a7b226964223a227265736f757263652d313233222c226361706162696c69747954797065223a225245534f55524345222c226e616d65223a22557064617465642054657374205265736f75726365222c226f776e6572223a226b6e697276636861696e64643134653236323665316363373266633662356333393163626331316164313731383438353031222c2276657273696f6e223a22312e312e30222c226465736372697074696f6e223a22557064617465642074657374207265736f7572636520666f7220756e69742074657374696e67222c226761734665654e524e223a3230302c2274696d657374616d70223a313734373735343730322c22637573746f6d4d65746164617461223a7b226b657931223a22757064617465642d76616c756531222c226b657932223a2276616c756532227d2c227265736f7572636554797065223a2246494c45222c22636f6e74656e7448617368223a227368613235363a31323334353637383930616263646566222c22736368656d61223a7b2273756d6d617279223a2254686973206973207265736f757263652032202841504929222c226c6f636174696f6e48696e7473223a5b22687474703a2f2f6170692e6578616d706c652e636f6d2f72657332225d7d7d2c226361706162696c6974794944223a227265736f757263652d313233222c226361706162696c69747954797065223a225245534f55524345227d2a0608cfbdb2c10630643a157570646174655f6361706162696c6974795f74786e
2025/05/20 11:25:03 Signature bytes (hex): 304502202c92feb8aa767be7fa6b879477250160b2983ad7b2c62462769e335a64a1a570022100e5250bbb43e760ed4822173693871f37d8381aa0fc16ba6a2fac127ff2524ed8
2025/05/20 11:25:03 [INFO] Block.Hash: Block #0 being converted to proto. Timestamp: 1747754702, Nonce: 0, NumTx: 0
2025/05/20 11:25:03 [INFO] ProtoBlock fields for block #0:
2025/05/20 11:25:03 [INFO] - BlockNumber: 0
2025/05/20 11:25:03 [INFO] - PrevHash: 
2025/05/20 11:25:03 [INFO] - Nonce: 0
2025/05/20 11:25:03 [INFO] - Timestamp: 2025-05-20 15:25:02 +0000 UTC
2025/05/20 11:25:03 [INFO] - Transactions count: 0
2025/05/20 11:25:03 [INFO] - ProposerAddress: 
2025/05/20 11:25:03 [INFO] Checks passed for block 1. Proceeding to process transactions and update state.
2025/05/20 11:25:03 [INFO] Pre-loaded balances for 1 affected accounts for block 1.
2025/05/20 11:25:03 [INFO] Processing tx 1/1 (Hash: 0x0b767d27803267ae069dad7de7ba21e8becfdc1246600d609e7d7f128575b3e4) in block 1
2025/05/20 11:25:03 [ERROR] Invalid transaction 0x0b767d27803267ae069dad7de7ba21e8becfdc1246600d609e7d7f128575b3e4 in block 1 during AddBlock: insufficient balance for network fee for tx 0x0b767d27803267ae069dad7de7ba21e8becfdc1246600d609e7d7f128575b3e4. From: KNIRVCHAINdd14e2626e1cc72fc6b5c391cbc11ad171848501, Balance: 0, Fee: 100: <nil>
2025/05/20 11:25:03 [DEBUG] GetCapabilityByID: Attempting to get capability with key: mcp:capability:resource-123
2025/05/20 11:25:03 [DEBUG] GetCapabilityByID: Key mcp:capability:resource-123 not found in DB.
    mcp_blockchain_test.go:338: Failed to get capability from database: failed to get capability: key not found: leveldb: not found
--- FAIL: TestMCPCapabilityUpdate (0.40s)
=== RUN   TestMCPTransactionCreation
--- PASS: TestMCPTransactionCreation (0.00s)
=== RUN   TestMCPTransactionVerification
2025/05/20 11:25:03 Verification hash input: 
2KNIRVCHAIN16c41c5c8516e653f824f857cae3f9b6da291289recipient-address-123"�{"capabilityDescriptor":{"id":"resource-123","capabilityType":"RESOURCE","name":"Test Resource","owner":"KNIRVCHAIN16c41c5c8516e653f824f857cae3f9b6da291289","version":"1.0.0","description":"Test resource for unit testing","gasFeeNRN":100,"timestamp":0,"customMetadata":{"key1":"value1"},"resourceType":"FILE","contentHash":"sha256:1234567890abcdef","schema":{"summary":"This is resource 2 (API)","locationHints":["http://api.example.com/res2"]}}}tamperedϽ��0d:register_capability_txn
2025/05/20 11:25:03 Verification hash bytes (hex): 0a326b6e697276636861696e313663343163356338353136653635336638323466383537636165336639623664613239313238391215726563697069656e742d616464726573732d31323322c6037b226361706162696c69747944657363726970746f72223a7b226964223a227265736f757263652d313233222c226361706162696c69747954797065223a225245534f55524345222c226e616d65223a2254657374205265736f75726365222c226f776e6572223a226b6e697276636861696e31366334316335633835313665363533663832346638353763616533663962366461323931323839222c2276657273696f6e223a22312e302e30222c226465736372697074696f6e223a2254657374207265736f7572636520666f7220756e69742074657374696e67222c226761734665654e524e223a3130302c2274696d657374616d70223a302c22637573746f6d4d65746164617461223a7b226b657931223a2276616c756531227d2c227265736f7572636554797065223a2246494c45222c22636f6e74656e7448617368223a227368613235363a31323334353637383930616263646566222c22736368656d61223a7b2273756d6d617279223a2254686973206973207265736f757263652032202841504929222c226c6f636174696f6e48696e7473223a5b22687474703a2f2f6170692e6578616d706c652e636f6d2f72657332225d7d7d7d74616d70657265642a0608cfbdb2c10630643a1772656769737465725f6361706162696c6974795f74786e
2025/05/20 11:25:03 Signature bytes (hex): 3046022100f60fcb9b0b77c6d6567bbdad94ef55c0d52037725efbf52e70d32f98bfb13d52022100de92a0abfb463ede8a4f71286885818a0f93dbedb365c3212428295c295d0375
2025/05/20 11:25:03 Verification failed: ECDSA signature verification returned false for txn 0xe264c79afefe5bfa067ac219a97e00c5385ba41a960a711aeb284cabf3133e34
--- PASS: TestMCPTransactionVerification (0.00s)
=== RUN   TestMCPStructSerialization
--- PASS: TestMCPStructSerialization (0.00s)
=== RUN   TestNewContextRecord
--- PASS: TestNewContextRecord (0.00s)
=== RUN   TestNetworkTransactionFlow_MultiNode
    network_transaction_test.go:155: Setting up test environment...
    network_transaction_test.go:177: Starting Node 1 on port 5000...
    network_transaction_test.go:180: Failed to start Node 1: failed to start node: chdir /home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP: no such file or directory
--- FAIL: TestNetworkTransactionFlow_MultiNode (0.00s)
=== RUN   TestPeerLifecycle_Integration
    dev_lifecycle_test.go:241: Setting up dev lifecycle integration test...
    dev_lifecycle_test.go:271: Starting Root Node (Network Mode) on HTTP:5050, P2P:4050
    dev_lifecycle_test.go:277: Started node process (mode: network, pid: 1398891) with args: [run . -network --database_path /tmp/agent-root-test-2931579085/root.db --no-wallet-server --port 5050 --p2p.port 4050]
    dev_lifecycle_test.go:277:   (Note: Logged ports 5050/4050 might be ignored by process depending on mode)
    dev_lifecycle_test.go:280: Waiting for node at http://localhost:5050 to become healthy...
2025/05/20 11:25:10 [INFO] Block.Hash: Block #0 being converted to proto. Timestamp: 1678886400, Nonce: 0, NumTx: 0
2025/05/20 11:25:10 [INFO] ProtoBlock fields for block #0:
2025/05/20 11:25:10 [INFO] - BlockNumber: 0
2025/05/20 11:25:10 [INFO] - PrevHash: 
2025/05/20 11:25:10 [INFO] - Nonce: 0
2025/05/20 11:25:10 [INFO] - Timestamp: 2023-03-15 13:20:00 +0000 UTC
2025/05/20 11:25:10 [INFO] - Transactions count: 0
2025/05/20 11:25:10 [INFO] - ProposerAddress: KNIRVCHAINb53c1e30b8a578c091dd40612bfd1433991b4e09
2025/05/20 11:25:10 [INFO] Initialized deterministic Genesis Block. Hash: c3a7b1ecbfa373db8a37da060e1f2f8927ed5d09d2e4b1bc11d53797d0bd4d3a
2025/05/20 11:25:10 KNIRVCHAIN Node starting. Version: dev, OS: linux, Arch: amd64. Log file: KNIRVCHAIN.log
2025/05/20 11:25:10 Determined node role: Client
2025/05/20 11:25:10 No -config flag provided, searching default locations...
2025/05/20 11:25:10 Using role-specific data directory: /home/gperry/.config/KNIRVCHAIN/client_data for role Client
2025/05/20 11:25:10 Config path for role Client: /home/gperry/.config/KNIRVCHAIN/client_data/config.json
2025/05/20 11:25:10 Checking config path: /home/gperry/.config/KNIRVCHAIN/client_data/config.json for role Client
2025/05/20 11:25:10 Loading config from: /home/gperry/.config/KNIRVCHAIN/client_data/config.json
2025/05/20 11:25:10 Successfully loaded config from: /home/gperry/.config/KNIRVCHAIN/client_data/config.json
2025/05/20 11:25:10 Using role-specific data directory: /home/gperry/.config/KNIRVCHAIN/client_data for role Client
2025/05/20 11:25:10 Wallet path for role Client: /home/gperry/.config/KNIRVCHAIN/client_data/wallet.dat
2025/05/20 11:25:10 Successfully validated wallet for address 'KNIRVCHAIN00d0cebc06980121871fba13d7d8774c71b43b4a'
2025/05/20 11:25:10 Installation is complete. Continuing with node initialization...
2025/05/20 11:25:10 Configuring Multi-Node Network Mode...
2025/05/20 11:25:10 Using role-specific data directory: /home/gperry/.config/KNIRVCHAIN/client_data for role Client
2025/05/20 11:25:10 GUI is disabled for Network Mode.
2025/05/20 11:25:10 Starting in multi-node network mode...
2025/05/20 11:25:10 Starting Main Node (HTTP: 5050, P2P: 4050, GUI: false, DB: /tmp/agent-root-test-2931579085/root.db)
2025/05/20 11:25:10 FATAL: actualReflectionNodeConfig is nil in network mode with GUI.
exit status 1
    dev_lifecycle_test.go:280: Timeout waiting for node at http://localhost:5050 to become healthy
    dev_lifecycle_test.go:90: Cleaning up node process group (mode: network, pid: 1398891)...
    dev_lifecycle_test.go:98: Sent SIGKILL to process group -1398891.
    dev_lifecycle_test.go:104: Node process group cleanup attempt complete for pid 1398891.
--- FAIL: TestPeerLifecycle_Integration (30.00s)
=== RUN   TestRoleBasedConfigPaths
=== RUN   TestRoleBasedConfigPaths/Root
2025/05/20 11:25:33 Using role-specific data directory: /home/gperry/.config/KNIRVCHAIN/root_data for role Root
2025/05/20 11:25:33 Config path for role Root: /home/gperry/.config/KNIRVCHAIN/root_data/config.json
2025/05/20 11:25:33 Using role-specific data directory: /home/gperry/.config/KNIRVCHAIN/root_data for role Root
2025/05/20 11:25:33 Config path for role Root: /home/gperry/.config/KNIRVCHAIN/root_data/config.json
2025/05/20 11:25:33 Checking config path: /home/gperry/.config/KNIRVCHAIN/root_data/config.json for role Root
2025/05/20 11:25:33 Using role-specific data directory: /home/gperry/.config/KNIRVCHAIN/root_data for role Root
2025/05/20 11:25:33 DEBUG: GetBlockchainDatabasePath - Constructed finalPath: '/home/gperry/.config/KNIRVCHAIN/root_data/agent_root.db' for role Root
2025/05/20 11:25:33 Loading config from: /home/gperry/.config/KNIRVCHAIN/root_data/config.json
=== RUN   TestRoleBasedConfigPaths/Bootnode
2025/05/20 11:25:33 Using role-specific data directory: /home/gperry/.config/KNIRVCHAIN/bootnode_data for role Bootnode
2025/05/20 11:25:33 Config path for role Bootnode: /home/gperry/.config/KNIRVCHAIN/bootnode_data/config.json
2025/05/20 11:25:33 Using role-specific data directory: /home/gperry/.config/KNIRVCHAIN/bootnode_data for role Bootnode
2025/05/20 11:25:33 Config path for role Bootnode: /home/gperry/.config/KNIRVCHAIN/bootnode_data/config.json
2025/05/20 11:25:33 Checking config path: /home/gperry/.config/KNIRVCHAIN/bootnode_data/config.json for role Bootnode
2025/05/20 11:25:33 Loading config from: /home/gperry/.config/KNIRVCHAIN/bootnode_data/config.json
=== RUN   TestRoleBasedConfigPaths/Peer
2025/05/20 11:25:33 Using role-specific data directory: /home/gperry/.config/KNIRVCHAIN/dev_data for role Peer
2025/05/20 11:25:33 Config path for role Peer: /home/gperry/.config/KNIRVCHAIN/dev_data/config.json
2025/05/20 11:25:33 Using role-specific data directory: /home/gperry/.config/KNIRVCHAIN/dev_data for role Peer
2025/05/20 11:25:33 Config path for role Peer: /home/gperry/.config/KNIRVCHAIN/dev_data/config.json
2025/05/20 11:25:33 Checking config path: /home/gperry/.config/KNIRVCHAIN/dev_data/config.json for role Peer
2025/05/20 11:25:33 Loading config from: /home/gperry/.config/KNIRVCHAIN/dev_data/config.json
=== RUN   TestRoleBasedConfigPaths/Client
2025/05/20 11:25:33 Using role-specific data directory: /home/gperry/.config/KNIRVCHAIN/client_data for role Client
2025/05/20 11:25:33 Config path for role Client: /home/gperry/.config/KNIRVCHAIN/client_data/config.json
2025/05/20 11:25:33 Using role-specific data directory: /home/gperry/.config/KNIRVCHAIN/client_data for role Client
2025/05/20 11:25:33 Config path for role Client: /home/gperry/.config/KNIRVCHAIN/client_data/config.json
2025/05/20 11:25:33 Checking config path: /home/gperry/.config/KNIRVCHAIN/client_data/config.json for role Client
2025/05/20 11:25:33 Loading config from: /home/gperry/.config/KNIRVCHAIN/client_data/config.json
--- PASS: TestRoleBasedConfigPaths (0.05s)
    --- PASS: TestRoleBasedConfigPaths/Root (0.00s)
    --- PASS: TestRoleBasedConfigPaths/Bootnode (0.05s)
    --- PASS: TestRoleBasedConfigPaths/Peer (0.00s)
    --- PASS: TestRoleBasedConfigPaths/Client (0.00s)
=== RUN   TestWalletConsistencyChecks
    role_integration_test.go:137: SaveWallet failed: failed to encrypt wallet data: crypto/aes: invalid key size 8
--- FAIL: TestWalletConsistencyChecks (0.00s)
=== RUN   TestGUIRoleBasedFeatures
=== RUN   TestGUIRoleBasedFeatures/Root_WithWallet_PaymentEnabled
=== RUN   TestGUIRoleBasedFeatures/Root_WithWallet_PaymentDisabled
    role_integration_test.go:318: Expected showRootSettings to be true, got false
=== RUN   TestGUIRoleBasedFeatures/Root_NoWallet_PaymentEnabled
=== RUN   TestGUIRoleBasedFeatures/Bootnode_WithWallet_PaymentEnabled
=== RUN   TestGUIRoleBasedFeatures/Bootnode_WithWallet_PaymentDisabled
=== RUN   TestGUIRoleBasedFeatures/Bootnode_NoWallet_PaymentEnabled
=== RUN   TestGUIRoleBasedFeatures/Peer_WithWallet
=== RUN   TestGUIRoleBasedFeatures/Peer_NoWallet
=== RUN   TestGUIRoleBasedFeatures/Client_WithWallet
=== RUN   TestGUIRoleBasedFeatures/Client_NoWallet
--- FAIL: TestGUIRoleBasedFeatures (0.00s)
    --- PASS: TestGUIRoleBasedFeatures/Root_WithWallet_PaymentEnabled (0.00s)
    --- FAIL: TestGUIRoleBasedFeatures/Root_WithWallet_PaymentDisabled (0.00s)
    --- PASS: TestGUIRoleBasedFeatures/Root_NoWallet_PaymentEnabled (0.00s)
    --- PASS: TestGUIRoleBasedFeatures/Bootnode_WithWallet_PaymentEnabled (0.00s)
    --- PASS: TestGUIRoleBasedFeatures/Bootnode_WithWallet_PaymentDisabled (0.00s)
    --- PASS: TestGUIRoleBasedFeatures/Bootnode_NoWallet_PaymentEnabled (0.00s)
    --- PASS: TestGUIRoleBasedFeatures/Peer_WithWallet (0.00s)
    --- PASS: TestGUIRoleBasedFeatures/Peer_NoWallet (0.00s)
    --- PASS: TestGUIRoleBasedFeatures/Client_WithWallet (0.00s)
    --- PASS: TestGUIRoleBasedFeatures/Client_NoWallet (0.00s)
=== RUN   TestTransactionFlow
2025/05/20 11:25:33 Successfully opened LevelDB at path: test_db/transaction_test_1747754733069573204.db
2025/05/20 11:25:33 [INFO] No existing blockchain data found for key 'test_chain_1747754733606297617'. Creating new blockchain.
2025/05/20 11:25:33 [INFO] Creating new blockchain for test_chain_1747754733606297617 using provided Genesis block.
2025/05/20 11:25:33 [INFO] Successfully created and stored new blockchain for key 'test_chain_1747754733606297617'.
    transaction_test.go:69: Added initial funding transaction 0x951b67bd5f245bbebe1aea4343ceef63c98af4ec43ad4f200f780527a96b2e4a for sender KNIRVCHAIN53b41f91c0cf1f1d08aead058529a76f5725314d
    transaction_test.go:81: Waiting for initial funding transaction to be mined...
2025/05/20 11:25:33 [INFO] Starting to Mine...
2025/05/20 11:25:33 [INFO] New transaction signal received, checking pool...
2025/05/20 11:25:33 [INFO] Found 1 verified transactions in pool. Preparing to mine.
2025/05/20 11:25:33 [INFO] Attempting to mine block with 1 verified transactions (sorted)...
2025/05/20 11:25:33 [INFO] Block.Hash: Block #0 being converted to proto. Timestamp: 1747754733, Nonce: 0, NumTx: 0
2025/05/20 11:25:33 [INFO] ProtoBlock fields for block #0:
2025/05/20 11:25:33 [INFO] - BlockNumber: 0
2025/05/20 11:25:33 [INFO] - PrevHash: 
2025/05/20 11:25:33 [INFO] - Nonce: 0
2025/05/20 11:25:33 [INFO] - Timestamp: 2025-05-20 15:25:33 +0000 UTC
2025/05/20 11:25:33 [INFO] - Transactions count: 0
2025/05/20 11:25:33 [INFO] - ProposerAddress: 
2025/05/20 11:25:33 [INFO] Block.Hash: Block #1 being converted to proto. Timestamp: 1747754733, Nonce: 0, NumTx: 2
2025/05/20 11:25:33 [INFO] ProtoBlock fields for block #1:
2025/05/20 11:25:33 [INFO] - BlockNumber: 1
2025/05/20 11:25:33 [INFO] - PrevHash: 1402b291502bc9e3e27a0cb3a056d478ef4bc6a1bc77e1c7d91f4cca2ea070b0
2025/05/20 11:25:33 [INFO] - Nonce: 0
2025/05/20 11:25:33 [INFO] - Timestamp: 2025-05-20 15:25:33 +0000 UTC
2025/05/20 11:25:33 [INFO] - Transactions count: 2
2025/05/20 11:25:33 [INFO] - ProposerAddress: 
2025/05/20 11:25:33 [INFO] Block.Hash: Block #1 being converted to proto. Timestamp: 1747754733, Nonce: 1, NumTx: 2
2025/05/20 11:25:33 [INFO] ProtoBlock fields for block #1:
2025/05/20 11:25:33 [INFO] - BlockNumber: 1
2025/05/20 11:25:33 [INFO] - PrevHash: 1402b291502bc9e3e27a0cb3a056d478ef4bc6a1bc77e1c7d91f4cca2ea070b0
2025/05/20 11:25:33 [INFO] - Nonce: 1
2025/05/20 11:25:33 [INFO] - Timestamp: 2025-05-20 15:25:33 +0000 UTC
2025/05/20 11:25:33 [INFO] - Transactions count: 2
2025/05/20 11:25:33 [INFO] - ProposerAddress: 
2025/05/20 11:25:33 [INFO] Block.Hash: Block #1 being converted to proto. Timestamp: 1747754733, Nonce: 2, NumTx: 2
2025/05/20 11:25:33 [INFO] ProtoBlock fields for block #1:
2025/05/20 11:25:33 [INFO] - BlockNumber: 1
2025/05/20 11:25:33 [INFO] - PrevHash: 1402b291502bc9e3e27a0cb3a056d478ef4bc6a1bc77e1c7d91f4cca2ea070b0
2025/05/20 11:25:33 [INFO] - Nonce: 2
2025/05/20 11:25:33 [INFO] - Timestamp: 2025-05-20 15:25:33 +0000 UTC
2025/05/20 11:25:33 [INFO] - Transactions count: 2
2025/05/20 11:25:33 [INFO] - ProposerAddress: 
2025/05/20 11:25:33 [INFO] Block.Hash: Block #1 being converted to proto. Timestamp: 1747754733, Nonce: 3, NumTx: 2
2025/05/20 11:25:33 [INFO] ProtoBlock fields for block #1:
2025/05/20 11:25:33 [INFO] - BlockNumber: 1
2025/05/20 11:25:33 [INFO] - PrevHash: 1402b291502bc9e3e27a0cb3a056d478ef4bc6a1bc77e1c7d91f4cca2ea070b0
2025/05/20 11:25:33 [INFO] - Nonce: 3
2025/05/20 11:25:33 [INFO] - Timestamp: 2025-05-20 15:25:33 +0000 UTC
2025/05/20 11:25:33 [INFO] - Transactions count: 2
2025/05/20 11:25:33 [INFO] - ProposerAddress: 
2025/05/20 11:25:33 [INFO] Block.Hash: Block #1 being converted to proto. Timestamp: 1747754733, Nonce: 4, NumTx: 2
2025/05/20 11:25:33 [INFO] ProtoBlock fields for block #1:
2025/05/20 11:25:33 [INFO] - BlockNumber: 1
2025/05/20 11:25:33 [INFO] - PrevHash: 1402b291502bc9e3e27a0cb3a056d478ef4bc6a1bc77e1c7d91f4cca2ea070b0
2025/05/20 11:25:33 [INFO] - Nonce: 4
2025/05/20 11:25:33 [INFO] - Timestamp: 2025-05-20 15:25:33 +0000 UTC
2025/05/20 11:25:33 [INFO] - Transactions count: 2
2025/05/20 11:25:33 [INFO] - ProposerAddress: 
2025/05/20 11:25:33 [INFO] Block.Hash: Block #1 being converted to proto. Timestamp: 1747754733, Nonce: 5, NumTx: 2
2025/05/20 11:25:33 [INFO] ProtoBlock fields for block #1:
2025/05/20 11:25:33 [INFO] - BlockNumber: 1
2025/05/20 11:25:33 [INFO] - PrevHash: 1402b291502bc9e3e27a0cb3a056d478ef4bc6a1bc77e1c7d91f4cca2ea070b0
2025/05/20 11:25:33 [INFO] - Nonce: 5
2025/05/20 11:25:33 [INFO] - Timestamp: 2025-05-20 15:25:33 +0000 UTC
2025/05/20 11:25:33 [INFO] - Transactions count: 2
2025/05/20 11:25:33 [INFO] - ProposerAddress: 
2025/05/20 11:25:33 [INFO] Block.Hash: Block #1 being converted to proto. Timestamp: 1747754733, Nonce: 6, NumTx: 2
2025/05/20 11:25:33 [INFO] ProtoBlock fields for block #1:
2025/05/20 11:25:33 [INFO] - BlockNumber: 1
2025/05/20 11:25:33 [INFO] - PrevHash: 1402b291502bc9e3e27a0cb3a056d478ef4bc6a1bc77e1c7d91f4cca2ea070b0
2025/05/20 11:25:33 [INFO] - Nonce: 6
2025/05/20 11:25:33 [INFO] - Timestamp: 2025-05-20 15:25:33 +0000 UTC
2025/05/20 11:25:33 [INFO] - Transactions count: 2
2025/05/20 11:25:33 [INFO] - ProposerAddress: 
2025/05/20 11:25:33 [INFO] Block.Hash: Block #1 being converted to proto. Timestamp: 1747754733, Nonce: 7, NumTx: 2
2025/05/20 11:25:33 [INFO] ProtoBlock fields for block #1:
2025/05/20 11:25:33 [INFO] - BlockNumber: 1
2025/05/20 11:25:33 [INFO] - PrevHash: 1402b291502bc9e3e27a0cb3a056d478ef4bc6a1bc77e1c7d91f4cca2ea070b0
2025/05/20 11:25:33 [INFO] - Nonce: 7
2025/05/20 11:25:33 [INFO] - Timestamp: 2025-05-20 15:25:33 +0000 UTC
2025/05/20 11:25:33 [INFO] - Transactions count: 2
2025/05/20 11:25:33 [INFO] - ProposerAddress: 
2025/05/20 11:25:33 [INFO] Block.Hash: Block #1 being converted to proto. Timestamp: 1747754733, Nonce: 8, NumTx: 2
2025/05/20 11:25:33 [INFO] ProtoBlock fields for block #1:
2025/05/20 11:25:33 [INFO] - BlockNumber: 1
2025/05/20 11:25:33 [INFO] - PrevHash: 1402b291502bc9e3e27a0cb3a056d478ef4bc6a1bc77e1c7d91f4cca2ea070b0
2025/05/20 11:25:33 [INFO] - Nonce: 8
2025/05/20 11:25:33 [INFO] - Timestamp: 2025-05-20 15:25:33 +0000 UTC
2025/05/20 11:25:33 [INFO] - Transactions count: 2
2025/05/20 11:25:33 [INFO] - ProposerAddress: 
2025/05/20 11:25:33 [INFO] Block.Hash: Block #1 being converted to proto. Timestamp: 1747754733, Nonce: 9, NumTx: 2
2025/05/20 11:25:33 [INFO] ProtoBlock fields for block #1:
2025/05/20 11:25:33 [INFO] - BlockNumber: 1
2025/05/20 11:25:33 [INFO] - PrevHash: 1402b291502bc9e3e27a0cb3a056d478ef4bc6a1bc77e1c7d91f4cca2ea070b0
2025/05/20 11:25:33 [INFO] - Nonce: 9
2025/05/20 11:25:33 [INFO] - Timestamp: 2025-05-20 15:25:33 +0000 UTC
2025/05/20 11:25:33 [INFO] - Transactions count: 2
2025/05/20 11:25:33 [INFO] - ProposerAddress: 
2025/05/20 11:25:33 [INFO] Block.Hash: Block #1 being converted to proto. Timestamp: 1747754733, Nonce: 10, NumTx: 2
2025/05/20 11:25:33 [INFO] ProtoBlock fields for block #1:
2025/05/20 11:25:33 [INFO] - BlockNumber: 1
2025/05/20 11:25:33 [INFO] - PrevHash: 1402b291502bc9e3e27a0cb3a056d478ef4bc6a1bc77e1c7d91f4cca2ea070b0
2025/05/20 11:25:33 [INFO] - Nonce: 10
2025/05/20 11:25:33 [INFO] - Timestamp: 2025-05-20 15:25:33 +0000 UTC
2025/05/20 11:25:33 [INFO] - Transactions count: 2
2025/05/20 11:25:33 [INFO] - ProposerAddress: 
2025/05/20 11:25:33 [INFO] Block.Hash: Block #1 being converted to proto. Timestamp: 1747754733, Nonce: 11, NumTx: 2
2025/05/20 11:25:33 [INFO] ProtoBlock fields for block #1:
2025/05/20 11:25:33 [INFO] - BlockNumber: 1
2025/05/20 11:25:33 [INFO] - PrevHash: 1402b291502bc9e3e27a0cb3a056d478ef4bc6a1bc77e1c7d91f4cca2ea070b0
2025/05/20 11:25:33 [INFO] - Nonce: 11
2025/05/20 11:25:33 [INFO] - Timestamp: 2025-05-20 15:25:33 +0000 UTC
2025/05/20 11:25:33 [INFO] - Transactions count: 2
2025/05/20 11:25:33 [INFO] - ProposerAddress: 
2025/05/20 11:25:33 [INFO] Block.Hash: Block #1 being converted to proto. Timestamp: 1747754733, Nonce: 12, NumTx: 2
2025/05/20 11:25:33 [INFO] ProtoBlock fields for block #1:
2025/05/20 11:25:33 [INFO] - BlockNumber: 1
2025/05/20 11:25:33 [INFO] - PrevHash: 1402b291502bc9e3e27a0cb3a056d478ef4bc6a1bc77e1c7d91f4cca2ea070b0
2025/05/20 11:25:33 [INFO] - Nonce: 12
2025/05/20 11:25:33 [INFO] - Timestamp: 2025-05-20 15:25:33 +0000 UTC
2025/05/20 11:25:33 [INFO] - Transactions count: 2
2025/05/20 11:25:33 [INFO] - ProposerAddress: 
2025/05/20 11:25:33 [INFO] Block.Hash: Block #1 being converted to proto. Timestamp: 1747754733, Nonce: 13, NumTx: 2
2025/05/20 11:25:33 [INFO] ProtoBlock fields for block #1:
2025/05/20 11:25:33 [INFO] - BlockNumber: 1
2025/05/20 11:25:33 [INFO] - PrevHash: 1402b291502bc9e3e27a0cb3a056d478ef4bc6a1bc77e1c7d91f4cca2ea070b0
2025/05/20 11:25:33 [INFO] - Nonce: 13
2025/05/20 11:25:33 [INFO] - Timestamp: 2025-05-20 15:25:33 +0000 UTC
2025/05/20 11:25:33 [INFO] - Transactions count: 2
2025/05/20 11:25:33 [INFO] - ProposerAddress: 
2025/05/20 11:25:33 [INFO] Block.Hash: Block #1 being converted to proto. Timestamp: 1747754733, Nonce: 14, NumTx: 2
2025/05/20 11:25:33 [INFO] ProtoBlock fields for block #1:
2025/05/20 11:25:33 [INFO] - BlockNumber: 1
2025/05/20 11:25:33 [INFO] - PrevHash: 1402b291502bc9e3e27a0cb3a056d478ef4bc6a1bc77e1c7d91f4cca2ea070b0
2025/05/20 11:25:33 [INFO] - Nonce: 14
2025/05/20 11:25:33 [INFO] - Timestamp: 2025-05-20 15:25:33 +0000 UTC
2025/05/20 11:25:33 [INFO] - Transactions count: 2
2025/05/20 11:25:33 [INFO] - ProposerAddress: 
2025/05/20 11:25:33 [INFO] Block.Hash: Block #1 being converted to proto. Timestamp: 1747754733, Nonce: 15, NumTx: 2
2025/05/20 11:25:33 [INFO] ProtoBlock fields for block #1:
2025/05/20 11:25:33 [INFO] - BlockNumber: 1
2025/05/20 11:25:33 [INFO] - PrevHash: 1402b291502bc9e3e27a0cb3a056d478ef4bc6a1bc77e1c7d91f4cca2ea070b0
2025/05/20 11:25:33 [INFO] - Nonce: 15
2025/05/20 11:25:33 [INFO] - Timestamp: 2025-05-20 15:25:33 +0000 UTC
2025/05/20 11:25:33 [INFO] - Transactions count: 2
2025/05/20 11:25:33 [INFO] - ProposerAddress: 
2025/05/20 11:25:33 [INFO] Block.Hash: Block #1 being converted to proto. Timestamp: 1747754733, Nonce: 16, NumTx: 2
2025/05/20 11:25:33 [INFO] ProtoBlock fields for block #1:
2025/05/20 11:25:33 [INFO] - BlockNumber: 1
2025/05/20 11:25:33 [INFO] - PrevHash: 1402b291502bc9e3e27a0cb3a056d478ef4bc6a1bc77e1c7d91f4cca2ea070b0
2025/05/20 11:25:33 [INFO] - Nonce: 16
2025/05/20 11:25:33 [INFO] - Timestamp: 2025-05-20 15:25:33 +0000 UTC
2025/05/20 11:25:33 [INFO] - Transactions count: 2
2025/05/20 11:25:33 [INFO] - ProposerAddress: 
2025/05/20 11:25:33 [INFO] Block.Hash: Block #1 being converted to proto. Timestamp: 1747754733, Nonce: 17, NumTx: 2
2025/05/20 11:25:33 [INFO] ProtoBlock fields for block #1:
2025/05/20 11:25:33 [INFO] - BlockNumber: 1
2025/05/20 11:25:33 [INFO] - PrevHash: 1402b291502bc9e3e27a0cb3a056d478ef4bc6a1bc77e1c7d91f4cca2ea070b0
2025/05/20 11:25:33 [INFO] - Nonce: 17
2025/05/20 11:25:33 [INFO] - Timestamp: 2025-05-20 15:25:33 +0000 UTC
2025/05/20 11:25:33 [INFO] - Transactions count: 2
2025/05/20 11:25:33 [INFO] - ProposerAddress: 
2025/05/20 11:25:33 [INFO] Block.Hash: Block #1 being converted to proto. Timestamp: 1747754733, Nonce: 18, NumTx: 2
2025/05/20 11:25:33 [INFO] ProtoBlock fields for block #1:
2025/05/20 11:25:33 [INFO] - BlockNumber: 1
2025/05/20 11:25:33 [INFO] - PrevHash: 1402b291502bc9e3e27a0cb3a056d478ef4bc6a1bc77e1c7d91f4cca2ea070b0
2025/05/20 11:25:33 [INFO] - Nonce: 18
2025/05/20 11:25:33 [INFO] - Timestamp: 2025-05-20 15:25:33 +0000 UTC
2025/05/20 11:25:33 [INFO] - Transactions count: 2
2025/05/20 11:25:33 [INFO] - ProposerAddress: 
2025/05/20 11:25:33 [INFO] Block.Hash: Block #1 being converted to proto. Timestamp: 1747754733, Nonce: 19, NumTx: 2
2025/05/20 11:25:33 [INFO] ProtoBlock fields for block #1:
2025/05/20 11:25:33 [INFO] - BlockNumber: 1
2025/05/20 11:25:33 [INFO] - PrevHash: 1402b291502bc9e3e27a0cb3a056d478ef4bc6a1bc77e1c7d91f4cca2ea070b0
2025/05/20 11:25:33 [INFO] - Nonce: 19
2025/05/20 11:25:33 [INFO] - Timestamp: 2025-05-20 15:25:33 +0000 UTC
2025/05/20 11:25:33 [INFO] - Transactions count: 2
2025/05/20 11:25:33 [INFO] - ProposerAddress: 
2025/05/20 11:25:33 [INFO] Block.Hash: Block #1 being converted to proto. Timestamp: 1747754733, Nonce: 20, NumTx: 2
2025/05/20 11:25:33 [INFO] ProtoBlock fields for block #1:
2025/05/20 11:25:33 [INFO] - BlockNumber: 1
2025/05/20 11:25:33 [INFO] - PrevHash: 1402b291502bc9e3e27a0cb3a056d478ef4bc6a1bc77e1c7d91f4cca2ea070b0
2025/05/20 11:25:33 [INFO] - Nonce: 20
2025/05/20 11:25:33 [INFO] - Timestamp: 2025-05-20 15:25:33 +0000 UTC
2025/05/20 11:25:33 [INFO] - Transactions count: 2
2025/05/20 11:25:33 [INFO] - ProposerAddress: 
2025/05/20 11:25:33 [INFO] Block.Hash: Block #1 being converted to proto. Timestamp: 1747754733, Nonce: 21, NumTx: 2
2025/05/20 11:25:33 [INFO] ProtoBlock fields for block #1:
2025/05/20 11:25:33 [INFO] - BlockNumber: 1
2025/05/20 11:25:33 [INFO] - PrevHash: 1402b291502bc9e3e27a0cb3a056d478ef4bc6a1bc77e1c7d91f4cca2ea070b0
2025/05/20 11:25:33 [INFO] - Nonce: 21
2025/05/20 11:25:33 [INFO] - Timestamp: 2025-05-20 15:25:33 +0000 UTC
2025/05/20 11:25:33 [INFO] - Transactions count: 2
2025/05/20 11:25:33 [INFO] - ProposerAddress: 
2025/05/20 11:25:33 [INFO] Block.Hash: Block #1 being converted to proto. Timestamp: 1747754733, Nonce: 22, NumTx: 2
2025/05/20 11:25:33 [INFO] ProtoBlock fields for block #1:
2025/05/20 11:25:33 [INFO] - BlockNumber: 1
2025/05/20 11:25:33 [INFO] - PrevHash: 1402b291502bc9e3e27a0cb3a056d478ef4bc6a1bc77e1c7d91f4cca2ea070b0
2025/05/20 11:25:33 [INFO] - Nonce: 22
2025/05/20 11:25:33 [INFO] - Timestamp: 2025-05-20 15:25:33 +0000 UTC
2025/05/20 11:25:33 [INFO] - Transactions count: 2
2025/05/20 11:25:33 [INFO] - ProposerAddress: 
2025/05/20 11:25:33 [INFO] Block.Hash: Block #1 being converted to proto. Timestamp: 1747754733, Nonce: 23, NumTx: 2
2025/05/20 11:25:33 [INFO] ProtoBlock fields for block #1:
2025/05/20 11:25:33 [INFO] - BlockNumber: 1
2025/05/20 11:25:33 [INFO] - PrevHash: 1402b291502bc9e3e27a0cb3a056d478ef4bc6a1bc77e1c7d91f4cca2ea070b0
2025/05/20 11:25:33 [INFO] - Nonce: 23
2025/05/20 11:25:33 [INFO] - Timestamp: 2025-05-20 15:25:33 +0000 UTC
2025/05/20 11:25:33 [INFO] - Transactions count: 2
2025/05/20 11:25:33 [INFO] - ProposerAddress: 
2025/05/20 11:25:33 [INFO] Block.Hash: Block #1 being converted to proto. Timestamp: 1747754733, Nonce: 24, NumTx: 2
2025/05/20 11:25:33 [INFO] ProtoBlock fields for block #1:
2025/05/20 11:25:33 [INFO] - BlockNumber: 1
2025/05/20 11:25:33 [INFO] - PrevHash: 1402b291502bc9e3e27a0cb3a056d478ef4bc6a1bc77e1c7d91f4cca2ea070b0
2025/05/20 11:25:33 [INFO] - Nonce: 24
2025/05/20 11:25:33 [INFO] - Timestamp: 2025-05-20 15:25:33 +0000 UTC
2025/05/20 11:25:33 [INFO] - Transactions count: 2
2025/05/20 11:25:33 [INFO] - ProposerAddress: 
2025/05/20 11:25:33 [INFO] Block.Hash: Block #1 being converted to proto. Timestamp: 1747754733, Nonce: 24, NumTx: 2
2025/05/20 11:25:33 [INFO] ProtoBlock fields for block #1:
2025/05/20 11:25:33 [INFO] - BlockNumber: 1
2025/05/20 11:25:33 [INFO] - PrevHash: 1402b291502bc9e3e27a0cb3a056d478ef4bc6a1bc77e1c7d91f4cca2ea070b0
2025/05/20 11:25:33 [INFO] - Nonce: 24
2025/05/20 11:25:33 [INFO] - Timestamp: 2025-05-20 15:25:33 +0000 UTC
2025/05/20 11:25:33 [INFO] - Transactions count: 2
2025/05/20 11:25:33 [INFO] - ProposerAddress: 
2025/05/20 11:25:33 [INFO] Attempting to add block number 1 (Hash: 045f8f51bdedba4a6dac05f656ad713a5f7a29b472dc749cd6b25d04e79ced20)
2025/05/20 11:25:33 [INFO] AddBlock: Current chain length before add: 1
2025/05/20 11:25:33 [INFO] Block.Hash: Block #1 being converted to proto. Timestamp: 1747754733, Nonce: 24, NumTx: 2
2025/05/20 11:25:33 [INFO] ProtoBlock fields for block #1:
2025/05/20 11:25:33 [INFO] - BlockNumber: 1
2025/05/20 11:25:33 [INFO] - PrevHash: 1402b291502bc9e3e27a0cb3a056d478ef4bc6a1bc77e1c7d91f4cca2ea070b0
2025/05/20 11:25:33 [INFO] - Nonce: 24
2025/05/20 11:25:33 [INFO] - Timestamp: 2025-05-20 15:25:33 +0000 UTC
2025/05/20 11:25:33 [INFO] - Transactions count: 2
2025/05/20 11:25:33 [INFO] - ProposerAddress: 
2025/05/20 11:25:33 [INFO] Block.Hash: Block #0 being converted to proto. Timestamp: 1747754733, Nonce: 0, NumTx: 0
2025/05/20 11:25:33 [INFO] ProtoBlock fields for block #0:
2025/05/20 11:25:33 [INFO] - BlockNumber: 0
2025/05/20 11:25:33 [INFO] - PrevHash: 
2025/05/20 11:25:33 [INFO] - Nonce: 0
2025/05/20 11:25:33 [INFO] - Timestamp: 2025-05-20 15:25:33 +0000 UTC
2025/05/20 11:25:33 [INFO] - Transactions count: 0
2025/05/20 11:25:33 [INFO] - ProposerAddress: 
2025/05/20 11:25:33 [INFO] Checks passed for block 1. Proceeding to process transactions and update state.
2025/05/20 11:25:33 [INFO] Pre-loaded balances for 3 affected accounts for block 1.
2025/05/20 11:25:33 [INFO] Processing tx 1/2 (Hash: 0x951b67bd5f245bbebe1aea4343ceef63c98af4ec43ad4f200f780527a96b2e4a) in block 1
2025/05/20 11:25:33 [INFO] Processing tx 2/2 (Hash: 0xd389bec88be73cc73006aa4e02a27188475341dc9ec6cb1a833f0595a7321972) in block 1
2025/05/20 11:25:34 [INFO] Successfully updated account balances in DB for block 1.
2025/05/20 11:25:34 [INFO] Removed 1 transactions from pool found in block 1.
2025/05/20 11:25:34 [INFO] Block 1 appended to in-memory chain.
2025/05/20 11:25:34 [INFO] Successfully added block number 1 to the blockchain and DB. New total blocks: 2
2025/05/20 11:25:34 [INFO] Successfully Mined block #1 with 2 transactions (incl. reward). Hash: 045f8f51bdedba4a6dac05f656ad713a5f7a29b472dc749cd6b25d04e79ced20
2025/05/20 11:25:34 [INFO] Broadcasting block #1
2025/05/20 11:25:34 [INFO] ProofOfWorkMining: Successfully mined block #1. Hash: 045f8f51bdedba4a6dac05f656ad713a5f7a29b472dc749cd6b25d04e79ced20. Calling AddBlock.
    transaction_test.go:96: Funding transaction 0x951b67bd5f245bbebe1aea4343ceef63c98af4ec43ad4f200f780527a96b2e4a mined in block 1
    transaction_test.go:111: Creating and signing the main test transaction...
    transaction_test.go:128: Adding main test transaction 0x7a6c1a17882bc436f814513efe13b729cbe1fc4bcc168d987319d940ff9bd816 to pool...
2025/05/20 11:25:34 Verification hash input: 
2KNIRVCHAIN53b41f91c0cf1f1d08aead058529a76f5725314d2KNIRVCHAINc491a14673ca8464c39b64c762291658b89afac7�"test transaction�
2025/05/20 11:25:34 Verification hash bytes (hex): 0a326b6e697276636861696e3533623431663931633063663166316430386165616430353835323961373666353732353331346412326b6e697276636861696e6334393161313436373363613834363463333962363463373632323931363538623839616661633718e807221074657374207472616e73616374696f6e2a0608eebdb2c106
2025/05/20 11:25:34 Signature bytes (hex): 3046022100ec890b32ab01d69aeb4a2b6bb18675150155eb10ba39909a0185f16e2b4a20aa0221008afecdafae6a2ad9f6d88ef3acc8878fbfc34d179fc9fdd00c5313f269245f58
2025/05/20 11:25:34 Verification hash input: 
2KNIRVCHAIN53b41f91c0cf1f1d08aead058529a76f5725314d2KNIRVCHAINc491a14673ca8464c39b64c762291658b89afac7�"test transaction�
2025/05/20 11:25:34 Verification hash bytes (hex): 0a326b6e697276636861696e3533623431663931633063663166316430386165616430353835323961373666353732353331346412326b6e697276636861696e6334393161313436373363613834363463333962363463373632323931363538623839616661633718e807221074657374207472616e73616374696f6e2a0608eebdb2c106
2025/05/20 11:25:34 Signature bytes (hex): 3046022100ec890b32ab01d69aeb4a2b6bb18675150155eb10ba39909a0185f16e2b4a20aa0221008afecdafae6a2ad9f6d88ef3acc8878fbfc34d179fc9fdd00c5313f269245f58
2025/05/20 11:25:34 [INFO] calculateTotalCryptoLocked for KNIRVCHAIN53b41f91c0cf1f1d08aead058529a76f5725314d: Iterating 2 blocks...
2025/05/20 11:25:34 [INFO]   Block 0: Processing...
2025/05/20 11:25:34 [INFO]   Block 0: Contains 0 transactions.
2025/05/20 11:25:34 [INFO]   Block 1: Processing...
2025/05/20 11:25:34 [INFO]   Block 1: Contains 2 transactions.
2025/05/20 11:25:34 [INFO]     Txn 0 (Hash: 0x951b67): Status='SUCCESS', From='KNIRVCHAINb53c1e30b8a578c091dd40612bfd1433991b4e09', To='KNIRVCHAIN53b41f91c0cf1f1d08aead058529a76f5725314d', Value=1000000
2025/05/20 11:25:34 [INFO]       Txn Status is SUCCESS. Checking addresses...
2025/05/20 11:25:34 [INFO]       MATCHED 'To': Added 1000000. New sum: 1000000
2025/05/20 11:25:34 [INFO]     Txn 1 (Hash: 0xd389be): Status='SUCCESS', From='KNIRVCHAINb53c1e30b8a578c091dd40612bfd1433991b4e09', To='KNIRVCHAINc491a14673ca8464c39b64c762291658b89afac7', Value=120000
2025/05/20 11:25:34 [INFO]       Txn Status is SUCCESS. Checking addresses...
2025/05/20 11:25:34 [INFO]       Txn does not involve address KNIRVCHAIN53b41f91c0cf1f1d08aead058529a76f5725314d
2025/05/20 11:25:34 [INFO] calculateTotalCryptoLocked for KNIRVCHAIN53b41f91c0cf1f1d08aead058529a76f5725314d finished. Final calculated balance: 1000000
2025/05/20 11:25:34 [INFO] Simulated Balance Check for KNIRVCHAIN53b41f91c0cf1f1d08aead058529a76f5725314d -> KNIRVCHAINc491a14673ca8464c39b64c762291658b89afac7 (Value: 1000, Fee: 0): Initial on-chain balance for KNIRVCHAIN53b41f91c0cf1f1d08aead058529a76f5725314d = 1000000
2025/05/20 11:25:34 [INFO]   = Final Check: Final Simulated Balance (1000000) >= New Txn Total Cost (Value: 1000 + Fee: 0 = 1000) -> true
2025/05/20 11:25:34 [INFO] Added transaction 0x7a6c1a17882bc436f814513efe13b729cbe1fc4bcc168d987319d940ff9bd816 to pool
2025/05/20 11:25:34 [INFO] Mining stopped gracefully
    transaction_test.go:141: Transaction 0x7a6c1a17882bc436f814513efe13b729cbe1fc4bcc168d987319d940ff9bd816 not marked as verified in pool
--- FAIL: TestTransactionFlow (1.51s)
=== RUN   TestURIGeneratorHandler_Integration
    uri_generator_test.go:32: Failed to start test node: failed to start node: chdir /home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP: no such file or directory
--- FAIL: TestURIGeneratorHandler_Integration (0.00s)
=== RUN   TestParseResourceURI
=== RUN   TestParseResourceURI/Valid_Chain_URI
    uri_generator_test.go:255: Params: map[]
=== RUN   TestParseResourceURI/Valid_Chain_URI_with_Path
    uri_generator_test.go:255: Params: map[]
=== RUN   TestParseResourceURI/Valid_Chain_URI_with_Query
    uri_generator_test.go:255: Params: map[hash:xyz789]
=== RUN   TestParseResourceURI/Valid_NRN_URI
    uri_generator_test.go:255: Params: map[]
=== RUN   TestParseResourceURI/Invalid_Scheme
=== RUN   TestParseResourceURI/Invalid_Authority_Format
=== RUN   TestParseResourceURI/Invalid_Resource_Type
    uri_generator_test.go:239: Unexpected error: unsupported resource type: invalid
--- FAIL: TestParseResourceURI (0.00s)
    --- PASS: TestParseResourceURI/Valid_Chain_URI (0.00s)
    --- PASS: TestParseResourceURI/Valid_Chain_URI_with_Path (0.00s)
    --- PASS: TestParseResourceURI/Valid_Chain_URI_with_Query (0.00s)
    --- PASS: TestParseResourceURI/Valid_NRN_URI (0.00s)
    --- PASS: TestParseResourceURI/Invalid_Scheme (0.00s)
    --- PASS: TestParseResourceURI/Invalid_Authority_Format (0.00s)
    --- FAIL: TestParseResourceURI/Invalid_Resource_Type (0.00s)
=== RUN   TestGenerateResourceURI
=== RUN   TestGenerateResourceURI/Chain_URI
    uri_generator_test.go:311: Generated URI: agent://abc123.chain/
=== RUN   TestGenerateResourceURI/Chain_URI_with_Path
    uri_generator_test.go:311: Generated URI: agent://abc123.chain/block
=== RUN   TestGenerateResourceURI/Chain_URI_with_Path_and_Params
    uri_generator_test.go:311: Generated URI: agent://abc123.chain/block?hash=xyz789
=== RUN   TestGenerateResourceURI/NRN_URI
    uri_generator_test.go:311: Generated URI: agent://content123.nrn/
--- PASS: TestGenerateResourceURI (0.00s)
    --- PASS: TestGenerateResourceURI/Chain_URI (0.00s)
    --- PASS: TestGenerateResourceURI/Chain_URI_with_Path (0.00s)
    --- PASS: TestGenerateResourceURI/Chain_URI_with_Path_and_Params (0.00s)
    --- PASS: TestGenerateResourceURI/NRN_URI (0.00s)
=== RUN   TestParseURI
=== RUN   TestParseURI/Valid_URI_with_subpath
=== RUN   TestParseURI/Valid_URI_without_subpath
=== RUN   TestParseURI/Valid_URI_with_chainID
=== RUN   TestParseURI/Invalid_scheme
=== RUN   TestParseURI/Missing_path
=== RUN   TestParseURI/Missing_resource_type
--- PASS: TestParseURI (0.00s)
    --- PASS: TestParseURI/Valid_URI_with_subpath (0.00s)
    --- PASS: TestParseURI/Valid_URI_without_subpath (0.00s)
    --- PASS: TestParseURI/Valid_URI_with_chainID (0.00s)
    --- PASS: TestParseURI/Invalid_scheme (0.00s)
    --- PASS: TestParseURI/Missing_path (0.00s)
    --- PASS: TestParseURI/Missing_resource_type (0.00s)
=== RUN   TestWalletManager
=== RUN   TestWalletManager/SaveLoadWallet
2025/05/20 11:25:34 Wallet saved successfully to /tmp/KNIRVCHAIN-wallet-test2989748331/test_wallet.dat
=== RUN   TestWalletManager/SaveLoadMasterWallet
2025/05/20 11:25:34 Wallet saved successfully to /tmp/KNIRVCHAIN-wallet-test2989748331/test_master_wallet.dat
=== RUN   TestWalletManager/ErrorHandling
    wallet_manager_test.go:151: Expected os.ErrNotExist, got failed to load wallet from file: file does not exist
--- FAIL: TestWalletManager (0.00s)
    --- PASS: TestWalletManager/SaveLoadWallet (0.00s)
    --- PASS: TestWalletManager/SaveLoadMasterWallet (0.00s)
    --- FAIL: TestWalletManager/ErrorHandling (0.00s)
FAIL
FAIL    KNIRVCHAIN      42.166s
FAIL
gperry@cloud-eq:~/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO

---

<div class="footer-links">


© 2025 KNIRV Network
</div>
