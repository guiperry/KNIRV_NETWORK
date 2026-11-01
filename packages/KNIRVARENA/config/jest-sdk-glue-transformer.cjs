/**
 * Jest runs the arena as CommonJS, but @knirv/sdk is ES modules that read
 * `import.meta.url` as the default location of their .wasm files. The SDK passes
 * the module bytes itself (the bundled base64 copy), so those URLs are never
 * fetched; give them a harmless value and let babel-jest do the ESM -> CJS
 * conversion as it does for the rest of the arena.
 */
const babelJest = require('babel-jest').default;
const path = require('path');
// Same Babel config as the rest of the arena's JavaScript (it lives in config/).
const inner = babelJest.createTransformer({ configFile: path.join(__dirname, 'babel.config.cjs') });

module.exports = {
  process(source, filename, options) {
    return inner.process(source.replace(/import\.meta\.url/g, "require('url').pathToFileURL(__filename).href"), filename, options);
  },
  getCacheKey(source, filename, options) {
    return inner.getCacheKey(source, filename, options) + ':sdk-glue-1';
  },
};
