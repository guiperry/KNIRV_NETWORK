// agent-tunnel-registry/api/registrationRoutes.js
const express = require('express');
const registryManager = require('../registry/registryManager');

const router = express.Router();

router.post('/register', (req, res) => {
    const { devId, chainId, publicIp, publicP2pPort, type } = req.body;
    // This endpoint is for nodes that are *not* using the control channel for tunneling,
    // i.e., they are publicly reachable directly.
    if (!devId || !publicIp || !publicP2pPort) {
        return res.status(400).json({ error: 'Missing required fields for public registration: devId, publicIp, publicP2pPort' });
    }
    const nodeInfo = registryManager.registerNodeViaApi(devId, chainId, publicIp, publicP2pPort, type || 'dev');
    res.status(201).json({ message: 'Node registered successfully via API', nodeInfo });
});

router.get('/nodes', (req, res) => {
    res.json(registryManager.getAllNodes());
});

router.get('/node/dev/:devId', (req, res) => {
    const node = registryManager.getNodeByPeerId(req.params.devId);
    node ? res.json(node) : res.status(404).json({ error: 'Node not found by PeerID' });
});

router.get('/node/chain/:chainId', (req, res) => {
    const node = registryManager.getNodeByChainId(req.params.chainId);
    node ? res.json(node) : res.status(404).json({ error: 'Node not found by ChainID' });
});

module.exports = router;