// Simple TypeScript test to verify ts-jest setup
describe('Simple TypeScript Test', () => {
  test('should handle TypeScript syntax', () => {
    const testValue: string = 'KNIRVWALLET';
    expect(testValue).toBe('KNIRVWALLET');
  });

  test('should work with interfaces', () => {
    interface TestWallet {
      name: string;
      version: number;
    }
    
    const wallet: TestWallet = {
      name: 'KNIRVWALLET',
      version: 1
    };
    
    expect(wallet.name).toBe('KNIRVWALLET');
    expect(wallet.version).toBe(1);
  });
});
