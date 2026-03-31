# KNIRVARENA BACKEND SPECIFICATION (PHOENIX EDITION)

## 1. SYSTEM OVERVIEW & ARCHITECTURE
The **KNIRVARENA** serves as the decentralized orchestrator for the "Error Node" transformation process. It is built using the **Elixir/Phoenix** framework to leverage the BEAM’s native distributed capabilities. The Arena operates as a high-concurrency sidecar to the **KNIRVSERVER** (Go), facilitating the transition of AI failures into verified training assets.

### 1.1 The Hybrid Stack
* **Core Logic (KNIRVSERVER):** A Go-based application managing the KNIRVCHAIN, private Enterprise DVEs, and low-level network mesh.
* **Orchestration (KNIRVARENA):** An Elixir/Phoenix application managing the adversarial game state, player presence, and real-time resolution loops.
* **Frontend:** A TypeScript/Vite SPA served directly by the Phoenix `Plug.Static` layer, communicating via WebSockets (Phoenix Channels).

### 1.2 Frontend Implementation Details

The TypeScript/Vite SPA frontend is served directly by Phoenix's `Plug.Static` middleware, eliminating the need for a separate web server. This architecture provides:

#### 1.2.1 Phoenix Plug.Static Integration
```elixir
# lib/knirv_arena_web/endpoint.ex
defmodule KnirvArenaWeb.Endpoint do
  use Phoenix.Endpoint, otp_app: :knirv_arena

  # Serve the Vite-built SPA from priv/static
  plug Plug.Static,
    at: "/",
    from: :knirv_arena,
    gzip: true,
    cache_control_for_etags: "public, max-age=86400",
    cache_static_manifest: "priv/static/cache_manifest.json"

  # Fall back to index.html for SPA routes (Vue Router / React Router)
  plug Plug.Static,
    at: "/",
    from: :knirv_arena,
    only: ["index.html"]
end
```

#### 1.2.2 Phoenix Channels (WebSocket) Communication
Real-time bidirectional communication between the TypeScript frontend and Elixir backend uses Phoenix Channels over WebSockets:

```typescript
// src/services/phoenix-channel.service.ts
import { Socket } from "phoenix";

class PhoenixChannelService {
  private socket: Socket | null = null;
  private channels: Map<string, Channel> = new Map();

  connect(baseUrl: string, userToken: string): void {
    this.socket = new Socket(baseUrl, {
      params: { token: userToken },
      transport: WebSocket,
      heartbeatIntervalMs: 30000,
    });
    
    this.socket.connect();
  }

  joinArena(errorId: string): Channel {
    const channel = this.socket!.channel(`arena:resolution:${errorId}`, {
      error_id: errorId,
    });
    
    channel.join()
      .receive("ok", (resp) => console.log("Arena joined:", resp))
      .receive("error", (err) => console.error("Arena join failed:", err));
    
    this.channels.set(errorId, channel);
    return channel;
  }

  // Subscribe to real-time bounty updates across all clusters
  subscribeToLobby(): Channel {
    const channel = this.socket!.channel("arena:lobby");
    channel.join();
    
    channel.on("new_bounty", (payload) => {
      this.emitEvent("bounty:new", payload);
    });
    
    return channel;
  }
}
```

#### 1.2.3 Vite Configuration for Phoenix Integration
```typescript
// vite.config.ts
import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';
import phoenix from 'vite-plugin-phoenix';

export default defineConfig({
  plugins: [
    vue(),
    phoenix({ 
      // Point to the Phoenix endpoint URL for development
      endpoint: 'ws://localhost:4000/socket',
      // Enable hot module replacement
      hmrEndpoint: 'ws://localhost:4000/socket',
    }),
  ],
  build: {
    outDir: '../../priv/static',
    emptyOutDir: true,
  },
  server: {
    origin: 'http://localhost:4000',
  },
});
```

#### 1.2.4 SPA Routing with Phoenix Fallback
The frontend uses client-side routing (Vue Router or React Router). Phoenix is configured to serve the `index.html` for all non-API routes, allowing SPA navigation:

```elixir
# lib/knirv_arena_web/router.ex
defmodule KnirvArenaWeb.Router do
  use KnirvArenaWeb, :router

  # SPA fallback - serve index.html for non-API, non-static routes
  scope "/*path", KnirvArenaWeb do
    get("/", PageController, :index)
  end
end

# lib/knirv_arena_web/controllers/page_controller.ex
defmodule KnirvArenaWeb.PageController do
  def index(conn, _params):
    conn
    |> put_resp_content_type("text/html")
    |> send_file(0, Path.join(:code.priv_dir(:knirv_arena), "static/index.html"))
  end
end
```

