
module.exports = {
  presets: [
    '@babel/preset-env', // For compiling modern JavaScript down
    ['@babel/preset-react', { runtime: 'automatic' }], // For React JSX
    '@babel/preset-typescript', // For TypeScript
  ],
};
