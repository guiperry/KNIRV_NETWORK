

---

**Source**: KNIRVROOT/agent-bootnode-registry/BOOTNODE.md

# KNIRVROOT Bootnode Setup

This document explains how to set up and run a bootnode for the KNIRVROOT network.

## What is a Bootnode?

A bootnode is a special node in the KNIRVROOT network that helps other nodes discover each other. It acts as a rendezvous point for new nodes joining the network. Bootnodes register themselves with the KNIRVROOT Node Registry, which allows other nodes to discover them.

## Prerequisites

- Node.js (v14 or later) for running the registry service
- Go (v1.16 or later) for running the KNIRVROOT node
- A server with a public IP address
- Open ports:
  - 3003 (TCP) for the registry service
  - 3478 (UDP) for the STUN service
  - 5050 (TCP) for the P2P communication

## Setting Up the Registry Service

The registry service is a Node.js application that provides two main functions:
1. A registry for bootnodes to register themselves
2. A STUN service for nodes to discover their public IP address

To set up the registry service:

1. Install the required Node.js packages:
   ```
   npm install express stun dgram
   ```

2. Start the registry service:
   ```
   node registry-service.js
   ```

The registry service will start listening on port 3003 for HTTP requests and port 3478 for STUN requests.

## Running a Bootnode

To run a KNIRVROOT node as a bootnode:

1. Start the KNIRVROOT node with the `--bootnode` flag:
   ```
   ./KNIRVROOT --bootnode --p2p.port=5050
   ```

2. The node will automatically register itself with the agent.com central registry service. If this service is down, you can deploy your own.

3. Other nodes will be able to discover this bootnode through the registry.

## Verifying Bootnode Registration

To verify that your bootnode has registered successfully with the registry:

1. Check the logs of the registry service for registration messages.

2. Make an HTTP request to the registry service to list all registered nodes:
   ```
   curl http://registry.knirv.com:3003/nodes
   ```

## Troubleshooting

If your bootnode is not registering correctly:

1. Check that the registry service is running and accessible.

2. Verify that the required ports are open on your firewall.

3. Check the logs of the KNIRVROOT node for any error messages related to bootnode registration.

4. Try manually registering the bootnode with the registry:
   ```
   curl -X POST -H "Content-Type: application/json" -d '{"chainID":"your-chain-id","ip":"your-public-ip","port":5050}' http://registry.knirv.com:3003/register
   ```

## Security Considerations

In a production environment, you should consider:

1. Adding authentication to the registry service to prevent unauthorized registrations.

2. Using HTTPS for the registry service.

3. Implementing rate limiting to prevent abuse.

4. Regularly monitoring the registry for suspicious activity.

---

<div class="footer-links">


© 2025 KNIRV Network
</div>