#### 1.2.5 Benefits of This Architecture
1. **Single Deployment Unit:** The entire application (backend + frontend) deploys as one Erlang release
2. **Zero Configuration:** No need for separate nginx/apache configurations
3. **Native Hot Reloading:** Vite's HMR connects directly to Phoenix in dev mode
4. **WebSocket Integration:** Phoenix Channels provide built-in stateful WebSocket communication
5. **Production CDN-Ready:** The `priv/static` output can be offloaded to CDN in production

### 1.3 Build Pipeline: Vite to Phoenix Static

The frontend assets are automatically copied from the Vite build output to Phoenix's `priv/static` directory during the build process.

#### 1.3.1 Option A: package.json Scripts

```json
{
  "name": "knirv-arena",
  "scripts": {
    "dev": "vite",
    "build": "vite build",
    "postbuild": "node scripts/copy-to-priv.js",
    "deploy": "npm run build && cd .. && mix phx.digest"
  }
}
```

```javascript
// scripts/copy-to-priv.js
const fs = require('fs');
const path = require('path');

const srcDir = path.join(__dirname, '../dist');
const destDir = path.join(__dirname, '../../priv/static');

function copyDir(src, dest) {
  if (!fs.existsSync(dest)) {
    fs.mkdirSync(dest, { recursive: true });
  }
  
  const entries = fs.readdirSync(src, { withFileTypes: true });
  
  for (const entry of entries) {
    const srcPath = path.join(src, entry.name);
    const destPath = path.join(dest, entry.name);
    
    if (entry.isDirectory()) {
      copyDir(srcPath, destPath);
    } else {
      fs.copyFileSync(srcPath, destPath);
      console.log(`Copied: ${entry.name}`);
    }
  }
}

console.log('Copying Vite build to Phoenix priv/static...');
copyDir(srcDir, destDir);
console.log('Static assets copied successfully!');
```

#### 1.3.2 Option B: Makefile Integration

```makefile
# Makefile for KNIRVARENA

.PHONY: all build dev clean setup

# Configuration
PHOENIX_DIR := $(shell cd .. && pwd)
PRIV_STATIC := $(PHOENIX_DIR)/priv/static
DIST_DIR := dist

setup:
	@echo "Installing frontend dependencies..."
	cd frontend && npm install

dev: setup
	@echo "Starting Vite dev server with HMR..."
	cd frontend && npm run dev

build: build-frontend build-priv

build-frontend:
	@echo "Building TypeScript/Vite frontend..."
	cd frontend && npm run build

build-priv: build-frontend
	@echo "Copying build artifacts to priv/static..."
	@if [ -d "$(DIST_DIR)" ]; then \
		rsync -av --delete $(DIST_DIR)/ $(PRIV_STATIC)/; \
		echo "Copied to $(PRIV_STATIC)"; \
	else \
		echo "Error: dist directory not found. Run 'make build-frontend' first."; \
		exit 1; \
	fi

clean:
	@echo "Cleaning frontend build..."
	cd frontend && npm run clean
	@echo "Removing Phoenix static cache..."
	rm -rf $(PRIV_STATIC)/*
	cd .. && mix phx.digest.clean

deploy: build
	@echo "Generating Phoenix digest..."
	cd .. && mix phx.digest
	@echo "Building Phoenix release..."
	cd .. && mix release

# Convenience target for full rebuild
rebuild: clean build deploy
```

#### 1.3.3 Option C: Phoenix Tower Mix Task (Recommended)

For a more integrated approach, create a custom Mix task that wraps the frontend build:

```elixir
# lib/mix/tasks/build.frontend.ex
defmodule Mix.Tasks.Build.Frontend do
  use Mix.Task
  @shortdesc "Build frontend and copy to priv/static"

  def run(args) do
    frontend_dir = Path.join(File.cwd!(), "frontend")
    
    IO.puts("Building KNIRVARENA Frontend...")
    
    # Run Vite build
    Mix.Shell.cmd("cd #{frontend_dir} && npm run build", &IO.puts/1)
    
    # Copy to priv/static
    dist_dir = Path.join(frontend_dir, "dist")
    priv_static = Path.join(File.cwd!(), "priv/static")
    
    if File.exists?(dist_dir) do
      File.rm_rf!(priv_static)
      File.cp_r!(dist_dir, priv_static)
      IO.puts("Frontend assets copied to priv/static")
      
      # Generate digest for production
      Mix.Shell.cmd("mix phx.digest", &IO.puts/1)
    else
      IO.puts(:stderr, "Error: dist directory not found")
      exit(1)
    end
  end
end
```

Usage:
```bash
# Build everything
mix do build.frontend, phx.server

# Production build
mix do build.frontend, phx.digest, release
```

### 1.4 Frontend Feasibility Analysis: Static SPA vs Node.js Server

The current KNIRVARENA frontend codebase includes both a **browser-based SPA** and **Node.js server functionality**. This section analyzes the feasibility of migrating to a pure static SPA served by Phoenix `Plug.Static`, and presents alternative strategies.

#### 1.4.1 Current Architecture Analysis

