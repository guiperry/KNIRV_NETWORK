// Mock for react-native module
module.exports = {
  Platform: {
    OS: 'web',
    select: (obj) => obj.web || obj.default
  },
  Dimensions: {
    get: () => ({ width: 1024, height: 768 })
  },
  Alert: {
    alert: jest.fn()
  },
  AsyncStorage: {
    getItem: jest.fn().mockResolvedValue(null),
    setItem: jest.fn().mockResolvedValue(undefined),
    removeItem: jest.fn().mockResolvedValue(undefined),
    clear: jest.fn().mockResolvedValue(undefined)
  },
  Linking: {
    openURL: jest.fn().mockResolvedValue(undefined),
    canOpenURL: jest.fn().mockResolvedValue(true)
  },
  DeviceEventEmitter: {
    addListener: jest.fn(),
    removeListener: jest.fn()
  },
  NativeModules: {},
  NativeEventEmitter: class MockNativeEventEmitter {
    addListener = jest.fn()
    removeListener = jest.fn()
    removeAllListeners = jest.fn()
  }
};
