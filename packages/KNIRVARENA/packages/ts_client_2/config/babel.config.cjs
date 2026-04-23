module.exports = function(api) {
  api.cache(true);

  return {
    presets: [
      ['@babel/preset-env', {
        targets: {
          node: 'current'
        }
      }],
      ['@babel/preset-react', {
        runtime: 'automatic'
      }],
      '@babel/preset-typescript'
    ],
    plugins: [
      '@babel/plugin-proposal-class-properties',
      '@babel/plugin-transform-runtime'
    ],
    env: {
      test: {
        presets: [
          ['@babel/preset-env', {
            targets: {
              node: 'current'
            }
          }],
          ['@babel/preset-react', {
            runtime: 'automatic'
          }],
          '@babel/preset-typescript'
        ],
        plugins: [
          '@babel/plugin-proposal-class-properties',
          '@babel/plugin-transform-runtime'
        ]
      }
    }
  };
};