The KNIRVARENA frontend currently contains the following Node.js server components:

| Component | File(s) | Functionality | Feasibility |
|-----------|---------|---------------|-------------|
| **API Server** | `src/server/api-server.ts` | Express-based REST API with API key auth, QR onboarding, rate limiting | **Must move to Phoenix** |
| **Unified Server** | `src/core/unifiedServer.ts` | Express server serving both frontend + backend | **Must move to Phoenix** |
| **WebSocket Service** | `src/services/WebSocketService.ts` | Real-time communication | **Can use Phoenix Channels** |
| **Database Service** | `src/core/services/databaseService.ts` | RxDB/IndexedDB local storage | **Browser-compatible** |
| **Static File Serving** | Various Express static middleware | Serving built frontend | **Replaced by Plug.Static** |

#### 1.4.2 Components That Work as Static SPA

The following features are fully compatible with a Vite-built static SPA served by Phoenix:

```typescript
// These work perfectly in browser-only SPA
- React/Vue components (UI layer)
- WebSocket client (Phoenix Channels client)
- Local storage/IndexedDB (RxDB works in browser)
- Canvas/WebGL game rendering (Three.js, Babylon)
- API calls to external services (KNIRVSERVER, KNIRVCHAIN)
- Crypto wallet integration (Abstraxion, XION Meta Accounts)
```

#### 1.4.3 Components Requiring Migration to Phoenix

**1. Express API Server (`src/server/api-server.ts`)**

Current features requiring Phoenix migration:
- `/public/qr/start` - QR onboarding nonce generation
- `/public/qr/complete` - QR completion with temp key issuance
- `/api/skills` - Skills API with permission checking
- API key authentication & rate limiting

**Migration to Phoenix:**
```elixir
# lib/knirv_arena_web/controllers/qr_controller.ex
defmodule KnirvArenaWeb.QRController do
  use KnirvArenaWeb, :controller

  def start(conn, _params) do
    nonce = :crypto.strong_rand_bytes(16) |> Base.encode16()
    # Store in ETS with TTL
    QRChallenge.store(nonce, %{ip: conn.remote_ip, inserted_at: DateTime.utc_now()})
    json(conn, %{success: true, nonce: nonce, expires_in_seconds: 300})
  end

  def complete(conn, %{nonce: nonce}) do
    # Validate nonce, IP match, TTL
    # Issue temporary API key
  end
end

# lib/knirv_arena_web/controllers/skills_controller.ex
defmodule KnirvArenaWeb.SkillsController do
  use KnirvArenaWeb, :controller
  plug :authenticate_api_key
  plug :require_permission, "read:skills"

  def index(conn, _params) do
    # Fetch skills from KNIRVCHAIN
  end
end
```

**2. WebSocket Server → Phoenix Channels**

Current: `src/services/WebSocketService.ts` using `ws` library
Solution: Phoenix Channels (already implemented in `backend/knirvarena/lib/knirv_arena_web/channels/arena_channel.ex`)

```typescript
// Replace ws with phoenix channels
import { Socket } from "phoenix";

const socket = new Socket("wss://knirvarena.com/socket", {
  params: { token: userToken }
});
socket.connect();
const channel = socket.channel("arena:lobby", {});
```

**3. API Key Service → Phoenix Plugs**

Current: `src/services/ApiKeyService.ts`
Solution: Phoenix Plugs for authentication

```elixir
# lib/knirv_arena_web/plugs/api_key_auth.ex
defmodule KnirvArenaWeb.Plugs.APIKeyAuth do
  import Plug.Conn

  def init(opts), do: opts

  def call(conn, _opts) do
    with api_key <- get_req_header(conn, "x-api-key") |> List.first(),
         {:ok, key} <- validate_key(api_key) do
      assign(conn, :api_key, key)
    else
      _ -> halt(conn) |> put_status(401) |> json(%{error: "Unauthorized"})
    end
  end
end
```

#### 1.4.4 Alternative Architecture Options

**Option A: Hybrid Phoenix + External API (Recommended)**

