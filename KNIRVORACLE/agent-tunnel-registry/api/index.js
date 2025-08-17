// agent-tunnel-registry/api/index.js
const express = require('express');
const registrationRoutes = require('./registrationRoutes');
const uriRoutes = require('./uriRoutes');

const router = express.Router();

router.use('/registry', registrationRoutes);
router.use('/uri', uriRoutes);

module.exports = router;