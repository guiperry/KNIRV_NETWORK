// Mock React Native components for Jest testing
const React = require('react');

// Mock basic React Native components
export const View = ({ children, style, ...props }) => 
  React.createElement('div', { style, ...props }, children);

export const Text = ({ children, style, ...props }) => 
  React.createElement('span', { style, ...props }, children);

export const ScrollView = ({ children, style, ...props }) => 
  React.createElement('div', { style: { overflow: 'auto', ...style }, ...props }, children);

export const TouchableOpacity = ({ children, onPress, style, ...props }) => 
  React.createElement('button', { 
    onClick: onPress, 
    style: { 
      background: 'transparent', 
      border: 'none', 
      cursor: 'pointer',
      ...style 
    }, 
    ...props 
  }, children);

export const TouchableHighlight = ({ children, onPress, style, ...props }) => 
  React.createElement('button', { 
    onClick: onPress, 
    style: { 
      background: 'transparent', 
      border: 'none', 
      cursor: 'pointer',
      ...style 
    }, 
    ...props 
  }, children);

export const TextInput = ({ onChangeText, value, style, placeholder, ...props }) => 
  React.createElement('input', { 
    onChange: (e) => onChangeText && onChangeText(e.target.value),
    value,
    placeholder,
    style,
    ...props 
  });

export const Image = ({ source, style, ...props }) => 
  React.createElement('img', { 
    src: typeof source === 'object' ? source.uri : source,
    style,
    ...props 
  });

export const FlatList = ({ data, renderItem, keyExtractor, style, ...props }) => 
  React.createElement('div', { style, ...props }, 
    data && data.map((item, index) => 
      renderItem({ item, index, separators: {} })
    )
  );

export const SectionList = ({ sections, renderItem, renderSectionHeader, keyExtractor, style, ...props }) => 
  React.createElement('div', { style, ...props }, 
    sections && sections.map((section, sectionIndex) => 
      React.createElement('div', { key: sectionIndex }, [
        renderSectionHeader && renderSectionHeader({ section }),
        section.data.map((item, index) => 
          renderItem({ item, index, section, separators: {} })
        )
      ])
    )
  );

// Mock StyleSheet
export const StyleSheet = {
  create: (styles) => styles,
  flatten: (style) => style,
  absoluteFill: {
    position: 'absolute',
    left: 0,
    right: 0,
    top: 0,
    bottom: 0
  }
};

// Mock Dimensions
export const Dimensions = {
  get: (dimension) => ({
    width: 375,
    height: 667,
    scale: 2,
    fontScale: 1
  }),
  addEventListener: jest.fn(),
  removeEventListener: jest.fn()
};

// Mock Platform
export const Platform = {
  OS: 'web',
  Version: '1.0',
  select: (obj) => obj.web || obj.default
};

// Mock Alert
export const Alert = {
  alert: jest.fn(),
  prompt: jest.fn()
};

// Mock Animated
export const Animated = {
  View: View,
  Text: Text,
  ScrollView: ScrollView,
  Image: Image,
  Value: jest.fn(() => ({
    setValue: jest.fn(),
    addListener: jest.fn(),
    removeListener: jest.fn(),
    interpolate: jest.fn(() => ({
      setValue: jest.fn(),
      addListener: jest.fn(),
      removeListener: jest.fn()
    }))
  })),
  timing: jest.fn(() => ({
    start: jest.fn()
  })),
  spring: jest.fn(() => ({
    start: jest.fn()
  })),
  decay: jest.fn(() => ({
    start: jest.fn()
  })),
  sequence: jest.fn(),
  parallel: jest.fn(),
  stagger: jest.fn(),
  loop: jest.fn()
};

// Mock AppState
export const AppState = {
  currentState: 'active',
  addEventListener: jest.fn(),
  removeEventListener: jest.fn()
};

// Mock AsyncStorage
export const AsyncStorage = {
  getItem: jest.fn(() => Promise.resolve(null)),
  setItem: jest.fn(() => Promise.resolve()),
  removeItem: jest.fn(() => Promise.resolve()),
  clear: jest.fn(() => Promise.resolve()),
  getAllKeys: jest.fn(() => Promise.resolve([])),
  multiGet: jest.fn(() => Promise.resolve([])),
  multiSet: jest.fn(() => Promise.resolve()),
  multiRemove: jest.fn(() => Promise.resolve())
};

// CommonJS exports
module.exports = {
  View,
  Text,
  ScrollView,
  TouchableOpacity,
  TouchableHighlight,
  TextInput,
  Image,
  FlatList,
  SectionList,
  StyleSheet,
  Dimensions,
  Platform,
  Alert,
  Animated,
  AppState,
  AsyncStorage
};

// Also export individual components for named imports
module.exports.View = View;
module.exports.Text = Text;
module.exports.ScrollView = ScrollView;
module.exports.TouchableOpacity = TouchableOpacity;
module.exports.TouchableHighlight = TouchableHighlight;
module.exports.TextInput = TextInput;
module.exports.Image = Image;
module.exports.FlatList = FlatList;
module.exports.SectionList = SectionList;
module.exports.StyleSheet = StyleSheet;
module.exports.Dimensions = Dimensions;
module.exports.Platform = Platform;
module.exports.Alert = Alert;
module.exports.Animated = Animated;
module.exports.AppState = AppState;
module.exports.AsyncStorage = AsyncStorage;