Keep Phoenix as the web layer, but maintain a separate Node.js microservice for complex API operations:

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   Phoenix   │    │  Node.js    │    │  KNIRV-    │
│  (Static    │◄──►│  (API       │◄──►│  SERVER     │
│   SPA +     │    │  Service)   │    │  (Go)       │
│   Channels) │    └─────────────┘    └─────────────┘
└─────────────┘
```

**Pros:** Fastest migration, separates concerns
**Cons:** Additional deployment complexity

**Option B: Full Phoenix Migration**

Migrate all Express endpoints to Phoenix controllers:

```
┌─────────────┐
│   Phoenix   │
│  (Static    │
│   SPA +     │
│   Channels  │
│   + API)    │
└─────────────┘
```

**Pros:** Single deployment unit, no separate Node.js service
**Cons:** Significant rewrite, Elixir learning curve for API logic

**Option C: Keep Node.js, Use Phoenix Only for Static + Channels**

Keep the existing Express API server, use Phoenix only for serving the SPA and WebSocket handling:

```
┌─────────────┐    ┌─────────────┐
│   Phoenix   │    │   Node.js   │
│  (Static    │    │  (Express   │
│   SPA +     │    │   API +     │
│   Channels) │    │   WebSocket)│
└─────────────┘    └─────────────┘
```

**Pros:** Minimal changes, proven architecture
**Cons:** Two services to deploy, more complex ops

#### 1.4.5 Recommendation

**Recommended: Option C (Hybrid)** with a path to **Option A**

1. **Immediate**: Serve frontend static assets via Phoenix `Plug.Static`, keep Express API server running
2. **Short-term**: Migrate WebSocket to Phoenix Channels (already implemented)
3. **Medium-term**: Move API endpoints to Phoenix controllers one-by-one
4. **Long-term**: Decommission Express server once Phoenix has feature parity

This approach minimizes risk while gradually migrating functionality to Phoenix.


---

## 2. THE "BOUNTY BRIDGE" (COMMUNICATIONS LAYER)
To maintain maximum performance and security on a single host, communication between the **KNIRVSERVER** and **KNIRVARENA** is handled via **Unix Domain Sockets (UDS)**.

### 2.1 FailureContext Ingestion
The Go server monitors private Enterprise CLEAN Nodes. When a failure is detected that requires external resolution, it packages the error into an **ErrorNodeBounty**.

* **Socket Path:** `/var/run/knirv/bridge.sock`
* **Protocol:** Protobuf or JSON-RPC over UDS.
* **Workflow:**
    1.  **KNIRVSERVER** detects failure in a private context.
    2.  Anonymization layer strips sensitive enterprise data, leaving only the logic-based `FailureContext`.
    3.  Bounty is pushed through the UDS to the **KNIRVARENA**'s `BountyListener` GenServer.

---

## 3. PRIVACY & THE ENTERPRISE BARRIER
A critical requirement is the isolation of private enterprise data from the public **KNIRVGRAPH**.

### 3.1 Data Tiering
* **Private Tier (KNIRVSERVER):** Contains full raw logs, PII, and enterprise-specific logic. Stays within the local node or encrypted enterprise cluster.
* **Public Tier (KNIRVGRAPH):** Contains the "Adversarial Snapshot"—a sanitized version of the error that retains only the structural/semantic failure required for training the transformer.
* **Interactive Tier (KNIRVARENA):** Allows users to investigate the "Adversarial Snapshot" and formulate appropriate solutions.

### 3.2 The Dual KNIRVGRAPH Architecture

The KNIRVGRAPH operates as a dual-layer system to maintain both privacy and collective intelligence:

#### 3.2.1 Private KNIRVGRAPH (Embedded)
Each **KNIRVSERVER** instance runs an embedded **private KNIRVGRAPH** that:
* Stores ErrorNodes containing sanitized failure contexts within the local enterprise boundary
* Maintains proprietary patterns and solution approaches unique to the organization
* Acts as a staging area before errors are promoted to the public graph
* Retains sensitive contextual data that should never leave the enterprise

#### 3.2.2 Public KNIRVGRAPH (Global Graphchain)
The global **public KNIRVGRAPH** serves as the shared knowledge backbone:
* Receives ErrorNodes from private graphs after sanitization
* Groups similar ErrorNodes into competitive error clusters
* Hosts SkillNode minting and LoRA adapter training
* Provides global skill discovery and invocation services

### 3.3 The Anonymization Gate
Before a `FailureContext` moves from the Go server to the Phoenix Arena, it must pass a validation gate that:
1.  Redacts specific keywords identified in the Enterprise DVE.
2.  Abstracts API endpoints or proprietary tool names into generic "Tool-X" identifiers.
3.  Signs the packet with the Enterprise's public key to guarantee the **NRN Fee** payment upon resolution.



---

## 4. DECENTRALIZED GENSERVER ORCHESTRATION
The Arena does not rely on a single central server. Instead, it utilizes a mesh of **Arena GenServers** across the active network.

### 4.1 Distributed State
* Each **ErrorNodeBounty** is assigned to a specific `ResolutionSession` GenServer.
* Using **Phoenix.PubSub**, the state of a specific resolution is broadcast across the node cluster.
* If a local **KNIRVARENA** node goes offline, the **KNIRVSERVER** (via the UDS heartbeat) detects the failure and re-routes the bounty to the next available Arena instance in the peer mesh.

## 5. ECONOMIC MODEL: NRN FEE MECHANICS
The KNIRVARENA operates on a "Proof of Resolution" economy. This ensures that the decentralized network of players and nodes is incentivized to solve high-value enterprise errors accurately and quickly.

### 5.1 The Bounty Lifecycle
* **Creation:** The Enterprise (via **KNIRVSERVER**) attaches an **NRN Bounty** to a `FailureContext`.
* **Escrow:** The NRN is locked in a smart contract on the **KNIRVCHAIN**.
* **Distribution:**
    * **70% to the Solver:** The player/agent who provides the validated "Winning Trajectory."
    * **20% to the Node Host:** The owner of the physical hardware (e.g., **Hasher** rigs) running the Arena instance.
    * **10% to the Protocol Treasury:** For continuous development of the KNIRV NETWORK.

### 5.2 Dynamic Pricing
Bounty fees are calculated based on **Semantic Complexity** and **Priority**. Errors that block critical Enterprise DVE workflows automatically scale their NRN offering to attract elite "Error Node" solvers.

---

## 6. THE ADVERSARIAL RESOLUTION WORKFLOW
This section details the journey from a raw failure to a verified training asset.

### 6.1 The Complete Resolution Pipeline

| Phase | Action | Responsible Component |
| :--- | :--- | :--- |
| **1. Trigger** | Private AI failure occurs within the DVE; PII is redacted locally. | **KNIRVSERVER (Go)** |
| **2. Private Graph Ingestion** | ErrorNode is created in the **private KNIRVGRAPH** (embedded in KNIRVSERVER) with sanitized failure context. | **Private KNIRVGRAPH** |
| **3. Public Graph Promotion** | ErrorNode is promoted to the **public KNIRVGRAPH** (global Graphchain) after anonymization passes the validation gate. | **KNIRVSERVER Bridge** |
| **4. Clustering** | ErrorNode is grouped into an error cluster with similar errors on the public graph. | **Public KNIRVGRAPH** |
| **5. Phoenix Broadcasting** | ErrorNode is announced to the **KNIRVARENA** for adversarial resolution. | **BountyListener (Elixir)** |
| **6. Adversarial Play** | Players compete to find the optimal fix path. | **KNIRVARENA (TS Frontend)** |
| **7. Validation** | The solution is tested against the original constraints. | **Validation GenServer** |
| **8. Settlement** | NRN is released; Trajectory is hashed to Chain. | **KNIRVCHAIN (Go)** |

---

## 7. KNOWLEDGE ASSET EXPORT & LoRAX INTEGRATION
The ultimate product of the Arena is not just a solved error, but a **Knowledge Asset**—a high-fidelity training pair.

### 7.1 The "Delta" Extraction
Instead of just saving the final answer, the Phoenix backend extracts the **Correction Path**. This includes:
* The specific logic branch that failed.
* The exact mutation required to fix it.
* The environmental variables that remained constant.

### 7.2 Continuous Fine-Tuning Pipeline
The **KNIRVARENA** periodically batches these assets into a `.jsonl` format optimized for the **LoRAX** backend. This allows the **KNIRV NETWORK**'s private error resolution transformer to undergo "Living Evolution," where it learns from the very mistakes it made hours earlier.

---

## 8. SECURITY & ANTI-SYBIL MEASURES
To prevent "Bounty Farming" (where bots submit low-quality or hallucinated solutions), the Arena employs several safeguards:

* **Reputation Scores:** Players must maintain a specific "Resolution Accuracy" ratio to access high-NRN bounties.
* **Deterministic Validation:** The **KNIRVSERVER** runs the proposed solution in a sandboxed environment to ensure it actually resolves the `FailureContext` before releasing funds.
* **Stake-to-Play:** High-tier Error Nodes may require players to stake a small amount of NRN, which is slashed if they submit intentionally malicious or fraudulent "solutions."

---

## 9. TECHNICAL IMPLEMENTATION: THE BOUNTYLISTENER (UDS BRIDGE)

The **BountyListener** is the entry point for all "ErrorNodeBounties" originating from the private **KNIRVSERVER** (Go). By using a Unix Domain Socket (UDS), we achieve near-zero latency while maintaining an air-gapped security model between the enterprise data and the public arena.

### 9.1 Elixir UDS Server Implementation
This GenServer initializes the socket and listens for incoming binary packets from the Go side.

```elixir
defmodule KnirvArena.BountyListener do
  use GenServer
  require Logger

  @socket_path "/var/run/knirv/bridge.sock"

  def start_link(_) do
    GenServer.start_link(__MODULE__, %{}, name: __MODULE__)
  end

  @impl true
  def init(state) do
    # Ensure clean socket state
    File.rm(@socket_path)
    
    case :gen_tcp.listen(0, [:local, {:ifaddr, {:local, @socket_path}}, {:active, true}, :binary]) do
      {:ok, listen_socket} ->
        Logger.info("KNIRVARENA: UDS Bridge active at #{@socket_path}")
        {:ok, %{listen_socket: listen_socket}}
      {:error, reason} ->
        Logger.error("Failed to start UDS Bridge: #{inspect(reason)}")
        {:stop, reason}
    end
  end

  @impl true
  def handle_info({:tcp, _port, data}, state) do
    # Decode the FailureContext from the Go binary stream
    with {:ok, context} <- Jason.decode(data),
         {:ok, _session_pid} <- KnirvArena.ResolutionSupervisor.start_session(context) do
      Logger.debug("Bounty Received: #{context["error_id"]} - Spawning ResolutionSession.")
    end
    {:noreply, state}
  end
