import { MockLedgerConnector } from './../test-utils/mock-ledgerconnector';
import { KnirvWallet } from './wallet';

const mnemonic =
  'source bonus chronic canvas draft south burst lottery vacant surface solve popular case indicate oppose farm nothing bullet exhibit title speed wink action roast';

describe('create wallet by mnemonic', () => {
  it('create success', async () => {
    const wallet = await KnirvWallet.createByMnemonic(mnemonic);
    const walletMnemonic = wallet.getMnemonic();

    expect(walletMnemonic).toBe(mnemonic);
  });

  it('account initialize success', async () => {
    const wallet = await KnirvWallet.createByMnemonic(mnemonic);

    expect(wallet.accounts.length).toBe(1);
  });

  it("initilaize account' address is corret", async () => {
    const wallet = await KnirvWallet.createByMnemonic(mnemonic);
    const account = wallet.accounts[0];
    const address = await account.getAddress('g');

    // This is the real ripemd160(sha256(pubkey)) address for this mnemonic's
    // HD-path-0 account (also used, independently, as the dummy `creator`
    // address in wallet-sign.spec.ts's Document fixture). The previous
    // expected value here ('g1qqqq...luuxe', a near-all-zero-byte address)
    // was baked against a placeholder key generator that always returned a
    // zero-filled private key regardless of the mnemonic - see
    // hd-wallet-keyring.ts's generateKeyPair.
    expect(address).toBe('g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5');
  });
});

describe('create wallet by web3 auth', () => {
  it('create success', async () => {
    const privateKeyHexStr = 'ea97b9fddb7e6bf6867090a7a819657047949fbb9466d617f940538efd888605';
    const wallet = await KnirvWallet.createByWeb3Auth(privateKeyHexStr);

    expect(wallet.currentKeyring.type).toBe('WEB3_AUTH');
  });

  it('account initialize success', async () => {
    const privateKeyHexStr = 'ea97b9fddb7e6bf6867090a7a819657047949fbb9466d617f940538efd888605';
    const wallet = await KnirvWallet.createByWeb3Auth(privateKeyHexStr);

    expect(wallet.accounts.length).toBe(1);
  });

  it("initilaize account' address is corret", async () => {
    const privateKeyHexStr = 'ea97b9fddb7e6bf6867090a7a819657047949fbb9466d617f940538efd888605';
    const wallet = await KnirvWallet.createByWeb3Auth(privateKeyHexStr);
    const account = wallet.accounts[0];
    const address = await account.getAddress('g');

    // privateKeyHexStr above is the same key the 'source bonus chronic...'
    // mnemonic derives at HD path 0 (44'/118'/0'/0/0), so this intentionally
    // matches the mnemonic test's expected address above - same account
    // reached two different ways. The previous expected value here was baked
    // against the old unhashed publicKeyToAddress placeholder, which (unlike
    // the mnemonic path) did have a real pubkey to work with, so it produced
    // a real-looking but still-wrong address rather than a zero-byte one.
    expect(address).toBe('g1jg8mtutu9khhfwc4nxmuhcpftf0pajdhfvsqf5');
  });
});

describe('create wallet by ledger', () => {
  it('create success', async () => {
    try {
      const ledgerConnector = await MockLedgerConnector.create();
      const wallet = await KnirvWallet.createByLedger(ledgerConnector);

      expect(wallet.currentKeyring.type).toBe('LEDGER');
    } catch (e) {}
  });

  it('account initialize success', async () => {
    try {
      const ledgerConnector = await MockLedgerConnector.create();
      const wallet = await KnirvWallet.createByLedger(ledgerConnector);

      expect(wallet.accounts.length).toBe(1);
    } catch (e) {}
  });
});

describe('serialize', () => {
  // SALT_KEY: TESTTESTTESTTEST
  // PASSWORD: PASSWORD
  it('serialize success', async () => {
    const wallet = await KnirvWallet.createByMnemonic(mnemonic);
    const serialized = await wallet.serialize('PASSWORD');

    expect(serialized).toBeTruthy();
  });

  it('deserialize success', async () => {
    const created = await KnirvWallet.createByMnemonic(mnemonic);
    const serialized = await created.serialize('PASSWORD');
    const wallet = await KnirvWallet.deserialize(serialized, 'PASSWORD');

    expect(wallet).toBeTruthy();
    expect(wallet.accounts.length).toBe(1);
  });

  it('deserialize invalid password', async () => {
    const serialized =
      'U2FsdGVkX1/1NOyr0ePUNNNzXcR0lN4p5nHveepGqc0048xBRarEAHuUOCMh41qXmMxMZMTnNq0xrV3zd5COb2aa2ETbzW57rHcwTdVU11kvbkcFuKb8Vqghv0fGauy+/BxetqL+8VzqTAL8u5DQg29N0HUtSGPc5cFGBPDJ/TZBiDh43/fYi0mR75cG397eVxOMKJcKbWf6jNke9E2xp9QOlLAf6iZeqRitLlT20P7qw2UycoOmmg6Lfj7Z1o5xyyHe2rvt2BYQkC8ny3C2j7stFNRXcB0B8EZyQRwECkvRSmSBmnwX7PjHMQoqVSi/HxG/oLZNFhUcwHxFWpKF24So0iv+nkiLU4LEwjOZrbgpv8CFI1KXAGf442iOLvO+j5e5sRyCJqt53+U219UF1dbCtT7pr7WyeCfsAR6Eb3qoejUVR92m+PAfi/JWaYm2V7fDKflpnRVTzo7Eqn3+aQdUHZqjFXLo6k2sTw8hdZULA017sAe4WffOHL8XfRmi1m4SJUfBw3QrHNvdGYHmLX7rVMPDDGKMfbhk5W/H+e/7UX3Cz8rbS5mYcJNwOXNCgX6YYAxljdGR7HzRPKFIzBgOEOr4R3h7wibiq9hHrPKm0mDJYRkscRVQFvy78ko91KcE2NDkzyWwwXA7WAiWtlg3m0FlZxm/4DzmOVz5t7Kw2ln36W+xnT3qCbq37j/xii4PQDPCPuurMCa122BYOHYeuC/6Jd0l1GBuo5Vk2GQ4vNUq9YG55kCeI5cVznwLjEMBYDtlBb54uwxJgpBGHYYK1CQW+aVEl8AJihMrNuEesjznTSTNj9ZWXuVwnXIbRo9UjaEydlcfXpsew0WMFC8J2Brl08rrnX6N9wPWZ1V3ZO+ccGN5X3MsFIOADuNzYtyZdXrpeJxUChsiRKXIihu3qyjJp4OkaIqYSi/WoRpSL6CzmvhzPStRwU8yxIEZE8dnoUJroyQDrQRUhtYqB+PobWL7rjenwpAtCsUfu220N4XfuM/GGwkpg8jiepvToYWPzy9vJXBT9WZigsce6EPun2RieGRvq7FoCBExJnUPqWzFr3yew7ThwxxWcgtaX25Ej9ChFRl3YtrXuFg+d9vvzlA4OZcjoowT3BSVHfQJcJEL/f97fu24aIa2LKM6aNKcTsU/3YTYtxupLoK1v+xwLaT3R7rCMXyw+58u3grOweFEWtRjpmcIzvVDR/UZ';
    let wallet;
    try {
      wallet = await KnirvWallet.deserialize(serialized, 'INVALID_PASSWORD');
    } catch (e) {}

    expect(wallet).toBeFalsy();
  });
});
