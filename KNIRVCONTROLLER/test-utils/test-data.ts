// Test Data Constants and Fixtures for KNIRVWALLET Testing

export const TEST_MNEMONICS = {
  VALID_12_WORD: 'abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about',
  VALID_24_WORD: 'abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon art',
  INVALID: 'invalid mnemonic phrase that should fail validation',
  CUSTOM_TEST: 'source bonus chronic canvas draft south burst lottery vacant surface solve popular case indicate oppose farm nothing bullet exhibit title speed wink action roast'
};

export const TEST_PRIVATE_KEYS = {
  VALID_HEX: 'ea97b9fddb7e6bf6867090a7a819657047949fbb9466d617f940538efd888605',
  VALID_HEX_WITH_PREFIX: '0xea97b9fddb7e6bf6867090a7a819657047949fbb9466d617f940538efd888605',
  INVALID_SHORT: '1234567890abcdef',
  INVALID_LONG: 'ea97b9fddb7e6bf6867090a7a819657047949fbb9466d617f940538efd888605ea97b9fddb7e6bf6867090a7a819657047949fbb9466d617f940538efd888605'
};

export const TEST_ADDRESSES = {
  GNOLANG: 'g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5',
  ETHEREUM: '0x742d35Cc6634C0532925a3b8D4C9db96C4b4d8b6',
  XION: 'xion1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5',
  INVALID: 'invalid_address_format'
};

export const TEST_TRANSACTIONS = {
  SIMPLE_TRANSFER: {
    from: TEST_ADDRESSES.GNOLANG,
    to: 'g1234567890abcdef1234567890abcdef12345678',
    amount: '1000000',
    token: 'unrn',
    memo: 'Test transfer'
  },
  NRN_BURN_FOR_SKILL: {
    from: TEST_ADDRESSES.GNOLANG,
    to: 'g1skillcontractaddress1234567890abcdef123',
    amount: '500000',
    token: 'unrn',
    memo: 'Burn NRN for skill execution',
    metadata: {
      skillId: 'skill-123',
      parameters: { input: 'test data' }
    }
  },
  XION_META_ACCOUNT: {
    from: TEST_ADDRESSES.XION,
    to: 'xion1234567890abcdef1234567890abcdef12345678',
    amount: '2000000',
    token: 'uxion',
    memo: 'XION meta account transfer'
  }
};

export const TEST_NETWORKS = {
  KNIRV_TESTNET: {
    chainId: 'knirv-testnet-1',
    rpcEndpoint: 'http://localhost:1317',
    gasPrice: '0.025unrn',
    denom: 'unrn'
  },
  XION_TESTNET: {
    chainId: 'xion-testnet-1',
    rpcEndpoint: 'https://rpc.xion-testnet-1.burnt.com:443',
    gasPrice: '0.025uxion',
    denom: 'uxion'
  },
  ETHEREUM_TESTNET: {
    chainId: '5',
    rpcEndpoint: 'https://goerli.infura.io/v3/test',
    gasPrice: '20000000000',
    denom: 'ETH'
  }
};

export const TEST_WALLET_CONFIGS = {
  HD_WALLET: {
    name: 'Test HD Wallet',
    type: 'HD',
    mnemonic: TEST_MNEMONICS.VALID_12_WORD,
    derivationPath: "m/44'/118'/0'/0/0"
  },
  PRIVATE_KEY_WALLET: {
    name: 'Test Private Key Wallet',
    type: 'PRIVATE_KEY',
    privateKey: TEST_PRIVATE_KEYS.VALID_HEX
  },
  WEB3_AUTH_WALLET: {
    name: 'Test Web3Auth Wallet',
    type: 'WEB3_AUTH',
    privateKey: TEST_PRIVATE_KEYS.VALID_HEX
  },
  LEDGER_WALLET: {
    name: 'Test Ledger Wallet',
    type: 'LEDGER',
    derivationPath: "m/44'/118'/0'/0/0"
  },
  ADDRESS_ONLY_WALLET: {
    name: 'Test Address-Only Wallet',
    type: 'ADDRESS',
    address: TEST_ADDRESSES.GNOLANG
  }
};

