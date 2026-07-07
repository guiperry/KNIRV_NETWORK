import { KnirvWallet } from './wallet';

describe('oracle message signing', () => {
  it('signs keccak256 oracle messages with the current account key', async () => {
    const privateKey = '0000000000000000000000000000000000000000000000000000000000000001';
    const wallet = await KnirvWallet.createByWeb3Auth(privateKey);
    const message =
      'transfer:0x7e5f4552091a69125d5dfcb7b8c2659029395bdf:0x0000000000000000000000000000000000000002:5:0';

    const signature = await wallet.signOracleMessage(message);

    expect(signature).toBe(
      '0x61f3726f9c26522c833cba2e13438c4fbcf89b23b5d2e95df78169dfd7172e7254cf50ac93dcf5ab3d070e5a703ae5ad9cb48fe9d422a8bc06e13427fefb1ad901',
    );
  });
});
