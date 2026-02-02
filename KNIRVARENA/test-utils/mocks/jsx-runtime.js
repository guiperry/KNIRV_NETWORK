// Mock JSX runtime for React Native testing
module.exports = {
  jsx: (type, props, key) => {
    if (typeof type === 'string') {
      return { type, props: props || {}, key };
    }
    return type(props || {});
  },
  jsxs: (type, props, key) => {
    if (typeof type === 'string') {
      return { type, props: props || {}, key };
    }
    return type(props || {});
  },
  Fragment: ({ children }) => children
};
