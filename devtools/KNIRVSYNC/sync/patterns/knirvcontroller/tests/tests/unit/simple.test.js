// Simple test to verify Jest setup
describe('Simple Jest Test', () => {
  test('should pass basic test', () => {
    expect(1 + 1).toBe(2);
  });

  test('should handle async operations', async () => {
    const result = await Promise.resolve('test');
    expect(result).toBe('test');
  });

  test('should work with objects', () => {
    const testObj = {
      name: 'KNIRVWALLET',
      version: '1.0.0'
    };
    
    expect(testObj).toHaveProperty('name');
    expect(testObj.name).toBe('KNIRVWALLET');
  });
});
