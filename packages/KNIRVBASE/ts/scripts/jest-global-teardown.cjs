const fs = require('fs');
module.exports = async () => {
  if (global.__NRVCONFORM_DIR__) fs.rmSync(global.__NRVCONFORM_DIR__, { recursive: true, force: true });
};
