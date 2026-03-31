// Mock for @cosmjs/proto-signing module
module.exports = {
  DirectSecp256k1HdWallet: class MockDirectSecp256k1HdWallet {
    static fromMnemonic = jest.fn().mockResolvedValue({
      getAccounts: jest.fn().mockResolvedValue([{
        address: 'cosmos1test',
        algo: 'secp256k1',
        pubkey: new Uint8Array(33)
      }])
    })
  },
  makeSignDoc: jest.fn().mockReturnValue({
    bodyBytes: new Uint8Array(),
    authInfoBytes: new Uint8Array(),
    chainId: 'test-chain',
    accountNumber: 1
  }),
  makeAuthInfoBytes: jest.fn().mockReturnValue(new Uint8Array()),
  makeSignBytes: jest.fn().mockReturnValue(new Uint8Array()),
  Registry: class MockRegistry {
    register = jest.fn()
    encode = jest.fn().mockReturnValue(new Uint8Array())
    decode = jest.fn().mockReturnValue({})
  },
  encodePubkey: jest.fn().mockReturnValue({
    type: 'tendermint/PubKeySecp256k1',
    value: 'test-pubkey'
  }),
  decodePubkey: jest.fn().mockReturnValue({
    type: 'secp256k1',
    value: new Uint8Array(33)
  })
};