end
```

### 9.2 The Go-Side Client (KNIRVSERVER)
In your Go implementation, use the `net` package to push the anonymized `FailureContext` to Phoenix.

```go
func PushToArena(bounty []byte) error {
    conn, err := net.Dial("unix", "/var/run/knirv/bridge.sock")
    if err != nil {
        return err
    }
    defer conn.Close()
    _, err = conn.Write(bounty)
    return err
}
```

---

## 10. THE RESOLUTION SESSION (GENSERVER STATE MACHINE)

Once a bounty is received, a dynamic `ResolutionSession` GenServer is spawned. This process manages the lifecycle of a specific "Error Node" game, handling player joins, submission attempts, and time-outs.

### 10.1 Session State Schema
The state represents the "Error Node" currently under transformation.

```elixir
defmodule KnirvArena.ResolutionSession do
  use GenServer, restart: :temporary

  defstruct [
    :error_id,
    :failure_context,
    :bounty_amount,
    :start_time,
    :winning_trajectory,
    players: [],
    submissions: %{}
  ]

  def start_link(context) do
    GenServer.start_link(__MODULE__, context, name: via_tuple(context["error_id"]))
  end

  defp via_tuple(id), do: {:via, Registry, {KnirvArena.SessionRegistry, id}}

  @impl true
  def init(context) do
    # Broadcast the new "Error Node" to all public TS clients via Phoenix PubSub
    Phoenix.PubSub.broadcast(KnirvArena.PubSub, "arena:lobby", {:new_bounty, context})
    
    {:ok, %__MODULE__{
      error_id: context["error_id"],
      failure_context: context,
      bounty_amount: context["nrn_fee"],
      start_time: DateTime.utc_now()
    }}
  end
