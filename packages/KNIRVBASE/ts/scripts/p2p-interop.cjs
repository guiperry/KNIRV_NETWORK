// Drives a TS StandaloneConsensus node against a Go node (nrvconform p2p node).
// Run by src/conformance/p2p_interop.test.ts; prints one JSON result object.
//   node scripts/p2p-interop.cjs <nrvconform-binary> <tsPort> <goPort> <network> <secret> [goSecret]
// With a different goSecret the nodes must NOT share a gossip mesh (topic isolation).
const { spawn } = require('child_process');
const readline = require('readline');
const path = require('path');
const { StandaloneConsensus } = require(path.join(__dirname, '../dist/cjs/components/p2pconsensus'));

const [bin, tsPort, goPort, network, secret, goSecret] = process.argv.slice(2);
const isolated = goSecret !== undefined && goSecret !== secret;
const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
const result = { ok: false, steps: {} };

const tsEvents = [];
const handler = {
  onOperationReceived: (op) => { tsEvents.push({ event: 'op', op }); },
  onSyncRequestReceived: (req) => {
    tsEvents.push({ event: 'sync_request', req });
    return { network_id: req.network_id, collection: req.collection, more_available: false, operations: [{
      network_id: req.network_id, collection: req.collection, document_id: 'from-ts',
      data: Buffer.from('{"hello":"ts"}').toString('base64'), vector_clock: { ts: 1 }, timestamp: 2, peer_id: 'ts' }] };
  },
  onSyncResponseReceived: (resp) => { tsEvents.push({ event: 'sync_response', resp }); },
  onPeerDiscovered: () => undefined,
};

async function waitFor(arr, pred, ms = 30000) {
  const end = Date.now() + ms;
  while (Date.now() < end) { const h = arr.find(pred); if (h) return h; await sleep(100); }
  throw new Error('timeout waiting for event');
}

(async () => {
  const cfg = { enabled: true, network_id: network, socket_path: '', network_secret: secret, mode: 'standalone',
    gateway_url: '', gateway_timeout: '2s', listen_addr: '127.0.0.1', port: Number(tsPort), bootstrap_peers: [] };
  const ts = new StandaloneConsensus(cfg, handler);
  await ts.start();
  const tsAddr = `${ts.listenAddrs.find((a) => a.includes('127.0.0.1'))}`;
  result.tsPeerId = ts.peerId;

  // Go node bootstraps to the TS node.
  const go = spawn(bin, ['p2p', 'node', goPort, network, isolated ? goSecret : secret, tsAddr]);
  const goEvents = [];
  readline.createInterface({ input: go.stdout }).on('line', (l) => { try { goEvents.push(JSON.parse(l)); } catch {} });
  go.stderr.on('data', () => {});
  const goSend = (line) => go.stdin.write(line + '\n');

  try {
    await waitFor(goEvents, (e) => e.event === 'ready');
    // Wait until gossipsub has exchanged subscriptions in both directions.
    const end = Date.now() + (isolated ? 6000 : 30000);
    while (ts.topicPeers().length === 0 && Date.now() < end) await sleep(100);
    result.steps.mesh = ts.topicPeers().length > 0;
    if (isolated) {
      result.steps = { isolated: !result.steps.mesh };
      result.ok = result.steps.isolated;
      return;
    }
    await sleep(2000); // let the go side see our subscription too

    const op = (id, peer) => ({ network_id: network, collection: 'c', document_id: id,
      data: Buffer.from('{"type":"insert"}').toString('base64'), vector_clock: { [peer]: 1 }, timestamp: 1700000000000, peer_id: peer });

    // TS -> Go (gossipsub)
    await ts.broadcastOperation(op('ts-op', 'ts'));
    const atGo = await waitFor(goEvents, (e) => e.event === 'op' && e.op.document_id === 'ts-op');
    result.steps.tsToGo = atGo.op.data === op('x').data && atGo.op.peer_id === 'ts';

    // Go -> TS (gossipsub)
    goSend('PUB ' + JSON.stringify(op('go-op', 'go')));
    const atTs = await waitFor(tsEvents, (e) => e.event === 'op' && e.op.document_id === 'go-op');
    result.steps.goToTs = atTs.op.data === op('x').data && atTs.op.peer_id === 'go';

    // TS asks Go for a sync over the direct stream protocol; Go answers with one op.
    await ts.requestSync({ network_id: network, collection: 'c', vector_clock: { ts: 0 } });
    const resp = await waitFor(tsEvents, (e) => e.event === 'sync_response');
    result.steps.tsSyncsFromGo = resp.resp.operations.some((o) => o.document_id === 'from-go');

    // Go asks TS for a sync.
    goSend('SYNC ' + JSON.stringify({ network_id: network, collection: 'c', vector_clock: { go: 0 } }));
    await waitFor(tsEvents, (e) => e.event === 'sync_request');
    const goResp = await waitFor(goEvents, (e) => e.event === 'sync_response');
    result.steps.goSyncsFromTs = goResp.resp.operations.some((o) => o.document_id === 'from-ts');

    result.ok = Object.values(result.steps).every(Boolean);
  } catch (e) {
    result.error = String(e && e.message || e);
  } finally {
    go.stdin.end();
    await ts.stop();
    await sleep(300);
    console.log(JSON.stringify(result));
    process.exit(result.ok ? 0 : 1);
  }
})();