export const TEST_XION_CONFIGS = {
  TESTNET: {
    chainId: 'xion-testnet-1',
    rpcEndpoint: 'https://rpc.xion-testnet-1.burnt.com:443',
    gasPrice: '0.025uxion',
    nrnTokenAddress: 'xion1nrn_contract_address_for_testing',
    faucetAddress: 'xion1faucet_contract_address_for_testing'
  },
  MAINNET: {
    chainId: 'xion-mainnet-1',
    rpcEndpoint: 'https://rpc.xion-mainnet-1.burnt.com:443',
    gasPrice: '0.025uxion',
    nrnTokenAddress: 'xion1nrn_contract_address_mainnet',
    faucetAddress: 'xion1faucet_contract_address_mainnet'
  }
};

export const TEST_SKILL_INVOCATIONS = {
  SIMPLE_SKILL: {
    skillId: 'skill-test-001',
    parameters: {
      input: 'Hello, world!',
      model: 'CodeT5',
      maxTokens: 100
    },
    nrnAmount: '1000000',
    expectedOutput: 'Processed: Hello, world!'
  },
  COMPLEX_SKILL: {
    skillId: 'skill-test-002',
    parameters: {
      input: 'Complex data processing task',
      model: 'CodeT5',
      maxTokens: 500,
      temperature: 0.7,
      context: 'Previous conversation context'
    },
    nrnAmount: '5000000',
    expectedOutput: 'Complex processed result'
  }
};

export const TEST_SYNC_SESSIONS = {
  MOBILE_BROWSER_SYNC: {
    sessionId: 'sync-session-test-001',
    mobileDeviceId: 'mobile-device-test-001',
    browserInstanceId: 'browser-instance-test-001',
    qrCodeData: 'knirv://sync?session=sync-session-test-001&key=test-encryption-key',
    encryptionKey: 'test-encryption-key-for-sync'
  }
};

export const TEST_API_RESPONSES = {
  WALLET_CREATION_SUCCESS: {
    success: true,
    data: {
      address: TEST_ADDRESSES.GNOLANG,
      mnemonic: TEST_MNEMONICS.VALID_12_WORD,
      type: 'HD'
    }
  },
  BALANCE_RESPONSE: {
    success: true,
    data: {
      balance: '1000000',
      denom: 'unrn',
      formatted: '1.000000 NRN'
    }
  },
  TRANSACTION_SUCCESS: {
    success: true,
    data: {
      txHash: '0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef',
      blockHeight: 12345,
      gasUsed: '21000'
    }
  },
  ERROR_RESPONSE: {
    success: false,
    error: 'Test error message',
    code: 'TEST_ERROR_CODE'
  }
};

export const TEST_ENCRYPTION_DATA = {
  PASSWORD: 'test-password-123',
  SALT: 'TESTTESTTESTTEST',
  ENCRYPTED_WALLET_DATA: 'encrypted_wallet_data_for_testing',
  KDF_CONFIG: {
    algorithm: 'argon2id',
    params: {
      outputLength: 32,
      opsLimit: 24,
      memLimitKib: 12 * 1024,
    },
  }
};

export const TEST_WEBSOCKET_MESSAGES = {
  WALLET_SYNC_REQUEST: {
    type: 'WALLET_SYNC_REQUEST',
    sessionId: TEST_SYNC_SESSIONS.MOBILE_BROWSER_SYNC.sessionId,
    walletData: {
      address: TEST_ADDRESSES.GNOLANG,
      balance: '1000000'
    }
  },
  TRANSACTION_BROADCAST: {
    type: 'TRANSACTION_BROADCAST',
    sessionId: TEST_SYNC_SESSIONS.MOBILE_BROWSER_SYNC.sessionId,
    transaction: TEST_TRANSACTIONS.SIMPLE_TRANSFER
  }
};

// Test environment configurations
export const TEST_ENV_CONFIGS = {
  UNIT_TEST: {
    NODE_ENV: 'test',
    MOCK_SERVICES: true,
    SKIP_NETWORK_CALLS: true
  },
  INTEGRATION_TEST: {
    NODE_ENV: 'test',
    MOCK_SERVICES: false,
    SKIP_NETWORK_CALLS: false,
    GATEWAY_URL: 'http://localhost:8000',
    WALLET_URL: 'http://localhost:8083'
  },
  E2E_TEST: {
    NODE_ENV: 'test',
    MOCK_SERVICES: false,
    SKIP_NETWORK_CALLS: false,
    GATEWAY_URL: 'http://localhost:8000',
    WALLET_URL: 'http://localhost:8083',
    XION_RPC: 'https://rpc.xion-testnet-1.burnt.com:443'
  }
};
