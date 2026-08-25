/**
 * Jest executes transformed CommonJS, whereas Vite exposes build variables on
 * `import.meta.env`. Replace only that Vite-specific expression in tests so
 * components keep using the native Vite API in production builds.
 */
function transformImportMetaEnv({ types: t }) {
  return {
    name: 'transform-import-meta-env-for-jest',
    visitor: {
      MemberExpression(path) {
        const { object, property, computed } = path.node;
        if (
          !computed &&
          t.isIdentifier(property, { name: 'env' }) &&
          t.isMetaProperty(object) &&
          t.isIdentifier(object.meta, { name: 'import' }) &&
          t.isIdentifier(object.property, { name: 'meta' })
        ) {
          path.replaceWith(
            t.memberExpression(t.identifier('process'), t.identifier('env'))
          );
        }
      },
    },
  };
}

module.exports = {
  presets: [
    ['@babel/preset-env', { targets: { node: 'current' } }],
    '@babel/preset-react',
    '@babel/preset-typescript',
  ],
  plugins: [transformImportMetaEnv],
};
