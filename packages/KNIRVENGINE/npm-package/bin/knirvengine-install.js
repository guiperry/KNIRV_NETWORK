#!/usr/bin/env node

require('../scripts/install').main().catch((error) => {
  console.error(`[knirvengine-install] ${error.message}`);
  process.exit(1);
});