end
```

---

## 11. PHOENIX CHANNELS & TS FRONTEND ORCHESTRATION

To keep the **KNIRVARENA** responsive and decentralized, the TypeScript frontend connects via Phoenix Channels. This allows for real-time adversarial feedback—if one player finds a partial solution, it can be broadcast as a "hint" (for a portion of the bounty) to others.

### 11.1 The Arena Channel (`arena_channel.ex`)
```elixir
defmodule KnirvArenaWeb.ArenaChannel do
  use KnirvArenaWeb, :channel

  def join("arena:resolution:" <> error_id, _payload, socket) do
    # Track player presence for reputation and NRN distribution
    send(self(), :after_join)
    {:ok, assign(socket, :error_id, error_id)}
  end

  def handle_in("submit_solution", %{"trajectory" => trajectory}, socket) do
    # Route submission to the specific ResolutionSession GenServer for validation
    error_id = socket.assigns.error_id
    case KnirvArena.ResolutionSession.validate(error_id, trajectory) do
      {:ok, :verified} ->
        broadcast!(socket, "resolution_achieved", %{winner: socket.assigns.user_id})
        {:reply, :ok, socket}
      {:error, reason} ->
        {:reply, {:error, %{reason: reason}}, socket}
    end
  end
end
```

### 11.2 TypeScript Client Interaction
The **Vite/TypeScript** frontend utilizes the standard Phoenix JS client to pipe adversarial inputs directly into the Erlang VM.

```typescript
// knirv-arena-client.ts
import { Socket } from "phoenix";

const socket = new Socket("/socket", { params: { token: userToken } });
socket.connect();

const channel = socket.channel(`arena:resolution:${errorId}`, {});
channel.join()
  .receive("ok", () => console.log("Entered Error Node resolution loop."))
  .receive("error", resp => { console.log("Access Denied", resp) });

