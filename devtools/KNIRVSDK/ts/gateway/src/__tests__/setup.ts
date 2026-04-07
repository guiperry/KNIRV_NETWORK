// Mock fetch globally for all tests
global.fetch = jest.fn() as jest.MockedFunction<typeof fetch>;

beforeEach(() => {
  jest.clearAllMocks();
});