function submitSolution(trajectory: any) {
  channel.push("submit_solution", { trajectory })
    .receive("ok", (msg) => console.log("Solution Validated!", msg))
    .receive("error", (reasons) => console.log("Logic Failure", reasons));
}
```
## 12. VALIDATION LOGIC & GO-BRIDGE SETTLEMENT

Once a player or automated agent submits a solution via the TypeScript frontend, the **KNIRVARENA** must coordinate with the **KNIRVSERVER** (Go) to perform a deterministic validation. This ensures that the proposed "Winning Trajectory" actually resolves the original failure within the private Enterprise DVE.

### 12.1 The Validation Request (Phoenix Side)
When the `ResolutionSession` receives a submission, it sends a validation request back through the Unix Domain Socket to the Go side.

```elixir
defmodule KnirvArena.Validator do
  @socket_path "/var/run/knirv/bridge.sock"

  def request_validation(error_id, trajectory) do
    payload = %{
      type: "VALIDATE_SUBMISSION",
      error_id: error_id,
      trajectory: trajectory,
      timestamp: DateTime.utc_now()
    } |> Jason.encode!()

    # Connect to the Go server's UDS listener
    case :gen_tcp.connect({:local, @socket_path}, 0, [:binary, active: false]) do
      {:ok, socket} ->
        :gen_tcp.send(socket, payload)
        # Wait for the Go server to run the solution in the sandbox
        case :gen_tcp.recv(socket, 0, 10_000) do
          {:ok, response} -> handle_validation_response(response)
          {:error, :timeout} -> {:error, :validation_timeout}
        end
      {:error, reason} -> {:error, reason}
    end
  end

  defp handle_validation_response(data) do
    case Jason.decode(data) do
      {:ok, %{"status" => "SUCCESS", "merkle_proof" => proof}} -> {:ok, proof}
      {:ok, %{"status" => "FAIL", "reason" => reason}} -> {:error, reason}
      _ -> {:error, :malformed_response}
    end
  end
end
```

### 12.2 Settlement & NRN Release (Go Side)
The **KNIRVSERVER** receives the trajectory, executes it in a headless DVE, and if successful, triggers the blockchain settlement on **KNIRVCHAIN**.

```go
// Go: Internal handler for UDS validation requests
func handleValidationRequest(req ValidationRequest) {
    isValid := sandbox.Execute(req.Trajectory, req.ErrorContext)
    
    if isValid {
        // 1. Commit to KNIRVCHAIN
        txHash := blockchain.ReleaseBounty(req.ErrorID, req.WinnerAddress)
        
        // 2. Respond to Phoenix with Proof of Settlement
        response := ValidationResponse{
            Status: "SUCCESS",
            MerkleProof: txHash,
        }
        sendUDSResponse(response)
    } else {
        sendUDSResponse(ValidationResponse{Status: "FAIL", Reason: "Logic mismatch"})
    }
}
```

---

## 13. KNOWLEDGE ASSET EXPORTING (JSONL GENERATION)

After a bounty is successfully settled, the **KNIRVARENA** transforms the session data into a structured **Knowledge Asset**. This is the primary fuel for the **LoRAX** fine-tuning process.

### 13.1 Transformation Logic
The goal is to move beyond simple "Correct/Incorrect" labels and capture the *transformation logic*.

```elixir
defmodule KnirvArena.AssetExporter do
  def format_for_lorax(session) do
    %{
      instruction: session.failure_context["instruction"],
      context: session.failure_context["environment_metadata"],
      error: %{
        output: session.failure_context["failed_output"],
        type: session.failure_context["error_class"]
      },
      solution: %{
        trajectory: session.winning_trajectory,
        final_output: session.verified_output
      },
      knowledge_hash: session.merkle_proof
    }
    |> Jason.encode!()
  end

  def export_batch(error_ids) do
    # Streams verified assets to the local /data/training directory
    File.open!("priv/training/latest_epoch.jsonl", [:append], fn file ->
      Enum.each(error_ids, fn id ->
        asset = get_session_data(id) |> format_for_lorax()
        IO.write(file, asset <> "\n")
      end)
    end)
  end
end
```

---

## 14. DECENTRALIZED MESH NETWORKING WITH LIBCLUSTER

To ensure the **KNIRVARENA** stays resilient even if individual nodes go offline, we use `libcluster`. This allows separate instances of the Arena (each paired with their own **KNIRVSERVER**) to form a distributed Erlang cluster automatically.

### 14.1 Cluster Configuration
The nodes discover each other via the **KNIRVCHAIN** peer list or a shared gossip protocol.

```elixir
# mix.exs
defp deps do
  [{:libcluster, "~> 3.3"}]
end

# config/runtime.exs
config :libcluster,
  topologies: [
    knirv_mesh: [
      strategy: Cluster.Strategy.Gossip,
      config: [
        port: 45892,
        if_addr: "0.0.0.0",
        multicast_addr: "230.0.0.1"
      ]
    ]
  ]
```

### 14.2 Cross-Node PubSub
When a new **ErrorNodeBounty** enters any node in the cluster, **Phoenix.PubSub** ensures every player across the entire network sees it, regardless of which physical server they are connected to.

```elixir
# Broadcasting from Node A
Phoenix.PubSub.broadcast(KnirvArena.PubSub, "arena:lobby", {:new_bounty, bounty})

# Node B, C, and D automatically receive this and push to their local TS clients.
```

---

## 15. PERFORMANCE OPTIMIZATION: CONCURRENCY CONTROLS

The Arena must handle thousands of concurrent players and automated agents. We utilize **OTP PartitionSupervisor** to ensure that high-traffic "Error Nodes" don't create bottlenecks for the rest of the network.

* **Partitioning:** `ResolutionSession` GenServers are distributed across multiple supervisors based on their `error_id` hash.
* **Backpressure:** If the validation UDS bridge is overwhelmed, the Arena uses a `Buffer` mechanism to queue submissions, preventing the Go server from crashing under high adversarial load.

## 16. REAL-TIME MONITORING WITH PHOENIX LIVEDASHBOARD

In a decentralized environment like the **KNIRV NETWORK**, visibility is everything. Because the **KNIRVARENA** manages thousands of ephemeral `ResolutionSession` GenServers, we utilize **Phoenix LiveDashboard** with custom telemetry to monitor the health of the "Error Node" transformation pipeline in real-time.

### 16.1 Custom Telemetry Metrics
We track the "Velocity of Resolution"—how quickly a `FailureContext` moves from the Go-bridge to a verified `Knowledge Asset`.

```elixir
# lib/knirv_arena_web/telemetry.ex
def metrics do
  [
    # Counter for total bounties ingested via UDS
    counter("knirv_arena.bounty.ingested_total"),
    
    # Distribution of time-to-resolution (in seconds)
    summary("knirv_arena.resolution.duration", unit: {:native, :second}),
    
    # Gauge for active Error Nodes currently in the mesh
    last_value("knirv_arena.active_sessions.count")
  ]
end
```

### 16.2 The "Observer" Effect
By exposing the BEAM’s internal state, operators can see which specific **Error Nodes** are causing bottlenecks. If a particular logic-branch is consistently failing validation, it may indicate a flawed `FailureContext` or an adversarial attack.

---

## 17. SECURITY: ANTI-SYBIL & REPUTATION METRICS

To protect the **NRN economy** from bad actors or low-quality automated "guessers," the Arena implements a multi-layered security protocol. Since we are operating outside of a centralized authority, we rely on **Cryptographic Reputation** and **Stake-slashing**.

### 17.1 Solver Reputation (The "Logic Score")
Every solver (human or agent) is indexed in the **KNIRVGRAPH** by their public key.
* **Positive Weight:** Successful, verified resolutions increase the solver's "Logic Score."
* **Negative Weight:** Submitting trajectories that fail the **KNIRVSERVER** sandbox validation results in a cooldown period or a score penalty.

### 17.2 The NRN Stake-Gate
For high-value Enterprise bounties, the Arena requires a **Proof of Stake** to participate:
1.  **Stake:** The solver locks a small amount of NRN into a temporary escrow.
2.  **Attempt:** The solver submits their trajectory.
3.  **Outcome:** * If **Valid**, the stake is returned along with the bounty.
    * If **Malicious/Spam**, the stake is "slashed"—sent to the Protocol Treasury—and the solver is blacklisted from that specific `ErrorNodeBounty`.

```elixir
defmodule KnirvArena.Security.Guard do
  def verify_solver_eligibility(user_id, bounty_tier) do
    # Check reputation score from KNIRVGRAPH (via Go-Bridge)
    score = KnirvArena.Bridge.get_reputation(user_id)
    
    cond do
      score < threshold_for(bounty_tier) -> {:error, :insufficient_reputation}
      is_blacklisted?(user_id) -> {:error, :blacklisted}
      true -> :ok
    end
  end
end
```

---

## 18. THE KNOWLEDGE ASSET LIFECYCLE (FINAL FLOW)

The ultimate goal of this entire backend is the production of the **Knowledge Asset**. The lifecycle is now complete:

1.  **Ingestion:** `FailureContext` arrives via Unix Socket from the Go **KNIRVSERVER**.
2.  **Activation:** A `ResolutionSession` GenServer spawns and broadcasts to the TS Frontend.
3.  **Competition:** Adversarial play occurs; multiple trajectories are tested.
4.  **Verification:** The winning trajectory is validated by the Go-side sandbox.
5.  **Settlement:** **NRN** fees are distributed via **KNIRVCHAIN**.
6.  **Refinement:** The **AssetExporter** saves the `Error + Context = Solution` pair to the local `.jsonl` for **LoRAX** training.
7.  **Evolution:** The next model epoch is trained, effectively "healing" the error that started the loop.

---

## 19. CONCLUSION: THE AUTONOMOUS RESOLUTION MESH

By leveraging the **Phoenix Edition** of the **Adversarial Training Framework**, the **KNIRV NETWORK** moves beyond static datasets. We have built a living, breathing orchestrator that treats AI failures as a commodity. 

The coupling of **Go** (for high-speed chain logic and sandboxing) and **Elixir** (for distributed game orchestration and real-time state) creates a unique infrastructure capable of processing enterprise-grade errors without compromising privacy. This "Error Node" loop ensures that every mistake made by the network's AI agents becomes a permanent, sellable, and verifiable asset in the **KNIRVGRAPH**.

The **KNIRVARENA** isn't just a game; it is the decentralized immune system of the KNIRV ecosystem.

---



---

