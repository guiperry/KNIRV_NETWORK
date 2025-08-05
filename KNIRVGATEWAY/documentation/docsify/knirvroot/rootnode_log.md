

---

**Source**: KNIRVROOT/docs/Troubleshooting/completedFixes/rootnode_log.md

gperry@cloud-eq:~/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP_PROTO$ go run . -gui -root
2025/05/23 13:18:44 [INFO] Block.Hash: Block #0 being converted to proto. Timestamp: 1678886400, Nonce: 0, NumTx: 0
2025/05/23 13:18:44 [INFO] ProtoBlock fields for block #0:
2025/05/23 13:18:44 [INFO] - BlockNumber: 0
2025/05/23 13:18:44 [INFO] - PrevHash: 
2025/05/23 13:18:44 [INFO] - Nonce: 0
2025/05/23 13:18:44 [INFO] - Timestamp: 2023-03-15 13:20:00 +0000 UTC
2025/05/23 13:18:44 [INFO] - Transactions count: 0
2025/05/23 13:18:44 [INFO] - ProposerAddress: KNIRVCHAINb53c1e30b8a578c091dd40612bfd1433991b4e09
2025/05/23 13:18:44 [INFO] Initialized deterministic Genesis Block. Hash: c3a7b1ecbfa373db8a37da060e1f2f8927ed5d09d2e4b1bc11d53797d0bd4d3a
2025/05/23 13:18:44 ***********STARTING KNIRVCHAIN***********
2025/05/23 13:18:44 VERSION: dev, OS: linux, Arch: amd64
2025/05/23 13:18:44 LOGFILE: KNIRVCHAIN.log
2025/05/23 13:18:44 Determined node role: Root
2025/05/23 13:18:44 Root constants set from main package: BlockchainAddress=KNIRVCHAINb53c1e30b8a578c091dd40612bfd1433991b4e09, RootchainURL=http://localhost:5000
2025/05/23 13:18:44 Setting Root constants from main package for configuration loading
2025/05/23 13:18:44 Viper: Root role detected. Skipping all config file operations for security.
2025/05/23 13:18:44 Applied default settings for role Root
2025/05/23 13:18:44 Using BlockchainAddress from constants: KNIRVCHAINb53c1e30b8a578c091dd40612bfd1433991b4e09
2025/05/23 13:18:44 Using role-specific data directory: /home/gperry/.config/KNIRVCHAIN/root_data for role Root
2025/05/23 13:18:44 DEBUG: GetBlockchainDatabasePath - Constructed finalPath: '/home/gperry/.config/KNIRVCHAIN/root_data/agent_root.db' for role Root
2025/05/23 13:18:44 Using role-specific data directory: /home/gperry/.config/KNIRVCHAIN/root_data for role Root
2025/05/23 13:18:44 Set reflection database path for Root: /home/gperry/.config/KNIRVCHAIN/root_data/reflectionDatabase/agent_reflection.db
2025/05/23 13:18:44 Created Root configuration directly from constants and settings matrix (no file operations)
2025/05/23 13:18:44 Attempting to fetch and store public IP information...
2025/05/23 13:18:44 Successfully parsed IPInfo response: IP=172.58.128.39, Country=US, ASN=
2025/05/23 13:18:44 Root role detected: Saving IP info to root-data/env.local instead of config
2025/05/23 13:18:44 Successfully saved IPInfo to root-data/env.local for Root role
2025/05/23 13:18:44 ROOTCHAIN_URL has been updated to: http://172.58.128.39:9999
2025/05/23 13:18:44 Updated ROOTCHAIN_URL to: http://172.58.128.39:9999 based on fetched public IP.
2025/05/23 13:18:44 [INFO] Root using configuration directly from constants (no config file operations).
2025/05/23 13:18:44 Root: Configured MinersAddress ('KNIRVCHAINb53c1e30b8a578c091dd40612bfd1433991b4e09') matches predefined blockchain identity. Using hardcoded private key from constants.go.
2025/05/23 13:18:44 [INFO] - Root: Successfully initialized wallet using hardcoded key for KNIRVCHAINb53c1e30b8a578c091dd40612bfd1433991b4e09.
2025/05/23 13:18:44 [INFO] - Root wallet setup process complete.
2025/05/23 13:18:44 Final BlockchainDatabasePath after Viper and flag processing: /home/gperry/.config/KNIRVCHAIN/root_data/agent_root.db
2025/05/23 13:18:44 Final BlockchainDatabasePath: /home/gperry/.config/KNIRVCHAIN/root_data/agent_root.db
2025/05/23 13:18:44 Installation is complete. Continuing with node initialization...
2025/05/23 13:18:44 Configuring Single-Node Mode...
2025/05/23 13:18:44 GUI enabled for Main Node (agent-root-9999)
2025/05/23 13:18:44 [agent-root-9999] Pre-initializing components for GUI node...
2025/05/23 13:18:44 DEBUG: Attempting to use DB path for GUI node: '/home/gperry/.config/KNIRVCHAIN/root_data/agent_root.db'
2025/05/23 13:18:44 [Root][agent-root-9999] Discovery Manager initialized. PeerID: 12D3KooWEPuQYZR5vjfzrqnwNwvvkcpx4qEXSuFHCcd7p5GFpk8L (ClientOnly: false)
2025/05/23 13:18:44 [Root][agent-root-9999] Listening on: /ip4/127.0.0.1/tcp/19999/p2p/12D3KooWEPuQYZR5vjfzrqnwNwvvkcpx4qEXSuFHCcd7p5GFpk8L
2025/05/23 13:18:44 [Root][agent-root-9999] Listening on: /ip4/192.168.12.122/tcp/19999/p2p/12D3KooWEPuQYZR5vjfzrqnwNwvvkcpx4qEXSuFHCcd7p5GFpk8L
2025/05/23 13:18:44 [Root][agent-root-9999] Listening on: /ip6/::1/tcp/19999/p2p/12D3KooWEPuQYZR5vjfzrqnwNwvvkcpx4qEXSuFHCcd7p5GFpk8L
2025/05/23 13:18:44 [Root][agent-root-9999] Listening on: /ip6/2607:fb91:2837:4588::126a/tcp/19999/p2p/12D3KooWEPuQYZR5vjfzrqnwNwvvkcpx4qEXSuFHCcd7p5GFpk8L
2025/05/23 13:18:44 [INFO] Found existing blockchain data for key 'agent-root-9999'. Attempting to load.
2025/05/23 13:18:44 mDNS: Found local dev: 12D3KooWEuyiDLwXt66KaN7VHNk2Fc1fm9cCK19LmCLeqGdL8uVB, attempting connection
2025/05/23 13:18:44 [INFO] Block.Hash: Block #0 being converted to proto. Timestamp: 1678886400, Nonce: 0, NumTx: 0
2025/05/23 13:18:44 [INFO] ProtoBlock fields for block #0:
2025/05/23 13:18:44 [INFO] - BlockNumber: 0
2025/05/23 13:18:44 [INFO] - PrevHash: 
2025/05/23 13:18:44 [INFO] - Nonce: 0
2025/05/23 13:18:44 [INFO] - Timestamp: 2023-03-15 13:20:00 +0000 UTC
2025/05/23 13:18:44 [INFO] - Transactions count: 0
2025/05/23 13:18:44 [INFO] - ProposerAddress: KNIRVCHAINb53c1e30b8a578c091dd40612bfd1433991b4e09
2025/05/23 13:18:44 [INFO] Block.Hash: Block #0 being converted to proto. Timestamp: 1678886400, Nonce: 0, NumTx: 0
2025/05/23 13:18:44 [INFO] ProtoBlock fields for block #0:
2025/05/23 13:18:44 [INFO] - BlockNumber: 0
2025/05/23 13:18:44 [INFO] - PrevHash: 
2025/05/23 13:18:44 [INFO] - Nonce: 0
2025/05/23 13:18:44 [INFO] - Timestamp: 2023-03-15 13:20:00 +0000 UTC
2025/05/23 13:18:44 [INFO] - Transactions count: 0
2025/05/23 13:18:44 [INFO] - ProposerAddress: KNIRVCHAINb53c1e30b8a578c091dd40612bfd1433991b4e09
2025/05/23 13:18:44 [INFO] Successfully loaded existing blockchain for key 'agent-root-9999'. Genesis block matches.
2025/05/23 13:18:44 [agent-root-9999] GUI node components pre-initialized.
2025/05/23 13:18:44 Starting in single-node mode...
2025/05/23 13:18:44 Starting Main Node (HTTP: 9999, P2P: 19999, GUI: true, DB: /home/gperry/.config/KNIRVCHAIN/root_data/agent_root.db)
2025/05/23 13:18:44 [agent-root-9999] Waiting for P2P Consensus Manager initialization...
2025/05/23 13:18:44 [Root][agent-root-9999] P2P consensus manager subscribed to topics: agent-root-9999.blocks, agent-root-9999.transactions
2025/05/23 13:18:44 [Root][agent-root-9999] Registered chain sync handler for protocol /agent/chain-sync/1.0.0
2025/05/23 13:18:44 [agent-root-9999] Initializing node with pre-initialized components...
2025/05/23 13:18:44 [Root][agent-root-9999] Starting P2P consensus manager...
2025/05/23 13:18:44 [Root][agent-root-9999] P2P consensus manager started successfully.
2025/05/23 13:18:44 [agent-root-9999] Legacy Consensus Manager initialized.
2025/05/23 13:18:44 Port 9999 is in use, trying next port for chain agent-root-9999
2025/05/23 13:18:44 Chain agent-root-9999: HTTP server will use port 10000 instead of configured port 9999
2025/05/23 13:18:44 BlockchainServer for chain agent-root-9999 prepared for port 10000
2025/05/23 13:18:44 [agent-root-9999] Blockchain Server initialized to listen on :9999.
2025/05/23 13:18:44 [agent-root-9999] Starting Server on port 9999...
2025/05/23 13:18:44 Starting HTTP server listener for chain agent-root-9999 on port: 10000
2025/05/23 13:18:44 HTTP server ListenAndServe error for chain agent-root-9999: listen tcp :9999: bind: address already in use
2025/05/23 13:18:44 [agent-root-9999] ERROR: Blockchain HTTP Server failed: failed to start HTTP server listener for chain agent-root-9999: listen tcp :9999: bind: address already in use
2025/05/23 13:18:44 [agent-root-9999] Blockchain HTTP Server stopped.
2025/05/23 13:18:44 [agent-root-9999] P2P Consensus Manager initialized successfully.
2025/05/23 13:18:44 [agent-root-9999] Starting GUI on main goroutine...
2025/05/23 13:18:44 [agent-root-9999] Initializing payment processor for Root role...
2025/05/23 13:18:44 Loaded existing master wallet with address: KNIRVCHAIN3d160f3de1a82778eb182b259011a2593000cb92 from key 'master_wallet_key'
2025/05/23 13:18:44 [agent-root-9999] Using master wallet with address KNIRVCHAIN3d160f3de1a82778eb182b259011a2593000cb92 for token disbursement
2025/05/23 13:18:44 Initializing payment processor...
2025/05/23 13:18:44 Using master wallet with address: KNIRVCHAIN3d160f3de1a82778eb182b259011a2593000cb92 for token disbursement
2025/05/23 13:18:44 [agent-root-9999] Payment processor started successfully
2025/05/23 13:18:44 [agent-root-9999] Payment processor initialized successfully
2025/05/23 13:18:44 Warning: Could not get cancel function from main, creating a new one
2025/05/23 13:18:44 Warning: Could not get wait group from main, creating a new one
2025/05/23 13:18:44 Starting API and UI server on port 3000
2025/05/23 13:18:44 Starting payment processor webhook server on port 8088
2025/05/23 13:18:44 Blockchain server detected at port 9999
2025/05/23 13:18:45 Successfully parsed IPInfo response: IP=172.58.128.39, Country=US, ASN=
2025/05/23 13:18:45 Successfully fetched public IP from ipinfo.io: 172.58.128.39
2025/05/23 13:18:45 [Root][agent-root-9999] Successfully saved IPInfo to altgui/env.local for Root role
2025/05/23 13:18:45 [Root][agent-root-9999] Attempting to update NEXT_PUBLIC_BACKEND_URL in altgui/backend.config to http://172.58.128.39:9999
2025/05/23 13:18:45 [Root][agent-root-9999] Successfully updated GUI backend URL for altgui/backend.config
Overriding existing handler for signal 10. Set JSC_SIGNAL_FOR_GC if you want WebKit to use a different signal
2025/05/23 13:18:46 Loaded icon from altgui/out/favicon.ico
SIGABRT: abort
PC=0x7f02bca059fc m=8 sigcode=18446744073709551610
signal arrived during cgo execution

goroutine 316 gp=0xc0002788c0 m=8 mp=0xc000536008 [syscall]:
runtime.cgocall(0x125c3a0, 0xc0002d8718)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/cgocall.go:167 +0x4b fp=0xc0002d86f0 sp=0xc0002d86b8 pc=0x473b0b
github.com/getlantern/systray._Cfunc_nativeLoop()
	_cgo_gotypes.go:142 +0x47 fp=0xc0002d8718 sp=0xc0002d86f0 pc=0x11c9167
github.com/getlantern/systray.nativeLoop(...)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/getlantern/systray@v1.2.2/systray_nonwindows.go:18
github.com/getlantern/systray.Run(0x1b34b70?, 0x26456a0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/getlantern/systray@v1.2.2/systray.go:78 +0x19 fp=0xc0002d8738 sp=0xc0002d8718 pc=0x11c8879
main.(*SystrayManager).Run(0xc000345260)
	/home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP_PROTO/systray.go:70 +0x157 fp=0xc0002d87c8 sp=0xc0002d8738 pc=0x1242c97
main.(*GUI).createWebViewWindow.gowrap2()
	/home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP_PROTO/altgui.go:521 +0x25 fp=0xc0002d87e0 sp=0xc0002d87c8 pc=0x11cfae5
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc0002d87e8 sp=0xc0002d87e0 pc=0x481ec1
created by main.(*GUI).createWebViewWindow in goroutine 299
	/home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP_PROTO/altgui.go:521 +0x1bb

goroutine 1 gp=0xc0000061c0 m=nil [select (no cases), locked to thread]:
runtime.gopark(0x44efa0?, 0xc000780560?, 0xc0?, 0x61?, 0x11cfd29?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc0006dcf80 sp=0xc0006dcf60 pc=0x479f4e
runtime.block()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:103 +0x26 fp=0xc0006dcfb0 sp=0xc0006dcf80 pc=0x456c06
main.InitializeGUI(0xc000001d40, 0xc00012a108, 0xc000686e60, 0xc000a28090, 0xc000404788, 0xc000810480, {0x15c79b2, 0x4}, 0xc0002a81e0, 0x0)
	/home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP_PROTO/altgui.go:729 +0x1ae fp=0xc0006dd098 sp=0xc0006dcfb0 pc=0x11cfd2e
main.main()
	/home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP_PROTO/main.go:1056 +0x6049 fp=0xc0006ddf50 sp=0xc0006dd098 pc=0x12143e9
runtime.main()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:272 +0x28b fp=0xc0006ddfe0 sp=0xc0006ddf50 pc=0x4454ab
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc0006ddfe8 sp=0xc0006ddfe0 pc=0x481ec1

goroutine 2 gp=0xc000006c40 m=nil [force gc (idle)]:
runtime.gopark(0x0?, 0x0?, 0x0?, 0x0?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00008cfa8 sp=0xc00008cf88 pc=0x479f4e
runtime.goparkunlock(...)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:430
runtime.forcegchelper()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:337 +0xb3 fp=0xc00008cfe0 sp=0xc00008cfa8 pc=0x4457f3
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00008cfe8 sp=0xc00008cfe0 pc=0x481ec1
created by runtime.init.7 in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:325 +0x1a

goroutine 3 gp=0xc000007180 m=nil [GC sweep wait]:
runtime.gopark(0x1?, 0x0?, 0x0?, 0x0?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00008d780 sp=0xc00008d760 pc=0x479f4e
runtime.goparkunlock(...)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:430
runtime.bgsweep(0xc000054180)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/mgcsweep.go:317 +0xdf fp=0xc00008d7c8 sp=0xc00008d780 pc=0x42ee1f
runtime.gcenable.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/mgc.go:203 +0x25 fp=0xc00008d7e0 sp=0xc00008d7c8 pc=0x4234e5
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00008d7e8 sp=0xc00008d7e0 pc=0x481ec1
created by runtime.gcenable in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/mgc.go:203 +0x66

goroutine 4 gp=0xc000007340 m=nil [GC scavenge wait]:
runtime.gopark(0x10000?, 0x1b1a6c8?, 0x0?, 0x0?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00008df78 sp=0xc00008df58 pc=0x479f4e
runtime.goparkunlock(...)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:430
runtime.(*scavengerState).park(0x2621380)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/mgcscavenge.go:425 +0x49 fp=0xc00008dfa8 sp=0xc00008df78 pc=0x42c7e9
runtime.bgscavenge(0xc000054180)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/mgcscavenge.go:658 +0x59 fp=0xc00008dfc8 sp=0xc00008dfa8 pc=0x42cd79
runtime.gcenable.gowrap2()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/mgc.go:204 +0x25 fp=0xc00008dfe0 sp=0xc00008dfc8 pc=0x423485
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00008dfe8 sp=0xc00008dfe0 pc=0x481ec1
created by runtime.gcenable in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/mgc.go:204 +0xa5

goroutine 5 gp=0xc000007c00 m=nil [finalizer wait]:
runtime.gopark(0xc00008c648?, 0x4194e5?, 0xb0?, 0x1?, 0xc0000061c0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00008c620 sp=0xc00008c600 pc=0x479f4e
runtime.runfinq()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/mfinal.go:193 +0x107 fp=0xc00008c7e0 sp=0xc00008c620 pc=0x422567
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00008c7e8 sp=0xc00008c7e0 pc=0x481ec1
created by runtime.createfing in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/mfinal.go:163 +0x3d

goroutine 6 gp=0xc000007dc0 m=nil [select]:
runtime.gopark(0xc00008e760?, 0x2?, 0x0?, 0x0?, 0xc00008e71c?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00009eda8 sp=0xc00009ed88 pc=0x479f4e
runtime.selectgo(0xc00009ef60, 0xc00008e718, 0x0?, 0x0, 0x0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc00009eed0 sp=0xc00009eda8 pc=0x4573c5
github.com/ipfs/go-log/writer.(*MirrorWriter).logRoutine(0xc0000403c0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/ipfs/go-log@v1.0.5/writer/writer.go:71 +0x105 fp=0xc00009efc8 sp=0xc00009eed0 pc=0x10757c5
github.com/ipfs/go-log/writer.NewMirrorWriter.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/ipfs/go-log@v1.0.5/writer/writer.go:36 +0x25 fp=0xc00009efe0 sp=0xc00009efc8 pc=0x1075605
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00009efe8 sp=0xc00009efe0 pc=0x481ec1
created by github.com/ipfs/go-log/writer.NewMirrorWriter in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/ipfs/go-log@v1.0.5/writer/writer.go:36 +0xbf

goroutine 7 gp=0xc0001fea80 m=nil [select]:
runtime.gopark(0xc00008ef78?, 0x3?, 0x90?, 0x0?, 0xc00008ef72?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00011ee10 sp=0xc00011edf0 pc=0x479f4e
runtime.selectgo(0xc00011ef78, 0xc00008ef6c, 0xc0001a4c00?, 0x0, 0x0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc00011ef38 sp=0xc00011ee10 pc=0x4573c5
go.opencensus.io/stats/view.(*worker).start(0xc0001a4c00)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/go.opencensus.io@v0.24.0/stats/view/worker.go:292 +0x9f fp=0xc00011efc8 sp=0xc00011ef38 pc=0x109d0ff
go.opencensus.io/stats/view.init.0.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/go.opencensus.io@v0.24.0/stats/view/worker.go:34 +0x25 fp=0xc00011efe0 sp=0xc00011efc8 pc=0x109c465
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00011efe8 sp=0xc00011efe0 pc=0x481ec1
created by go.opencensus.io/stats/view.init.0 in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/go.opencensus.io@v0.24.0/stats/view/worker.go:34 +0x8d

goroutine 8 gp=0xc0001fec40 m=nil [chan receive]:
runtime.gopark(0xc00008f760?, 0x5f53e5?, 0x40?, 0x45?, 0x1b561a0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00008f718 sp=0xc00008f6f8 pc=0x479f4e
runtime.chanrecv(0xc000059500, 0x0, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/chan.go:639 +0x41c fp=0xc00008f790 sp=0xc00008f718 pc=0x412b5c
runtime.chanrecv1(0x0?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/chan.go:489 +0x12 fp=0xc00008f7b8 sp=0xc00008f790 pc=0x412712
runtime.unique_runtime_registerUniqueMapCleanup.func1(...)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/mgc.go:1732
runtime.unique_runtime_registerUniqueMapCleanup.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/mgc.go:1735 +0x2f fp=0xc00008f7e0 sp=0xc00008f7b8 pc=0x4264ef
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00008f7e8 sp=0xc00008f7e0 pc=0x481ec1
created by unique.runtime_registerUniqueMapCleanup in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/mgc.go:1730 +0x96

goroutine 15 gp=0xc000286380 m=nil [select]:
runtime.gopark(0xc0003196a8?, 0x2?, 0x70?, 0x0?, 0xc00031963c?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc0003194d8 sp=0xc0003194b8 pc=0x479f4e
runtime.selectgo(0xc0003196a8, 0xc000319638, 0x0?, 0x0, 0x0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc000319600 sp=0xc0003194d8 pc=0x4573c5
github.com/libp2p/go-libp2p/p2p/transport/quicreuse.(*reuse).gc(0xc00036d6d0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/transport/quicreuse/reuse.go:219 +0xfe fp=0xc0003197c8 sp=0xc000319600 pc=0xcec89e
github.com/libp2p/go-libp2p/p2p/transport/quicreuse.newReuse.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/transport/quicreuse/reuse.go:194 +0x25 fp=0xc0003197e0 sp=0xc0003197c8 pc=0xcec765
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc0003197e8 sp=0xc0003197e0 pc=0x481ec1
created by github.com/libp2p/go-libp2p/p2p/transport/quicreuse.newReuse in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/transport/quicreuse/reuse.go:194 +0x145

goroutine 16 gp=0xc000286700 m=nil [select]:
runtime.gopark(0xc000319ea8?, 0x2?, 0x70?, 0x0?, 0xc000319e3c?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00009fcd8 sp=0xc00009fcb8 pc=0x479f4e
runtime.selectgo(0xc00009fea8, 0xc000319e38, 0x798835?, 0x0, 0xc000248230?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc00009fe00 sp=0xc00009fcd8 pc=0x4573c5
github.com/libp2p/go-libp2p/p2p/transport/quicreuse.(*reuse).gc(0xc00036d720)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/transport/quicreuse/reuse.go:219 +0xfe fp=0xc00009ffc8 sp=0xc00009fe00 pc=0xcec89e
github.com/libp2p/go-libp2p/p2p/transport/quicreuse.newReuse.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/transport/quicreuse/reuse.go:194 +0x25 fp=0xc00009ffe0 sp=0xc00009ffc8 pc=0xcec765
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00009ffe8 sp=0xc00009ffe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p/p2p/transport/quicreuse.newReuse in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/transport/quicreuse/reuse.go:194 +0x145

goroutine 42 gp=0xc0002868c0 m=nil [select]:
runtime.gopark(0xc000a39f68?, 0x3?, 0x10?, 0x9e?, 0xc000a39ef2?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000a39d88 sp=0xc000a39d68 pc=0x479f4e
runtime.selectgo(0xc000a39f68, 0xc000a39eec, 0x22?, 0x0, 0x412020?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc000a39eb0 sp=0xc000a39d88 pc=0x4573c5
github.com/libp2p/go-libp2p/p2p/net/swarm.(*connectednessEventEmitter).runEmitter(0xc0001161b0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/connectedness_event_emitter.go:93 +0x116 fp=0xc000a39fc8 sp=0xc000a39eb0 pc=0xb9a796
github.com/libp2p/go-libp2p/p2p/net/swarm.newConnectednessEventEmitter.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/connectedness_event_emitter.go:47 +0x25 fp=0xc000a39fe0 sp=0xc000a39fc8 pc=0xb9a225
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc000a39fe8 sp=0xc000a39fe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p/p2p/net/swarm.newConnectednessEventEmitter in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/connectedness_event_emitter.go:47 +0x185

goroutine 23 gp=0xc000286a80 m=nil [GC worker (idle)]:
runtime.gopark(0xc0002ca380?, 0x0?, 0x1?, 0x18?, 0xc00008e770?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00008e738 sp=0xc00008e718 pc=0x479f4e
runtime.gcBgMarkWorker(0xc0002ca620)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/mgc.go:1363 +0xe9 fp=0xc00008e7c8 sp=0xc00008e738 pc=0x425809
runtime.gcBgMarkStartWorkers.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/mgc.go:1279 +0x25 fp=0xc00008e7e0 sp=0xc00008e7c8 pc=0x4256e5
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00008e7e8 sp=0xc00008e7e0 pc=0x481ec1
created by runtime.gcBgMarkStartWorkers in goroutine 36
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/mgc.go:1279 +0x105

goroutine 10 gp=0xc000104540 m=nil [IO wait]:
runtime.gopark(0x678d36?, 0xc00089cf05?, 0x40?, 0x45?, 0xb?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000189700 sp=0xc0001896e0 pc=0x479f4e
runtime.netpollblock(0x49d3b8?, 0x40ff66?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/netpoll.go:575 +0xf7 fp=0xc000189738 sp=0xc000189700 pc=0x43ddd7
internal/poll.runtime_pollWait(0x7f0270668680, 0x72)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/netpoll.go:351 +0x85 fp=0xc000189758 sp=0xc000189738 pc=0x479245
internal/poll.(*pollDesc).wait(0xc0002a0180?, 0xc0002fe000?, 0x0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_poll_runtime.go:84 +0x27 fp=0xc000189780 sp=0xc000189758 pc=0x50a5c7
internal/poll.(*pollDesc).waitRead(...)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_poll_runtime.go:89
internal/poll.(*FD).Read(0xc0002a0180, {0xc0002fe000, 0x1000, 0x1000})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_unix.go:165 +0x27a fp=0xc000189818 sp=0xc000189780 pc=0x50b8ba
net.(*netFD).Read(0xc0002a0180, {0xc0002fe000?, 0x50dbc5?, 0xc0002a0180?})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/fd_posix.go:55 +0x25 fp=0xc000189860 sp=0xc000189818 pc=0x61be05
net.(*conn).Read(0xc000156458, {0xc0002fe000?, 0xc000189a78?, 0x699bc4?})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/net.go:189 +0x45 fp=0xc0001898a8 sp=0xc000189860 pc=0x62dc45
net.(*TCPConn).Read(0x4fb?, {0xc0002fe000?, 0x18?, 0x500?})
	<autogenerated>:1 +0x25 fp=0xc0001898d8 sp=0xc0001898a8 pc=0x6454e5
crypto/tls.(*atLeastReader).Read(0xc00089a5b8, {0xc0002fe000?, 0x0?, 0xc00089a5b8?})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/crypto/tls/conn.go:809 +0x3b fp=0xc000189920 sp=0xc0001898d8 pc=0x69d43b
bytes.(*Buffer).ReadFrom(0xc0004269b8, {0x1b271a0, 0xc00089a5b8})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/bytes/buffer.go:211 +0x98 fp=0xc000189978 sp=0xc000189920 pc=0x53bcb8
crypto/tls.(*Conn).readFromUntil(0xc000426708, {0x1b26640, 0xc000156458}, 0xc000189a10?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/crypto/tls/conn.go:831 +0xde fp=0xc0001899b0 sp=0xc000189978 pc=0x69d61e
crypto/tls.(*Conn).readRecordOrCCS(0xc000426708, 0x0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/crypto/tls/conn.go:629 +0x3cf fp=0xc000189c28 sp=0xc0001899b0 pc=0x69a72f
crypto/tls.(*Conn).readRecord(...)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/crypto/tls/conn.go:591
crypto/tls.(*Conn).Read(0xc000426708, {0xc00041b000, 0x1000, 0x11?})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/crypto/tls/conn.go:1385 +0x150 fp=0xc000189c98 sp=0xc000189c28 pc=0x6a0f90
bufio.(*Reader).Read(0xc00027ff20, {0xc0001a83c0, 0x9, 0x79340e?})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/bufio/bufio.go:241 +0x197 fp=0xc000189cd0 sp=0xc000189c98 pc=0x6e3417
io.ReadAtLeast({0x1b253e0, 0xc00027ff20}, {0xc0001a83c0, 0x9, 0x9}, 0x9)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/io/io.go:335 +0x90 fp=0xc000189d18 sp=0xc000189cd0 pc=0x4ba2d0
io.ReadFull(...)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/io/io.go:354
net/http.http2readFrameHeader({0xc0001a83c0, 0x9, 0x74ee68?}, {0x1b253e0?, 0xc00027ff20?})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/http/h2_bundle.go:1642 +0x65 fp=0xc000189d68 sp=0xc000189d18 pc=0x726e45
net/http.(*http2Framer).ReadFrame(0xc0001a8380)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/http/h2_bundle.go:1909 +0x85 fp=0xc000189e10 sp=0xc000189d68 pc=0x727585
net/http.(*http2clientConnReadLoop).run(0xc000189fa8)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/http/h2_bundle.go:9496 +0xda fp=0xc000189f60 sp=0xc000189e10 pc=0x74a85a
net/http.(*http2ClientConn).readLoop(0xc000002000)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/http/h2_bundle.go:9392 +0x7c fp=0xc000189fc8 sp=0xc000189f60 pc=0x749e3c
net/http.(*http2Transport).newClientConn.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/http/h2_bundle.go:8006 +0x25 fp=0xc000189fe0 sp=0xc000189fc8 pc=0x742bc5
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc000189fe8 sp=0xc000189fe0 pc=0x481ec1
created by net/http.(*http2Transport).newClientConn in goroutine 9
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/http/h2_bundle.go:8006 +0xd1b

goroutine 24 gp=0xc0002876c0 m=nil [GC worker (idle)]:
runtime.gopark(0x0?, 0x0?, 0x0?, 0x0?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000088f38 sp=0xc000088f18 pc=0x479f4e
runtime.gcBgMarkWorker(0xc0002ca620)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/mgc.go:1363 +0xe9 fp=0xc000088fc8 sp=0xc000088f38 pc=0x425809
runtime.gcBgMarkStartWorkers.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/mgc.go:1279 +0x25 fp=0xc000088fe0 sp=0xc000088fc8 pc=0x4256e5
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc000088fe8 sp=0xc000088fe0 pc=0x481ec1
created by runtime.gcBgMarkStartWorkers in goroutine 36
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/mgc.go:1279 +0x105

goroutine 25 gp=0xc000287880 m=nil [GC worker (idle)]:
runtime.gopark(0x0?, 0x0?, 0x0?, 0x0?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000089738 sp=0xc000089718 pc=0x479f4e
runtime.gcBgMarkWorker(0xc0002ca620)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/mgc.go:1363 +0xe9 fp=0xc0000897c8 sp=0xc000089738 pc=0x425809
runtime.gcBgMarkStartWorkers.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/mgc.go:1279 +0x25 fp=0xc0000897e0 sp=0xc0000897c8 pc=0x4256e5
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc0000897e8 sp=0xc0000897e0 pc=0x481ec1
created by runtime.gcBgMarkStartWorkers in goroutine 36
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/mgc.go:1279 +0x105

goroutine 26 gp=0xc000287a40 m=nil [GC worker (idle)]:
runtime.gopark(0x0?, 0x0?, 0x0?, 0x0?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000089f38 sp=0xc000089f18 pc=0x479f4e
runtime.gcBgMarkWorker(0xc0002ca620)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/mgc.go:1363 +0xe9 fp=0xc000089fc8 sp=0xc000089f38 pc=0x425809
runtime.gcBgMarkStartWorkers.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/mgc.go:1279 +0x25 fp=0xc000089fe0 sp=0xc000089fc8 pc=0x4256e5
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc000089fe8 sp=0xc000089fe0 pc=0x481ec1
created by runtime.gcBgMarkStartWorkers in goroutine 36
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/mgc.go:1279 +0x105

goroutine 27 gp=0xc000287c00 m=nil [GC worker (idle)]:
runtime.gopark(0x8cf63ed8491e?, 0x0?, 0x0?, 0x0?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00008a738 sp=0xc00008a718 pc=0x479f4e
runtime.gcBgMarkWorker(0xc0002ca620)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/mgc.go:1363 +0xe9 fp=0xc00008a7c8 sp=0xc00008a738 pc=0x425809
runtime.gcBgMarkStartWorkers.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/mgc.go:1279 +0x25 fp=0xc00008a7e0 sp=0xc00008a7c8 pc=0x4256e5
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00008a7e8 sp=0xc00008a7e0 pc=0x481ec1
created by runtime.gcBgMarkStartWorkers in goroutine 36
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/mgc.go:1279 +0x105

goroutine 28 gp=0xc000287dc0 m=nil [GC worker (idle)]:
runtime.gopark(0x19a19b0?, 0xc0004c68a0?, 0x1a?, 0xa?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00008af38 sp=0xc00008af18 pc=0x479f4e
runtime.gcBgMarkWorker(0xc0002ca620)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/mgc.go:1363 +0xe9 fp=0xc00008afc8 sp=0xc00008af38 pc=0x425809
runtime.gcBgMarkStartWorkers.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/mgc.go:1279 +0x25 fp=0xc00008afe0 sp=0xc00008afc8 pc=0x4256e5
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00008afe8 sp=0xc00008afe0 pc=0x481ec1
created by runtime.gcBgMarkStartWorkers in goroutine 36
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/mgc.go:1279 +0x105

goroutine 29 gp=0xc0004d2000 m=nil [GC worker (idle)]:
runtime.gopark(0x8cf63e3872a0?, 0x0?, 0x0?, 0x0?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00008b738 sp=0xc00008b718 pc=0x479f4e
runtime.gcBgMarkWorker(0xc0002ca620)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/mgc.go:1363 +0xe9 fp=0xc00008b7c8 sp=0xc00008b738 pc=0x425809
runtime.gcBgMarkStartWorkers.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/mgc.go:1279 +0x25 fp=0xc00008b7e0 sp=0xc00008b7c8 pc=0x4256e5
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00008b7e8 sp=0xc00008b7e0 pc=0x481ec1
created by runtime.gcBgMarkStartWorkers in goroutine 36
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/mgc.go:1279 +0x105

goroutine 30 gp=0xc0004d21c0 m=nil [GC worker (idle)]:
runtime.gopark(0x8cf633ebbec8?, 0x0?, 0x0?, 0x0?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00008bf38 sp=0xc00008bf18 pc=0x479f4e
runtime.gcBgMarkWorker(0xc0002ca620)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/mgc.go:1363 +0xe9 fp=0xc00008bfc8 sp=0xc00008bf38 pc=0x425809
runtime.gcBgMarkStartWorkers.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/mgc.go:1279 +0x25 fp=0xc00008bfe0 sp=0xc00008bfc8 pc=0x4256e5
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00008bfe8 sp=0xc00008bfe0 pc=0x481ec1
created by runtime.gcBgMarkStartWorkers in goroutine 36
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/mgc.go:1279 +0x105

goroutine 38 gp=0xc0001fe000 m=nil [select]:
runtime.gopark(0xc000317780?, 0x2?, 0x70?, 0x0?, 0xc00031773c?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc0003175d8 sp=0xc0003175b8 pc=0x479f4e
runtime.selectgo(0xc000317780, 0xc000317738, 0xc0003177f8?, 0x0, 0xc00031fcb0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc000317700 sp=0xc0003175d8 pc=0x4573c5
github.com/syndtr/goleveldb/leveldb/util.(*BufferPool).drain(0xc0003ae000)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/syndtr/goleveldb@v1.0.0/leveldb/util/buffer_pool.go:206 +0xc8 fp=0xc0003177c8 sp=0xc000317700 pc=0x10fbd68
github.com/syndtr/goleveldb/leveldb/util.NewBufferPool.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/syndtr/goleveldb@v1.0.0/leveldb/util/buffer_pool.go:237 +0x25 fp=0xc0003177e0 sp=0xc0003177c8 pc=0x10fc0a5
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc0003177e8 sp=0xc0003177e0 pc=0x481ec1
created by github.com/syndtr/goleveldb/leveldb/util.NewBufferPool in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/syndtr/goleveldb@v1.0.0/leveldb/util/buffer_pool.go:237 +0x17b

goroutine 31 gp=0xc0001fe380 m=nil [select]:
runtime.gopark(0xc00031bf88?, 0x2?, 0x0?, 0x0?, 0xc00031bed4?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00031bd58 sp=0xc00031bd38 pc=0x479f4e
runtime.selectgo(0xc00031bf88, 0xc00031bed0, 0x0?, 0x0, 0x0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc00031be80 sp=0xc00031bd58 pc=0x4573c5
github.com/syndtr/goleveldb/leveldb.(*DB).compactionError(0xc000492340)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/syndtr/goleveldb@v1.0.0/leveldb/db_compaction.go:90 +0x149 fp=0xc00031bfc8 sp=0xc00031be80 pc=0x111cdc9
github.com/syndtr/goleveldb/leveldb.openDB.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/syndtr/goleveldb@v1.0.0/leveldb/db.go:142 +0x25 fp=0xc00031bfe0 sp=0xc00031bfc8 pc=0x1117405
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00031bfe8 sp=0xc00031bfe0 pc=0x481ec1
created by github.com/syndtr/goleveldb/leveldb.openDB in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/syndtr/goleveldb@v1.0.0/leveldb/db.go:142 +0x447

goroutine 32 gp=0xc0001fe8c0 m=nil [select]:
runtime.gopark(0xc00031c798?, 0x2?, 0x70?, 0x0?, 0xc00031c764?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc0000a1e00 sp=0xc0000a1de0 pc=0x479f4e
runtime.selectgo(0xc0000a1f98, 0xc00031c760, 0x223?, 0x0, 0x0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc0000a1f28 sp=0xc0000a1e00 pc=0x4573c5
github.com/syndtr/goleveldb/leveldb.(*DB).mpoolDrain(0xc000492340)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/syndtr/goleveldb@v1.0.0/leveldb/db_state.go:101 +0x9c fp=0xc0000a1fc8 sp=0xc0000a1f28 pc=0x1125d7c
github.com/syndtr/goleveldb/leveldb.openDB.gowrap2()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/syndtr/goleveldb@v1.0.0/leveldb/db.go:143 +0x25 fp=0xc0000a1fe0 sp=0xc0000a1fc8 pc=0x11173a5
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc0000a1fe8 sp=0xc0000a1fe0 pc=0x481ec1
created by github.com/syndtr/goleveldb/leveldb.openDB in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/syndtr/goleveldb@v1.0.0/leveldb/db.go:143 +0x485

goroutine 33 gp=0xc0001fefc0 m=nil [select]:
runtime.gopark(0xc00031cef8?, 0x3?, 0x0?, 0x0?, 0xc00031ce36?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00031ccb8 sp=0xc00031cc98 pc=0x479f4e
runtime.selectgo(0xc00031cef8, 0xc00031ce30, 0x798835?, 0x0, 0xc00024bc70?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc00031cde0 sp=0xc00031ccb8 pc=0x4573c5
github.com/syndtr/goleveldb/leveldb.(*DB).tCompaction(0xc000492340)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/syndtr/goleveldb@v1.0.0/leveldb/db_compaction.go:825 +0x6b7 fp=0xc00031cfc8 sp=0xc00031cde0 pc=0x11222d7
github.com/syndtr/goleveldb/leveldb.openDB.gowrap3()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/syndtr/goleveldb@v1.0.0/leveldb/db.go:149 +0x25 fp=0xc00031cfe0 sp=0xc00031cfc8 pc=0x1117345
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00031cfe8 sp=0xc00031cfe0 pc=0x481ec1
created by github.com/syndtr/goleveldb/leveldb.openDB in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/syndtr/goleveldb@v1.0.0/leveldb/db.go:149 +0x4f6

goroutine 50 gp=0xc0001ff180 m=nil [select]:
runtime.gopark(0xc00031d768?, 0x2?, 0x0?, 0x0?, 0xc00031d754?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00031d5f0 sp=0xc00031d5d0 pc=0x479f4e
runtime.selectgo(0xc00031d768, 0xc00031d750, 0x7900d4?, 0x0, 0xc000249dc0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc00031d718 sp=0xc00031d5f0 pc=0x4573c5
github.com/syndtr/goleveldb/leveldb.(*DB).mCompaction(0xc000492340)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/syndtr/goleveldb@v1.0.0/leveldb/db_compaction.go:762 +0xf3 fp=0xc00031d7c8 sp=0xc00031d718 pc=0x1121a93
github.com/syndtr/goleveldb/leveldb.openDB.gowrap4()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/syndtr/goleveldb@v1.0.0/leveldb/db.go:150 +0x25 fp=0xc00031d7e0 sp=0xc00031d7c8 pc=0x11172e5
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00031d7e8 sp=0xc00031d7e0 pc=0x481ec1
created by github.com/syndtr/goleveldb/leveldb.openDB in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/syndtr/goleveldb@v1.0.0/leveldb/db.go:150 +0x536

goroutine 51 gp=0xc0001ff340 m=nil [select]:
runtime.gopark(0xc0001875f0?, 0x5?, 0xe0?, 0x72?, 0xc00018744e?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc0001872b8 sp=0xc000187298 pc=0x479f4e
runtime.selectgo(0xc0001875f0, 0xc000187444, 0x0?, 0x0, 0x0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc0001873e0 sp=0xc0001872b8 pc=0x4573c5
github.com/libp2p/go-libp2p/p2p/net/connmgr.(*decayer).process(0xc0000e8460)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/connmgr/decay.go:164 +0x20c fp=0xc000187fc8 sp=0xc0001873e0 pc=0xb2d0ac
github.com/libp2p/go-libp2p/p2p/net/connmgr.NewDecayer.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/connmgr/decay.go:96 +0x25 fp=0xc000187fe0 sp=0xc000187fc8 pc=0xb2c785
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc000187fe8 sp=0xc000187fe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p/p2p/net/connmgr.NewDecayer in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/connmgr/decay.go:96 +0x245

goroutine 52 gp=0xc0001ff500 m=nil [select]:
runtime.gopark(0xc00008ef68?, 0x2?, 0x8?, 0x31?, 0xc00008ef54?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00008edf0 sp=0xc00008edd0 pc=0x479f4e
runtime.selectgo(0xc00008ef68, 0xc00008ef50, 0xc00008ef18?, 0x0, 0x6a221c?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc00008ef18 sp=0xc00008edf0 pc=0x4573c5
github.com/libp2p/go-libp2p/p2p/net/connmgr.(*BasicConnMgr).background(0xc00034b608)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/connmgr/connmgr.go:359 +0x12c fp=0xc00008efc8 sp=0xc00008ef18 pc=0xb282cc
github.com/libp2p/go-libp2p/p2p/net/connmgr.NewConnManager.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/connmgr/connmgr.go:153 +0x25 fp=0xc00008efe0 sp=0xc00008efc8 pc=0xb26e25
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00008efe8 sp=0xc00008efe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p/p2p/net/connmgr.NewConnManager in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/connmgr/connmgr.go:153 +0x376

goroutine 53 gp=0xc0001ff6c0 m=nil [select]:
runtime.gopark(0xc000088768?, 0x2?, 0xb8?, 0x35?, 0xc000088744?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc0000885e0 sp=0xc0000885c0 pc=0x479f4e
runtime.selectgo(0xc000088768, 0xc000088740, 0x6a2280?, 0x0, 0xc000159490?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc000088708 sp=0xc0000885e0 pc=0x4573c5
github.com/libp2p/go-libp2p/p2p/host/devstore/pstoremem.(*memoryAddrBook).background(0xc000416900, {0x1b34be0, 0xc00036c410})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/host/devstore/pstoremem/addr_book.go:242 +0x114 fp=0xc0000887b8 sp=0xc000088708 pc=0x9f8f14
github.com/libp2p/go-libp2p/p2p/host/devstore/pstoremem.NewAddrBook.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/host/devstore/pstoremem/addr_book.go:205 +0x28 fp=0xc0000887e0 sp=0xc0000887b8 pc=0x9f8dc8
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc0000887e8 sp=0xc0000887e0 pc=0x481ec1
created by github.com/libp2p/go-libp2p/p2p/host/devstore/pstoremem.NewAddrBook in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/host/devstore/pstoremem/addr_book.go:205 +0x1c5

goroutine 54 gp=0xc0001ff880 m=nil [select]:
runtime.gopark(0xc00008ff88?, 0x2?, 0x8?, 0x31?, 0xc00008ff54?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc0000a3df0 sp=0xc0000a3dd0 pc=0x479f4e
runtime.selectgo(0xc0000a3f88, 0xc00008ff50, 0x0?, 0x0, 0x0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc0000a3f18 sp=0xc0000a3df0 pc=0x4573c5
github.com/libp2p/go-libp2p/p2p/host/resource-manager.(*resourceManager).background(0xc000000000)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/host/resource-manager/rcmgr.go:424 +0x105 fp=0xc0000a3fc8 sp=0xc0000a3f18 pc=0xa3d865
github.com/libp2p/go-libp2p/p2p/host/resource-manager.NewResourceManager.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/host/resource-manager/rcmgr.go:212 +0x25 fp=0xc0000a3fe0 sp=0xc0000a3fc8 pc=0xa3be65
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc0000a3fe8 sp=0xc0000a3fe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p/p2p/host/resource-manager.NewResourceManager in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/host/resource-manager/rcmgr.go:212 +0xba7

goroutine 96 gp=0xc0001ffa40 m=nil [select]:
runtime.gopark(0xc000317f58?, 0x2?, 0x68?, 0x3a?, 0xc000317f3c?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc0000a4dd8 sp=0xc0000a4db8 pc=0x479f4e
runtime.selectgo(0xc0000a4f58, 0xc000317f38, 0xc000317f48?, 0x0, 0x0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc0000a4f00 sp=0xc0000a4dd8 pc=0x4573c5
github.com/libp2p/go-libp2p/p2p/host/autonat.(*autoNATService).background(0xc000681100, {0x1b34be0, 0xc0005f8910})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/host/autonat/svc.go:288 +0x148 fp=0xc0000a4fb8 sp=0xc0000a4f00 pc=0xf03788
github.com/libp2p/go-libp2p/p2p/host/autonat.(*autoNATService).Enable.gowrap2()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/host/autonat/svc.go:261 +0x28 fp=0xc0000a4fe0 sp=0xc0000a4fb8 pc=0xf033c8
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc0000a4fe8 sp=0xc0000a4fe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p/p2p/host/autonat.(*autoNATService).Enable in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/host/autonat/svc.go:261 +0x1a5

goroutine 43 gp=0xc0004d2380 m=nil [select]:
runtime.gopark(0xc0002da770?, 0x2?, 0x68?, 0x3a?, 0xc0002da764?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc0002da600 sp=0xc0002da5e0 pc=0x479f4e
runtime.selectgo(0xc0002da770, 0xc0002da760, 0x0?, 0x0, 0x0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc0002da728 sp=0xc0002da600 pc=0x4573c5
github.com/libp2p/go-libp2p/p2p/net/swarm.(*DialBackoff).background(0xc000152350, {0x1b34be0, 0xc00018e410})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/swarm_dial.go:128 +0xd0 fp=0xc0002da7b8 sp=0xc0002da728 pc=0xba8950
github.com/libp2p/go-libp2p/p2p/net/swarm.(*DialBackoff).init.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/swarm_dial.go:121 +0x28 fp=0xc0002da7e0 sp=0xc0002da7b8 pc=0xba8848
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc0002da7e8 sp=0xc0002da7e0 pc=0x481ec1
created by github.com/libp2p/go-libp2p/p2p/net/swarm.(*DialBackoff).init in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/swarm_dial.go:121 +0xb0

goroutine 75 gp=0xc0004d2a80 m=nil [select]:
runtime.gopark(0xc00018bf58?, 0x2?, 0x0?, 0x0?, 0xc00018bf54?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00018bdf0 sp=0xc00018bdd0 pc=0x479f4e
runtime.selectgo(0xc00018bf58, 0xc00018bf50, 0xc00093acf0?, 0x0, 0xc0002e85e8?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc00018bf18 sp=0xc00018bdf0 pc=0x4573c5
github.com/libp2p/go-libp2p/p2p/protocol/identify.(*ObservedAddrManager).worker(0xc0000ea000)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/protocol/identify/obsaddr.go:329 +0x10b fp=0xc00018bfc8 sp=0xc00018bf18 pc=0xf2deab
github.com/libp2p/go-libp2p/p2p/protocol/identify.NewObservedAddrManager.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/protocol/identify/obsaddr.go:191 +0x25 fp=0xc00018bfe0 sp=0xc00018bfc8 pc=0xf2c785
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00018bfe8 sp=0xc00018bfe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p/p2p/protocol/identify.NewObservedAddrManager in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/protocol/identify/obsaddr.go:191 +0x1d8

goroutine 77 gp=0xc000104a80 m=nil [select]:
runtime.gopark(0xc000319f50?, 0x4?, 0x8?, 0x31?, 0xc000319f08?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000319d80 sp=0xc000319d60 pc=0x479f4e
runtime.selectgo(0xc000319f50, 0xc000319f00, 0xc0002cba40?, 0x0, 0xcecd00?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc000319ea8 sp=0xc000319d80 pc=0x4573c5
github.com/libp2p/go-libp2p/p2p/protocol/identify.(*natEmitter).worker(0xc000390d20)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/protocol/identify/nat_emitter.go:62 +0x17f fp=0xc000319fc8 sp=0xc000319ea8 pc=0xf2b91f
github.com/libp2p/go-libp2p/p2p/protocol/identify.newNATEmitter.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/protocol/identify/nat_emitter.go:51 +0x25 fp=0xc000319fe0 sp=0xc000319fc8 pc=0xf2b6c5
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc000319fe8 sp=0xc000319fe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p/p2p/protocol/identify.newNATEmitter in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/protocol/identify/nat_emitter.go:51 +0x327

goroutine 98 gp=0xc000104e00 m=nil [chan receive]:
runtime.gopark(0x20?, 0xc00016e8c0?, 0x0?, 0x0?, 0xc000536008?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00011fbe0 sp=0xc00011fbc0 pc=0x479f4e
runtime.chanrecv(0xc00016e8c0, 0xc00011fd40, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/chan.go:639 +0x41c fp=0xc00011fc58 sp=0xc00011fbe0 pc=0x412b5c
runtime.chanrecv2(0x1b34c50?, 0xc00022d7a0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/chan.go:494 +0x12 fp=0xc00011fc80 sp=0xc00011fc58 pc=0x412732
github.com/libp2p/go-nat.DiscoverGateway({0x1b34c50?, 0xc00022d7a0?})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-nat@v0.2.0/nat.go:90 +0x7d fp=0xc00011fd60 sp=0xc00011fc80 pc=0xf69ffd
github.com/libp2p/go-libp2p/p2p/net/nat.DiscoverNAT({0x1b34c50?, 0xc00022d7a0?})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/nat/nat.go:39 +0x36 fp=0xc00011fe60 sp=0xc00011fd60 pc=0xf6d676
github.com/libp2p/go-libp2p/p2p/host/basic.init.func2({0x1b34c50?, 0xc00022d7a0?})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/host/basic/natmgr.go:46 +0x1d fp=0xc00011fe80 sp=0xc00011fe60 pc=0xf6f71d
github.com/libp2p/go-libp2p/p2p/host/basic.(*natManager).background(0xc000390fc0, {0x1b34be0, 0xc00046d9f0})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/host/basic/natmgr.go:110 +0xd8 fp=0xc00011ffb8 sp=0xc00011fe80 pc=0xf77618
github.com/libp2p/go-libp2p/p2p/host/basic.newNATManager.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/host/basic/natmgr.go:78 +0x28 fp=0xc00011ffe0 sp=0xc00011ffb8 pc=0xf77388
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00011ffe8 sp=0xc00011ffe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p/p2p/host/basic.newNATManager in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/host/basic/natmgr.go:78 +0x151

goroutine 47 gp=0xc0004d2c40 m=nil [select]:
runtime.gopark(0xc000121f28?, 0x5?, 0xe8?, 0x1d?, 0xc000121ea6?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000121d38 sp=0xc000121d18 pc=0x479f4e
runtime.selectgo(0xc000121f28, 0xc000121e9c, 0xc0002d6fa8?, 0x0, 0x1b34be0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc000121e60 sp=0xc000121d38 pc=0x4573c5
github.com/libp2p/go-nat.DiscoverNATs.func1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-nat@v0.2.0/nat.go:54 +0x210 fp=0xc000121fe0 sp=0xc000121e60 pc=0xf69cf0
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc000121fe8 sp=0xc000121fe0 pc=0x481ec1
created by github.com/libp2p/go-nat.DiscoverNATs in goroutine 98
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-nat@v0.2.0/nat.go:42 +0x86

goroutine 48 gp=0xc0004d2e00 m=nil [semacquire]:
runtime.gopark(0x474cfd?, 0x8?, 0xe0?, 0xf6?, 0x474cfd?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000504bf8 sp=0xc000504bd8 pc=0x479f4e
runtime.goparkunlock(...)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:430
runtime.semacquire1(0xc0003c4b10, 0x0, 0x1, 0x0, 0x12)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/sema.go:178 +0x225 fp=0xc000504c60 sp=0xc000504bf8 pc=0x458465
sync.runtime_Semacquire(0xc0002e9440?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/sema.go:71 +0x25 fp=0xc000504c98 sp=0xc000504c60 pc=0x47b305
sync.(*WaitGroup).Wait(0x1478620?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/sync/waitgroup.go:118 +0x48 fp=0xc000504cc0 sp=0xc000504c98 pc=0x490848
golang.org/x/sync/errgroup.(*Group).Wait(0xc0003c4b00)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/x/sync@v0.11.0/errgroup/errgroup.go:56 +0x25 fp=0xc000504ce0 sp=0xc000504cc0 pc=0xf54ac5
github.com/huin/goupnp/httpu.(*MultiClientCtx).DoWithContext(0xc0002e93e0, 0xc0003c8b40, 0x3)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/huin/goupnp@v1.3.0/httpu/multiclient.go:111 +0x1e7 fp=0xc000504d38 sp=0xc000504ce0 pc=0xf564c7
github.com/huin/goupnp/ssdp.RawSearch({0x1b34c50, 0xc00022d9d0}, {0x7f027055c0d8, 0xc0002e93e0}, {0x16151f2, 0x31}, 0x3)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/huin/goupnp@v1.3.0/ssdp/ssdp.go:107 +0x1ac fp=0xc000504dc8 sp=0xc000504d38 pc=0xf56c4c
github.com/huin/goupnp.DiscoverDevicesCtx({0x1b34c50, 0xc00022d7a0}, {0x16151f2, 0x31})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/huin/goupnp@v1.3.0/goupnp.go:91 +0x145 fp=0xc000504ee8 sp=0xc000504dc8 pc=0xf586a5
github.com/libp2p/go-nat.discoverUPNP_IG1.func1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-nat@v0.2.0/upnp.go:25 +0x85 fp=0xc000504fe0 sp=0xc000504ee8 pc=0xf6aee5
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc000504fe8 sp=0xc000504fe0 pc=0x481ec1
created by github.com/libp2p/go-nat.discoverUPNP_IG1 in goroutine 47
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-nat@v0.2.0/upnp.go:21 +0x86

goroutine 49 gp=0xc0004d2fc0 m=nil [semacquire]:
runtime.gopark(0x474cfd?, 0x8?, 0x0?, 0xa0?, 0x474cfd?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000507bf8 sp=0xc000507bd8 pc=0x479f4e
runtime.goparkunlock(...)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:430
runtime.semacquire1(0xc000530490, 0x0, 0x1, 0x0, 0x12)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/sema.go:178 +0x225 fp=0xc000507c60 sp=0xc000507bf8 pc=0x458465
sync.runtime_Semacquire(0xc0005080a8?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/sema.go:71 +0x25 fp=0xc000507c98 sp=0xc000507c60 pc=0x47b305
sync.(*WaitGroup).Wait(0x1478620?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/sync/waitgroup.go:118 +0x48 fp=0xc000507cc0 sp=0xc000507c98 pc=0x490848
golang.org/x/sync/errgroup.(*Group).Wait(0xc000530480)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/x/sync@v0.11.0/errgroup/errgroup.go:56 +0x25 fp=0xc000507ce0 sp=0xc000507cc0 pc=0xf54ac5
github.com/huin/goupnp/httpu.(*MultiClientCtx).DoWithContext(0xc000508048, 0xc00052c8c0, 0x3)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/huin/goupnp@v1.3.0/httpu/multiclient.go:111 +0x1e7 fp=0xc000507d38 sp=0xc000507ce0 pc=0xf564c7
github.com/huin/goupnp/ssdp.RawSearch({0x1b34c50, 0xc000158000}, {0x7f027055c0d8, 0xc000508048}, {0x1615223, 0x31}, 0x3)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/huin/goupnp@v1.3.0/ssdp/ssdp.go:107 +0x1ac fp=0xc000507dc8 sp=0xc000507d38 pc=0xf56c4c
github.com/huin/goupnp.DiscoverDevicesCtx({0x1b34c50, 0xc00022d7a0}, {0x1615223, 0x31})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/huin/goupnp@v1.3.0/goupnp.go:91 +0x145 fp=0xc000507ee8 sp=0xc000507dc8 pc=0xf586a5
github.com/libp2p/go-nat.discoverUPNP_IG2.func1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-nat@v0.2.0/upnp.go:82 +0x85 fp=0xc000507fe0 sp=0xc000507ee8 pc=0xf6b825
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc000507fe8 sp=0xc000507fe0 pc=0x481ec1
created by github.com/libp2p/go-nat.discoverUPNP_IG2 in goroutine 47
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-nat@v0.2.0/upnp.go:78 +0x86

goroutine 138 gp=0xc0004d3180 m=nil [select]:
runtime.gopark(0xc0002d6f68?, 0x3?, 0x8?, 0x31?, 0xc0002d6ef2?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc0002d6d88 sp=0xc0002d6d68 pc=0x479f4e
runtime.selectgo(0xc0002d6f68, 0xc0002d6eec, 0xc0004d2c40?, 0x0, 0xc0002d6ee8?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc0002d6eb0 sp=0xc0002d6d88 pc=0x4573c5
github.com/libp2p/go-libp2p/p2p/net/swarm.(*connectednessEventEmitter).runEmitter(0xc000559320)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/connectedness_event_emitter.go:93 +0x116 fp=0xc0002d6fc8 sp=0xc0002d6eb0 pc=0xb9a796
github.com/libp2p/go-libp2p/p2p/net/swarm.newConnectednessEventEmitter.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/connectedness_event_emitter.go:47 +0x25 fp=0xc0002d6fe0 sp=0xc0002d6fc8 pc=0xb9a225
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc0002d6fe8 sp=0xc0002d6fe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p/p2p/net/swarm.newConnectednessEventEmitter in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/connectedness_event_emitter.go:47 +0x185

goroutine 115 gp=0xc0004d3340 m=nil [IO wait]:
runtime.gopark(0xc0007b9220?, 0xc000120b38?, 0x0?, 0x0?, 0xc000120b28?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000120af8 sp=0xc000120ad8 pc=0x479f4e
runtime.netpollblock(0x1b26000?, 0x258ed70?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/netpoll.go:575 +0xf7 fp=0xc000120b30 sp=0xc000120af8 pc=0x43ddd7
internal/poll.runtime_pollWait(0x7f0270668568, 0x72)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/netpoll.go:351 +0x85 fp=0xc000120b50 sp=0xc000120b30 pc=0x479245
internal/poll.(*pollDesc).wait(0xc000173600?, 0x50?, 0x0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_poll_runtime.go:84 +0x27 fp=0xc000120b78 sp=0xc000120b50 pc=0x50a5c7
internal/poll.(*pollDesc).waitRead(...)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_poll_runtime.go:89
internal/poll.(*FD).RawRead(0xc000173600, 0xc0007b9220)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_unix.go:717 +0x125 fp=0xc000120bd8 sp=0xc000120b78 pc=0x510605
net.(*rawConn).Read(0xc000156cb0, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/rawconn.go:44 +0x36 fp=0xc000120c10 sp=0xc000120bd8 pc=0x632416
golang.org/x/net/internal/socket.(*Conn).recvMsg(0xc000297e80, 0xc0004039e0, 0x0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/x/net@v0.35.0/internal/socket/rawconn_msg.go:27 +0x144 fp=0xc000120c68 sp=0xc000120c10 pc=0xb32404
golang.org/x/net/internal/socket.(*Conn).RecvMsg(...)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/x/net@v0.35.0/internal/socket/socket.go:247
golang.org/x/net/ipv4.(*payloadHandler).ReadFrom(0xc0003bd780, {0xc0003dc000, 0xffff, 0xffff})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/x/net@v0.35.0/ipv4/payload_cmsg.go:31 +0x4ae fp=0xc000120d08 sp=0xc000120c68 pc=0xb3c32e
github.com/koron/go-ssdp/internal/multicast.(*Conn).ReadPackets(0xc0003c2ab0, 0x12a05f200, 0xc000120e28)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/koron/go-ssdp@v0.0.5/internal/multicast/multicast.go:183 +0x9c fp=0xc000120d58 sp=0xc000120d08 pc=0xf67ddc
github.com/koron/go-ssdp.Search({0x15cb023, 0x8}, 0x5, {0x0, 0x0}, {0x0?, 0x0?, 0x0?})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/koron/go-ssdp@v0.0.5/search.go:111 +0x36c fp=0xc000120e70 sp=0xc000120d58 pc=0xf6888c
github.com/libp2p/go-nat.discoverUPNP_GenIGDev.func1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-nat@v0.2.0/upnp.go:152 +0x99 fp=0xc000120fe0 sp=0xc000120e70 pc=0xf6c3d9
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc000120fe8 sp=0xc000120fe0 pc=0x481ec1
created by github.com/libp2p/go-nat.discoverUPNP_GenIGDev in goroutine 47
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-nat@v0.2.0/upnp.go:149 +0x8c

goroutine 139 gp=0xc0004d3500 m=nil [select]:
runtime.gopark(0xc000571f70?, 0x2?, 0x68?, 0x3a?, 0xc000571f64?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000571e00 sp=0xc000571de0 pc=0x479f4e
runtime.selectgo(0xc000571f70, 0xc000571f60, 0x0?, 0x0, 0x0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc000571f28 sp=0xc000571e00 pc=0x4573c5
github.com/libp2p/go-libp2p/p2p/net/swarm.(*DialBackoff).background(0xc000535150, {0x1b34be0, 0xc000513400})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/swarm_dial.go:128 +0xd0 fp=0xc000571fb8 sp=0xc000571f28 pc=0xba8950
github.com/libp2p/go-libp2p/p2p/net/swarm.(*DialBackoff).init.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/swarm_dial.go:121 +0x28 fp=0xc000571fe0 sp=0xc000571fb8 pc=0xba8848
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc000571fe8 sp=0xc000571fe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p/p2p/net/swarm.(*DialBackoff).init in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/swarm_dial.go:121 +0xb0

goroutine 130 gp=0xc00053a380 m=nil [semacquire]:
runtime.gopark(0x0?, 0x0?, 0xc0?, 0xa0?, 0x474cfd?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc0002db5e8 sp=0xc0002db5c8 pc=0x479f4e
runtime.goparkunlock(...)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:430
runtime.semacquire1(0xc0005304d0, 0x0, 0x1, 0x0, 0x12)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/sema.go:178 +0x225 fp=0xc0002db650 sp=0xc0002db5e8 pc=0x458465
sync.runtime_Semacquire(0xc0005080d8?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/sema.go:71 +0x25 fp=0xc0002db688 sp=0xc0002db650 pc=0x47b305
sync.(*WaitGroup).Wait(0x14eef20?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/sync/waitgroup.go:118 +0x48 fp=0xc0002db6b0 sp=0xc0002db688 pc=0x490848
golang.org/x/sync/errgroup.(*Group).Wait(0xc0005304c0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/x/sync@v0.11.0/errgroup/errgroup.go:56 +0x25 fp=0xc0002db6d0 sp=0xc0002db6b0 pc=0xf54ac5
github.com/huin/goupnp/httpu.(*MultiClientCtx).sendRequestsCtx(0xc000508048, 0xc000566d90, 0xc00052ca00, 0x3)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/huin/goupnp@v1.3.0/httpu/multiclient.go:131 +0x10e fp=0xc0002db718 sp=0xc0002db6d0 pc=0xf568ce
github.com/huin/goupnp/httpu.(*MultiClientCtx).DoWithContext.func1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/huin/goupnp@v1.3.0/httpu/multiclient.go:100 +0x59 fp=0xc0002db778 sp=0xc0002db718 pc=0xf566f9
golang.org/x/sync/errgroup.(*Group).Go.func1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/x/sync@v0.11.0/errgroup/errgroup.go:78 +0x50 fp=0xc0002db7e0 sp=0xc0002db778 pc=0xf54c30
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc0002db7e8 sp=0xc0002db7e0 pc=0x481ec1
created by golang.org/x/sync/errgroup.(*Group).Go in goroutine 49
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/x/sync@v0.11.0/errgroup/errgroup.go:75 +0x96

goroutine 131 gp=0xc00053a540 m=nil [chan receive]:
runtime.gopark(0x0?, 0x0?, 0x0?, 0x0?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000570650 sp=0xc000570630 pc=0x479f4e
runtime.chanrecv(0xc000566d90, 0xc000570740, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/chan.go:639 +0x41c fp=0xc0005706c8 sp=0xc000570650 pc=0x412b5c
runtime.chanrecv2(0x0?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/chan.go:494 +0x12 fp=0xc0005706f0 sp=0xc0005706c8 pc=0x412732
github.com/huin/goupnp/httpu.(*MultiClientCtx).DoWithContext.func2()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/huin/goupnp@v1.3.0/httpu/multiclient.go:105 +0x47 fp=0xc000570778 sp=0xc0005706f0 pc=0xf56587
golang.org/x/sync/errgroup.(*Group).Go.func1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/x/sync@v0.11.0/errgroup/errgroup.go:78 +0x50 fp=0xc0005707e0 sp=0xc000570778 pc=0xf54c30
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc0005707e8 sp=0xc0005707e0 pc=0x481ec1
created by golang.org/x/sync/errgroup.(*Group).Go in goroutine 49
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/x/sync@v0.11.0/errgroup/errgroup.go:75 +0x96

goroutine 132 gp=0xc00053a700 m=nil [IO wait]:
runtime.gopark(0x0?, 0x0?, 0x0?, 0x0?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00011b9d0 sp=0xc00011b9b0 pc=0x479f4e
runtime.netpollblock(0x0?, 0x40ff66?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/netpoll.go:575 +0xf7 fp=0xc00011ba08 sp=0xc00011b9d0 pc=0x43ddd7
internal/poll.runtime_pollWait(0x7f0270668450, 0x72)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/netpoll.go:351 +0x85 fp=0xc00011ba28 sp=0xc00011ba08 pc=0x479245
internal/poll.(*pollDesc).wait(0xc000177500?, 0xc0007a7000?, 0x0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_poll_runtime.go:84 +0x27 fp=0xc00011ba50 sp=0xc00011ba28 pc=0x50a5c7
internal/poll.(*pollDesc).waitRead(...)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_poll_runtime.go:89
internal/poll.(*FD).ReadFromInet4(0xc000177500, {0xc0007a7000, 0x800, 0x800}, 0xc00011bb98)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_unix.go:248 +0x21c fp=0xc00011bae0 sp=0xc00011ba50 pc=0x50c25c
net.(*netFD).readFromInet4(0xc000177500, {0xc0007a7000?, 0x28?, 0xc00011bba0?}, 0x41dc8b?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/fd_posix.go:66 +0x25 fp=0xc00011bb30 sp=0xc00011bae0 pc=0x61c005
net.(*UDPConn).readFrom(0xc000790090?, {0xc0007a7000?, 0x0?, 0xc000790090?}, 0xc000790090)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/udpsock_posix.go:52 +0x1b2 fp=0xc00011bc20 sp=0xc00011bb30 pc=0x63ac72
net.(*UDPConn).readFromUDP(0xc000556000, {0xc0007a7000?, 0x800?, 0x132dd40?}, 0xc00053a701?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/udpsock.go:149 +0x30 fp=0xc00011bc78 sp=0xc00011bc20 pc=0x639070
net.(*UDPConn).ReadFrom(0xc000556000, {0xc0007a7000, 0x800, 0x800})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/udpsock.go:158 +0x4a fp=0xc00011bcb0 sp=0xc00011bc78 pc=0x63920a
github.com/huin/goupnp/httpu.(*HTTPUClient).DoWithContext(0xc000508018, 0xc00052ca00, 0x3)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/huin/goupnp@v1.3.0/httpu/httpu.go:184 +0x86a fp=0xc00011bf30 sp=0xc00011bcb0 pc=0xf55aea
github.com/huin/goupnp/httpu.(*MultiClientCtx).sendRequestsCtx.func1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/huin/goupnp@v1.3.0/httpu/multiclient.go:123 +0x3c fp=0xc00011bf78 sp=0xc00011bf30 pc=0xf5695c
golang.org/x/sync/errgroup.(*Group).Go.func1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/x/sync@v0.11.0/errgroup/errgroup.go:78 +0x50 fp=0xc00011bfe0 sp=0xc00011bf78 pc=0xf54c30
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00011bfe8 sp=0xc00011bfe0 pc=0x481ec1
created by golang.org/x/sync/errgroup.(*Group).Go in goroutine 130
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/x/sync@v0.11.0/errgroup/errgroup.go:75 +0x96

goroutine 133 gp=0xc00053a8c0 m=nil [IO wait]:
runtime.gopark(0x0?, 0x0?, 0x0?, 0x0?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00011a9d0 sp=0xc00011a9b0 pc=0x479f4e
runtime.netpollblock(0x0?, 0x40ff66?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/netpoll.go:575 +0xf7 fp=0xc00011aa08 sp=0xc00011a9d0 pc=0x43ddd7
internal/poll.runtime_pollWait(0x7f0270668338, 0x72)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/netpoll.go:351 +0x85 fp=0xc00011aa28 sp=0xc00011aa08 pc=0x479245
internal/poll.(*pollDesc).wait(0xc000177580?, 0xc000495800?, 0x0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_poll_runtime.go:84 +0x27 fp=0xc00011aa50 sp=0xc00011aa28 pc=0x50a5c7
internal/poll.(*pollDesc).waitRead(...)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_poll_runtime.go:89
internal/poll.(*FD).ReadFromInet4(0xc000177580, {0xc000495800, 0x800, 0x800}, 0xc00011ab98)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_unix.go:248 +0x21c fp=0xc00011aae0 sp=0xc00011aa50 pc=0x50c25c
net.(*netFD).readFromInet4(0xc000177580, {0xc000495800?, 0x28?, 0xc00011aba0?}, 0x41dc8b?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/fd_posix.go:66 +0x25 fp=0xc00011ab30 sp=0xc00011aae0 pc=0x61c005
net.(*UDPConn).readFrom(0xc0003c39b0?, {0xc000495800?, 0x0?, 0xc0003c39b0?}, 0xc0003c39b0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/udpsock_posix.go:52 +0x1b2 fp=0xc00011ac20 sp=0xc00011ab30 pc=0x63ac72
net.(*UDPConn).readFromUDP(0xc000556008, {0xc000495800?, 0x800?, 0x132dd40?}, 0xc00053a801?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/udpsock.go:149 +0x30 fp=0xc00011ac78 sp=0xc00011ac20 pc=0x639070
net.(*UDPConn).ReadFrom(0xc000556008, {0xc000495800, 0x800, 0x800})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/udpsock.go:158 +0x4a fp=0xc00011acb0 sp=0xc00011ac78 pc=0x63920a
github.com/huin/goupnp/httpu.(*HTTPUClient).DoWithContext(0xc000508030, 0xc00052ca00, 0x3)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/huin/goupnp@v1.3.0/httpu/httpu.go:184 +0x86a fp=0xc00011af30 sp=0xc00011acb0 pc=0xf55aea
github.com/huin/goupnp/httpu.(*MultiClientCtx).sendRequestsCtx.func1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/huin/goupnp@v1.3.0/httpu/multiclient.go:123 +0x3c fp=0xc00011af78 sp=0xc00011af30 pc=0xf5695c
golang.org/x/sync/errgroup.(*Group).Go.func1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/x/sync@v0.11.0/errgroup/errgroup.go:78 +0x50 fp=0xc00011afe0 sp=0xc00011af78 pc=0xf54c30
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00011afe8 sp=0xc00011afe0 pc=0x481ec1
created by golang.org/x/sync/errgroup.(*Group).Go in goroutine 130
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/x/sync@v0.11.0/errgroup/errgroup.go:75 +0x96

goroutine 134 gp=0xc00053aa80 m=nil [select]:
runtime.gopark(0xc0005717b0?, 0x2?, 0x18?, 0x3f?, 0xc00057179c?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000571640 sp=0xc000571620 pc=0x479f4e
runtime.selectgo(0xc0005717b0, 0xc000571798, 0x0?, 0x0, 0x100000000000000?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc000571768 sp=0xc000571640 pc=0x4573c5
github.com/huin/goupnp/httpu.(*HTTPUClient).DoWithContext.func1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/huin/goupnp@v1.3.0/httpu/httpu.go:160 +0x6b fp=0xc0005717e0 sp=0xc000571768 pc=0xf561cb
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc0005717e8 sp=0xc0005717e0 pc=0x481ec1
created by github.com/huin/goupnp/httpu.(*HTTPUClient).DoWithContext in goroutine 133
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/huin/goupnp@v1.3.0/httpu/httpu.go:159 +0x3a8

goroutine 135 gp=0xc00053ac40 m=nil [select]:
runtime.gopark(0xc000570fb0?, 0x2?, 0x0?, 0x0?, 0xc000570f9c?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000570e40 sp=0xc000570e20 pc=0x479f4e
runtime.selectgo(0xc000570fb0, 0xc000570f98, 0x0?, 0x0, 0x100000000000000?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc000570f68 sp=0xc000570e40 pc=0x4573c5
github.com/huin/goupnp/httpu.(*HTTPUClient).DoWithContext.func1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/huin/goupnp@v1.3.0/httpu/httpu.go:160 +0x6b fp=0xc000570fe0 sp=0xc000570f68 pc=0xf561cb
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc000570fe8 sp=0xc000570fe0 pc=0x481ec1
created by github.com/huin/goupnp/httpu.(*HTTPUClient).DoWithContext in goroutine 132
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/huin/goupnp@v1.3.0/httpu/httpu.go:159 +0x3a8

goroutine 117 gp=0xc0004d36c0 m=nil [semacquire]:
runtime.gopark(0xa0?, 0xc00022d788?, 0xa0?, 0xf7?, 0x474cfd?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc0002d65e8 sp=0xc0002d65c8 pc=0x479f4e
runtime.goparkunlock(...)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:430
runtime.semacquire1(0xc0003c4b50, 0x0, 0x1, 0x0, 0x12)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/sema.go:178 +0x225 fp=0xc0002d6650 sp=0xc0002d65e8 pc=0x458465
sync.runtime_Semacquire(0xc0002e9470?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/sema.go:71 +0x25 fp=0xc0002d6688 sp=0xc0002d6650 pc=0x47b305
sync.(*WaitGroup).Wait(0x14eef20?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/sync/waitgroup.go:118 +0x48 fp=0xc0002d66b0 sp=0xc0002d6688 pc=0x490848
golang.org/x/sync/errgroup.(*Group).Wait(0xc0003c4b40)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/x/sync@v0.11.0/errgroup/errgroup.go:56 +0x25 fp=0xc0002d66d0 sp=0xc0002d66b0 pc=0xf54ac5
github.com/huin/goupnp/httpu.(*MultiClientCtx).sendRequestsCtx(0xc0002e93e0, 0xc00016eb60, 0xc0003c8c80, 0x3)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/huin/goupnp@v1.3.0/httpu/multiclient.go:131 +0x10e fp=0xc0002d6718 sp=0xc0002d66d0 pc=0xf568ce
github.com/huin/goupnp/httpu.(*MultiClientCtx).DoWithContext.func1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/huin/goupnp@v1.3.0/httpu/multiclient.go:100 +0x59 fp=0xc0002d6778 sp=0xc0002d6718 pc=0xf566f9
golang.org/x/sync/errgroup.(*Group).Go.func1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/x/sync@v0.11.0/errgroup/errgroup.go:78 +0x50 fp=0xc0002d67e0 sp=0xc0002d6778 pc=0xf54c30
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc0002d67e8 sp=0xc0002d67e0 pc=0x481ec1
created by golang.org/x/sync/errgroup.(*Group).Go in goroutine 48
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/x/sync@v0.11.0/errgroup/errgroup.go:75 +0x96

goroutine 118 gp=0xc0004d3880 m=nil [chan receive]:
runtime.gopark(0xc0002e9398?, 0x0?, 0xd0?, 0xbf?, 0xf6c3d9?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc0002dbe50 sp=0xc0002dbe30 pc=0x479f4e
runtime.chanrecv(0xc00016eb60, 0xc0002dbf40, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/chan.go:639 +0x41c fp=0xc0002dbec8 sp=0xc0002dbe50 pc=0x412b5c
runtime.chanrecv2(0x132dd40?, 0x1?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/chan.go:494 +0x12 fp=0xc0002dbef0 sp=0xc0002dbec8 pc=0x412732
github.com/huin/goupnp/httpu.(*MultiClientCtx).DoWithContext.func2()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/huin/goupnp@v1.3.0/httpu/multiclient.go:105 +0x47 fp=0xc0002dbf78 sp=0xc0002dbef0 pc=0xf56587
golang.org/x/sync/errgroup.(*Group).Go.func1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/x/sync@v0.11.0/errgroup/errgroup.go:78 +0x50 fp=0xc0002dbfe0 sp=0xc0002dbf78 pc=0xf54c30
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc0002dbfe8 sp=0xc0002dbfe0 pc=0x481ec1
created by golang.org/x/sync/errgroup.(*Group).Go in goroutine 48
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/x/sync@v0.11.0/errgroup/errgroup.go:75 +0x96

goroutine 119 gp=0xc0004d3a40 m=nil [IO wait]:
runtime.gopark(0x0?, 0x0?, 0x0?, 0x0?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc0005069d0 sp=0xc0005069b0 pc=0x479f4e
runtime.netpollblock(0x0?, 0x40ff66?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/netpoll.go:575 +0xf7 fp=0xc000506a08 sp=0xc0005069d0 pc=0x43ddd7
internal/poll.runtime_pollWait(0x7f0270668220, 0x72)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/netpoll.go:351 +0x85 fp=0xc000506a28 sp=0xc000506a08 pc=0x479245
internal/poll.(*pollDesc).wait(0xc00017cc00?, 0xc0007a6000?, 0x0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_poll_runtime.go:84 +0x27 fp=0xc000506a50 sp=0xc000506a28 pc=0x50a5c7
internal/poll.(*pollDesc).waitRead(...)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_poll_runtime.go:89
internal/poll.(*FD).ReadFromInet4(0xc00017cc00, {0xc0007a6000, 0x800, 0x800}, 0xc000506b98)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_unix.go:248 +0x21c fp=0xc000506ae0 sp=0xc000506a50 pc=0x50c25c
net.(*netFD).readFromInet4(0xc00017cc00, {0xc0007a6000?, 0x28?, 0xc000506ba0?}, 0x41dc8b?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/fd_posix.go:66 +0x25 fp=0xc000506b30 sp=0xc000506ae0 pc=0x61c005
net.(*UDPConn).readFrom(0xc000790030?, {0xc0007a6000?, 0x0?, 0xc000790030?}, 0xc000790030)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/udpsock_posix.go:52 +0x1b2 fp=0xc000506c20 sp=0xc000506b30 pc=0x63ac72
net.(*UDPConn).readFromUDP(0xc000156cc8, {0xc0007a6000?, 0x800?, 0x132dd40?}, 0xc0004d3a01?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/udpsock.go:149 +0x30 fp=0xc000506c78 sp=0xc000506c20 pc=0x639070
net.(*UDPConn).ReadFrom(0xc000156cc8, {0xc0007a6000, 0x800, 0x800})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/udpsock.go:158 +0x4a fp=0xc000506cb0 sp=0xc000506c78 pc=0x63920a
github.com/huin/goupnp/httpu.(*HTTPUClient).DoWithContext(0xc0002e93b0, 0xc0003c8c80, 0x3)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/huin/goupnp@v1.3.0/httpu/httpu.go:184 +0x86a fp=0xc000506f30 sp=0xc000506cb0 pc=0xf55aea
github.com/huin/goupnp/httpu.(*MultiClientCtx).sendRequestsCtx.func1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/huin/goupnp@v1.3.0/httpu/multiclient.go:123 +0x3c fp=0xc000506f78 sp=0xc000506f30 pc=0xf5695c
golang.org/x/sync/errgroup.(*Group).Go.func1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/x/sync@v0.11.0/errgroup/errgroup.go:78 +0x50 fp=0xc000506fe0 sp=0xc000506f78 pc=0xf54c30
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc000506fe8 sp=0xc000506fe0 pc=0x481ec1
created by golang.org/x/sync/errgroup.(*Group).Go in goroutine 117
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/x/sync@v0.11.0/errgroup/errgroup.go:75 +0x96

goroutine 120 gp=0xc0004d3c00 m=nil [IO wait]:
runtime.gopark(0x0?, 0x0?, 0x0?, 0x0?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc0005059d0 sp=0xc0005059b0 pc=0x479f4e
runtime.netpollblock(0x0?, 0x40ff66?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/netpoll.go:575 +0xf7 fp=0xc000505a08 sp=0xc0005059d0 pc=0x43ddd7
internal/poll.runtime_pollWait(0x7f0270668108, 0x72)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/netpoll.go:351 +0x85 fp=0xc000505a28 sp=0xc000505a08 pc=0x479245
internal/poll.(*pollDesc).wait(0xc00017cc80?, 0xc0007a6800?, 0x0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_poll_runtime.go:84 +0x27 fp=0xc000505a50 sp=0xc000505a28 pc=0x50a5c7
internal/poll.(*pollDesc).waitRead(...)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_poll_runtime.go:89
internal/poll.(*FD).ReadFromInet4(0xc00017cc80, {0xc0007a6800, 0x800, 0x800}, 0xc000505b98)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_unix.go:248 +0x21c fp=0xc000505ae0 sp=0xc000505a50 pc=0x50c25c
net.(*netFD).readFromInet4(0xc00017cc80, {0xc0007a6800?, 0x28?, 0xc000505ba0?}, 0x41dc8b?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/fd_posix.go:66 +0x25 fp=0xc000505b30 sp=0xc000505ae0 pc=0x61c005
net.(*UDPConn).readFrom(0xc000790060?, {0xc0007a6800?, 0x0?, 0xc000790060?}, 0xc000790060)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/udpsock_posix.go:52 +0x1b2 fp=0xc000505c20 sp=0xc000505b30 pc=0x63ac72
net.(*UDPConn).readFromUDP(0xc000156cd0, {0xc0007a6800?, 0x800?, 0x132dd40?}, 0xc0004d3c01?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/udpsock.go:149 +0x30 fp=0xc000505c78 sp=0xc000505c20 pc=0x639070
net.(*UDPConn).ReadFrom(0xc000156cd0, {0xc0007a6800, 0x800, 0x800})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/udpsock.go:158 +0x4a fp=0xc000505cb0 sp=0xc000505c78 pc=0x63920a
github.com/huin/goupnp/httpu.(*HTTPUClient).DoWithContext(0xc0002e93c8, 0xc0003c8c80, 0x3)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/huin/goupnp@v1.3.0/httpu/httpu.go:184 +0x86a fp=0xc000505f30 sp=0xc000505cb0 pc=0xf55aea
github.com/huin/goupnp/httpu.(*MultiClientCtx).sendRequestsCtx.func1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/huin/goupnp@v1.3.0/httpu/multiclient.go:123 +0x3c fp=0xc000505f78 sp=0xc000505f30 pc=0xf5695c
golang.org/x/sync/errgroup.(*Group).Go.func1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/x/sync@v0.11.0/errgroup/errgroup.go:78 +0x50 fp=0xc000505fe0 sp=0xc000505f78 pc=0xf54c30
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc000505fe8 sp=0xc000505fe0 pc=0x481ec1
created by golang.org/x/sync/errgroup.(*Group).Go in goroutine 117
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/x/sync@v0.11.0/errgroup/errgroup.go:75 +0x96

goroutine 121 gp=0xc0004d3dc0 m=nil [select]:
runtime.gopark(0xc000316fb0?, 0x2?, 0xb8?, 0x35?, 0xc000316f9c?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000316e40 sp=0xc000316e20 pc=0x479f4e
runtime.selectgo(0xc000316fb0, 0xc000316f98, 0x0?, 0x0, 0x100000000000000?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc000316f68 sp=0xc000316e40 pc=0x4573c5
github.com/huin/goupnp/httpu.(*HTTPUClient).DoWithContext.func1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/huin/goupnp@v1.3.0/httpu/httpu.go:160 +0x6b fp=0xc000316fe0 sp=0xc000316f68 pc=0xf561cb
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc000316fe8 sp=0xc000316fe0 pc=0x481ec1
created by github.com/huin/goupnp/httpu.(*HTTPUClient).DoWithContext in goroutine 120
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/huin/goupnp@v1.3.0/httpu/httpu.go:159 +0x3a8

goroutine 122 gp=0xc000286c40 m=nil [select]:
runtime.gopark(0xc0003167b0?, 0x2?, 0x0?, 0x0?, 0xc00031679c?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000316640 sp=0xc000316620 pc=0x479f4e
runtime.selectgo(0xc0003167b0, 0xc000316798, 0x0?, 0x0, 0x100000000000000?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc000316768 sp=0xc000316640 pc=0x4573c5
github.com/huin/goupnp/httpu.(*HTTPUClient).DoWithContext.func1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/huin/goupnp@v1.3.0/httpu/httpu.go:160 +0x6b fp=0xc0003167e0 sp=0xc000316768 pc=0xf561cb
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc0003167e8 sp=0xc0003167e0 pc=0x481ec1
created by github.com/huin/goupnp/httpu.(*HTTPUClient).DoWithContext in goroutine 119
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/huin/goupnp@v1.3.0/httpu/httpu.go:159 +0x3a8

goroutine 99 gp=0xc000104fc0 m=nil [select]:
runtime.gopark(0xc0002d75d8?, 0x2?, 0x8?, 0x31?, 0xc0002d757c?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000277c00 sp=0xc000277be0 pc=0x479f4e
runtime.selectgo(0xc000277dd8, 0xc0002d7578, 0x0?, 0x0, 0x474cfd?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc000277d28 sp=0xc000277c00 pc=0x4573c5
github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/client.(*Listener).Accept(0xc000477860)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/protocol/circuitv2/client/listen.go:21 +0x131 fp=0xc000277e18 sp=0xc000277d28 pc=0xf7e851
github.com/libp2p/go-libp2p/p2p/net/upgrader.(*listener).handleIncoming(0xc000391030)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/upgrader/listener.go:75 +0xe3 fp=0xc000277fc8 sp=0xc000277e18 pc=0xbcf523
github.com/libp2p/go-libp2p/p2p/net/upgrader.(*upgrader).UpgradeListener.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/upgrader/upgrader.go:119 +0x25 fp=0xc000277fe0 sp=0xc000277fc8 pc=0xbd0f45
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc000277fe8 sp=0xc000277fe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p/p2p/net/upgrader.(*upgrader).UpgradeListener in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/upgrader/upgrader.go:119 +0x1c5

goroutine 100 gp=0xc000105180 m=nil [chan receive]:
runtime.gopark(0x0?, 0x0?, 0x0?, 0x0?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc0002d7d60 sp=0xc0002d7d40 pc=0x479f4e
runtime.chanrecv(0xc000058620, 0xc0002d7e38, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/chan.go:639 +0x41c fp=0xc0002d7dd8 sp=0xc0002d7d60 pc=0x412b5c
runtime.chanrecv2(0x4a401cf19bd2fe4c?, 0x8c0992613d106906?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/chan.go:494 +0x12 fp=0xc0002d7e00 sp=0xc0002d7dd8 pc=0x412732
github.com/libp2p/go-libp2p/p2p/net/upgrader.(*listener).Accept(0xc000391030)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/upgrader/listener.go:173 +0x3a fp=0xc0002d7e58 sp=0xc0002d7e00 pc=0xbd031a
github.com/libp2p/go-libp2p/p2p/net/swarm.(*Swarm).AddListenAddr.func2()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/swarm_listen.go:161 +0xee fp=0xc0002d7fe0 sp=0xc0002d7e58 pc=0xbadf0e
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc0002d7fe8 sp=0xc0002d7fe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p/p2p/net/swarm.(*Swarm).AddListenAddr in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/swarm_listen.go:139 +0x215

goroutine 137 gp=0xc000105340 m=nil [select]:
runtime.gopark(0xc00056ef68?, 0x2?, 0x68?, 0x3a?, 0xc00056ef44?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00056ede0 sp=0xc00056edc0 pc=0x479f4e
runtime.selectgo(0xc00056ef68, 0xc00056ef40, 0x3?, 0x0, 0xc00056ef50?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc00056ef08 sp=0xc00056ede0 pc=0x4573c5
github.com/libp2p/go-libp2p/p2p/host/devstore/pstoremem.(*memoryAddrBook).background(0xc000177780, {0x1b34be0, 0xc0005124b0})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/host/devstore/pstoremem/addr_book.go:242 +0x114 fp=0xc00056efb8 sp=0xc00056ef08 pc=0x9f8f14
github.com/libp2p/go-libp2p/p2p/host/devstore/pstoremem.NewAddrBook.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/host/devstore/pstoremem/addr_book.go:205 +0x28 fp=0xc00056efe0 sp=0xc00056efb8 pc=0x9f8dc8
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00056efe8 sp=0xc00056efe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p/p2p/host/devstore/pstoremem.NewAddrBook in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/host/devstore/pstoremem/addr_book.go:205 +0x1c5

goroutine 113 gp=0xc000105500 m=nil [IO wait]:
runtime.gopark(0x0?, 0x0?, 0x0?, 0x0?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000501b10 sp=0xc000501af0 pc=0x479f4e
runtime.netpollblock(0x0?, 0x40ff66?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/netpoll.go:575 +0xf7 fp=0xc000501b48 sp=0xc000501b10 pc=0x43ddd7
internal/poll.runtime_pollWait(0x7f0270667ff0, 0x72)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/netpoll.go:351 +0x85 fp=0xc000501b68 sp=0xc000501b48 pc=0x479245
internal/poll.(*pollDesc).wait(0xc000168180?, 0xfb3801?, 0x0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_poll_runtime.go:84 +0x27 fp=0xc000501b90 sp=0xc000501b68 pc=0x50a5c7
internal/poll.(*pollDesc).waitRead(...)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_poll_runtime.go:89
internal/poll.(*FD).Accept(0xc000168180)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_unix.go:620 +0x295 fp=0xc000501c38 sp=0xc000501b90 pc=0x50f995
net.(*netFD).accept(0xc000168180)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/fd_unix.go:172 +0x29 fp=0xc000501cf0 sp=0xc000501c38 pc=0x61ddc9
net.(*TCPListener).accept(0xc0004e62c0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/tcpsock_posix.go:159 +0x1e fp=0xc000501d40 sp=0xc000501cf0 pc=0x637d3e
net.(*TCPListener).Accept(0xc0004e62c0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/tcpsock.go:372 +0x30 fp=0xc000501d70 sp=0xc000501d40 pc=0x636bf0
github.com/multiformats/go-multiaddr/net.(*maListener).Accept(0x101?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/multiformats/go-multiaddr@v0.14.0/net/net.go:243 +0x24 fp=0xc000501e00 sp=0xc000501d70 pc=0x9ecd44
github.com/libp2p/go-libp2p/p2p/net/reuseport.(*listener).Accept(0x5366346d6c446b6d?)
	<autogenerated>:1 +0x24 fp=0xc000501e18 sp=0xc000501e00 pc=0xcff924
github.com/libp2p/go-libp2p/p2p/net/upgrader.(*listener).handleIncoming(0xc0003917a0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/upgrader/listener.go:75 +0xe3 fp=0xc000501fc8 sp=0xc000501e18 pc=0xbcf523
github.com/libp2p/go-libp2p/p2p/net/upgrader.(*upgrader).UpgradeListener.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/upgrader/upgrader.go:119 +0x25 fp=0xc000501fe0 sp=0xc000501fc8 pc=0xbd0f45
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc000501fe8 sp=0xc000501fe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p/p2p/net/upgrader.(*upgrader).UpgradeListener in goroutine 112
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/upgrader/upgrader.go:119 +0x1c5

goroutine 146 gp=0xc0001056c0 m=nil [chan receive]:
runtime.gopark(0x5a783334396d6e49?, 0x3857420a63716b59?, 0x39?, 0x7a?, 0x4c6863765269785a?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc0002d8d60 sp=0xc0002d8d40 pc=0x479f4e
runtime.chanrecv(0xc0000588c0, 0xc0002d8e38, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/chan.go:639 +0x41c fp=0xc0002d8dd8 sp=0xc0002d8d60 pc=0x412b5c
runtime.chanrecv2(0x424f305256445951?, 0x47676e4a46455942?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/chan.go:494 +0x12 fp=0xc0002d8e00 sp=0xc0002d8dd8 pc=0x412732
github.com/libp2p/go-libp2p/p2p/net/upgrader.(*listener).Accept(0xc0003917a0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/upgrader/listener.go:173 +0x3a fp=0xc0002d8e58 sp=0xc0002d8e00 pc=0xbd031a
github.com/libp2p/go-libp2p/p2p/net/swarm.(*Swarm).AddListenAddr.func2()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/swarm_listen.go:161 +0xee fp=0xc0002d8fe0 sp=0xc0002d8e58 pc=0xbadf0e
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc0002d8fe8 sp=0xc0002d8fe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p/p2p/net/swarm.(*Swarm).AddListenAddr in goroutine 112
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/swarm_listen.go:139 +0x215

goroutine 147 gp=0xc000105880 m=nil [IO wait]:
runtime.gopark(0x0?, 0x0?, 0x0?, 0x0?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000502b10 sp=0xc000502af0 pc=0x479f4e
runtime.netpollblock(0x15ca983?, 0x40ff66?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/netpoll.go:575 +0xf7 fp=0xc000502b48 sp=0xc000502b10 pc=0x43ddd7
internal/poll.runtime_pollWait(0x7f0270667ed8, 0x72)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/netpoll.go:351 +0x85 fp=0xc000502b68 sp=0xc000502b48 pc=0x479245
internal/poll.(*pollDesc).wait(0xc000168280?, 0x0?, 0x0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_poll_runtime.go:84 +0x27 fp=0xc000502b90 sp=0xc000502b68 pc=0x50a5c7
internal/poll.(*pollDesc).waitRead(...)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_poll_runtime.go:89
internal/poll.(*FD).Accept(0xc000168280)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_unix.go:620 +0x295 fp=0xc000502c38 sp=0xc000502b90 pc=0x50f995
net.(*netFD).accept(0xc000168280)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/fd_unix.go:172 +0x29 fp=0xc000502cf0 sp=0xc000502c38 pc=0x61ddc9
net.(*TCPListener).accept(0xc0004e6340)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/tcpsock_posix.go:159 +0x1e fp=0xc000502d40 sp=0xc000502cf0 pc=0x637d3e
net.(*TCPListener).Accept(0xc0004e6340)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/tcpsock.go:372 +0x30 fp=0xc000502d70 sp=0xc000502d40 pc=0x636bf0
github.com/multiformats/go-multiaddr/net.(*maListener).Accept(0xbcf88f?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/multiformats/go-multiaddr@v0.14.0/net/net.go:243 +0x24 fp=0xc000502e00 sp=0xc000502d70 pc=0x9ecd44
github.com/libp2p/go-libp2p/p2p/net/reuseport.(*listener).Accept(0x0?)
	<autogenerated>:1 +0x24 fp=0xc000502e18 sp=0xc000502e00 pc=0xcff924
github.com/libp2p/go-libp2p/p2p/net/upgrader.(*listener).handleIncoming(0xc000391880)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/upgrader/listener.go:75 +0xe3 fp=0xc000502fc8 sp=0xc000502e18 pc=0xbcf523
github.com/libp2p/go-libp2p/p2p/net/upgrader.(*upgrader).UpgradeListener.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/upgrader/upgrader.go:119 +0x25 fp=0xc000502fe0 sp=0xc000502fc8 pc=0xbd0f45
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc000502fe8 sp=0xc000502fe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p/p2p/net/upgrader.(*upgrader).UpgradeListener in goroutine 112
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/upgrader/upgrader.go:119 +0x1c5

goroutine 148 gp=0xc000105a40 m=nil [chan receive]:
runtime.gopark(0x15c746b?, 0x3?, 0x1?, 0x2b?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000880d60 sp=0xc000880d40 pc=0x479f4e
runtime.chanrecv(0xc000058930, 0xc000880e38, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/chan.go:639 +0x41c fp=0xc000880dd8 sp=0xc000880d60 pc=0x412b5c
runtime.chanrecv2(0xc0003c25d0?, 0x47fef2?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/chan.go:494 +0x12 fp=0xc000880e00 sp=0xc000880dd8 pc=0x412732
github.com/libp2p/go-libp2p/p2p/net/upgrader.(*listener).Accept(0xc000391880)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/upgrader/listener.go:173 +0x3a fp=0xc000880e58 sp=0xc000880e00 pc=0xbd031a
github.com/libp2p/go-libp2p/p2p/net/swarm.(*Swarm).AddListenAddr.func2()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/swarm_listen.go:161 +0xee fp=0xc000880fe0 sp=0xc000880e58 pc=0xbadf0e
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc000880fe8 sp=0xc000880fe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p/p2p/net/swarm.(*Swarm).AddListenAddr in goroutine 112
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/swarm_listen.go:139 +0x215

goroutine 149 gp=0xc000105c00 m=nil [select]:
runtime.gopark(0xc000500f38?, 0x2?, 0x8?, 0x31?, 0xc000500eac?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000500d28 sp=0xc000500d08 pc=0x479f4e
runtime.selectgo(0xc000500f38, 0xc000500ea8, 0xc000014bb8?, 0x0, 0x1?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc000500e50 sp=0xc000500d28 pc=0x4573c5
github.com/libp2p/go-libp2p/p2p/host/autorelay.(*AutoRelay).background(0xc000391570)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/host/autorelay/autorelay.go:89 +0x234 fp=0xc000500fa8 sp=0xc000500e50 pc=0xf83f74
github.com/libp2p/go-libp2p/p2p/host/autorelay.(*AutoRelay).Start.func1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/host/autorelay/autorelay.go:76 +0x45 fp=0xc000500fe0 sp=0xc000500fa8 pc=0xf83ca5
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc000500fe8 sp=0xc000500fe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p/p2p/host/autorelay.(*AutoRelay).Start in goroutine 112
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/host/autorelay/autorelay.go:74 +0x5b

goroutine 151 gp=0xc0004f6000 m=nil [select]:
runtime.gopark(0xc000273d08?, 0x3?, 0x8?, 0x31?, 0xc000273cb2?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000273b50 sp=0xc000273b30 pc=0x479f4e
runtime.selectgo(0xc000273d08, 0xc000273cac, 0xc0004a8030?, 0x0, 0x0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc000273c78 sp=0xc000273b50 pc=0x4573c5
github.com/libp2p/go-libp2p/p2p/host/pstoremanager.(*PeerstoreManager).background(0xc000403ce0, {0x1b34be0, 0xc0004e8320}, {0x1b31c60, 0xc000477aa0})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/host/pstoremanager/pstoremanager.go:98 +0x265 fp=0xc000273fa8 sp=0xc000273c78 pc=0xf047c5
github.com/libp2p/go-libp2p/p2p/host/pstoremanager.(*PeerstoreManager).Start.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/host/pstoremanager/pstoremanager.go:80 +0x30 fp=0xc000273fe0 sp=0xc000273fa8 pc=0xf04490
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc000273fe8 sp=0xc000273fe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p/p2p/host/pstoremanager.(*PeerstoreManager).Start in goroutine 112
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/host/pstoremanager/pstoremanager.go:80 +0x213

goroutine 152 gp=0xc0004f61c0 m=nil [select]:
runtime.gopark(0xc00022ff30?, 0x2?, 0xc0?, 0x61?, 0xc00022febc?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00022fd38 sp=0xc00022fd18 pc=0x479f4e
runtime.selectgo(0xc00022ff30, 0xc00022feb8, 0xc000777710?, 0x0, 0x2?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc00022fe60 sp=0xc00022fd38 pc=0x4573c5
github.com/libp2p/go-libp2p/p2p/protocol/identify.(*idService).loop(0xc00040c000, {0x1b34be0, 0xc00015a3c0})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/protocol/identify/id.go:291 +0x409 fp=0xc00022ffb8 sp=0xc00022fe60 pc=0xf22e89
github.com/libp2p/go-libp2p/p2p/protocol/identify.(*idService).Start.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/protocol/identify/id.go:254 +0x28 fp=0xc00022ffe0 sp=0xc00022ffb8 pc=0xf22a48
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00022ffe8 sp=0xc00022ffe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p/p2p/protocol/identify.(*idService).Start in goroutine 112
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/protocol/identify/id.go:254 +0x190

goroutine 153 gp=0xc0004f6380 m=nil [select]:
runtime.gopark(0xc0005b5f48?, 0x3?, 0x0?, 0x0?, 0xc0005b5efa?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00079fd98 sp=0xc00079fd78 pc=0x479f4e
runtime.selectgo(0xc00079ff48, 0xc0005b5ef4, 0x4?, 0x0, 0x0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc00079fec0 sp=0xc00079fd98 pc=0x4573c5
github.com/libp2p/go-libp2p/p2p/host/basic.(*BasicHost).background(0xc000394000)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/host/basic/basic_host.go:612 +0x1db fp=0xc00079ffc8 sp=0xc00079fec0 pc=0xf7367b
github.com/libp2p/go-libp2p/p2p/host/basic.(*BasicHost).Start.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/host/basic/basic_host.go:439 +0x25 fp=0xc00079ffe0 sp=0xc00079ffc8 pc=0xf72265
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00079ffe8 sp=0xc00079ffe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p/p2p/host/basic.(*BasicHost).Start in goroutine 112
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/host/basic/basic_host.go:439 +0x105

goroutine 94 gp=0xc000278000 m=nil [select]:
runtime.gopark(0xc0007a3f98?, 0x2?, 0xe0?, 0xd5?, 0xc0007a3f74?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc0007a3e10 sp=0xc0007a3df0 pc=0x479f4e
runtime.selectgo(0xc0007a3f98, 0xc0007a3f70, 0xc00015a3c0?, 0x0, 0x0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc0007a3f38 sp=0xc0007a3e10 pc=0x4573c5
github.com/libp2p/go-libp2p/p2p/protocol/identify.(*idService).loop.func1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/protocol/identify/id.go:281 +0xdf fp=0xc0007a3fe0 sp=0xc0007a3f38 pc=0xf2307f
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc0007a3fe8 sp=0xc0007a3fe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p/p2p/protocol/identify.(*idService).loop in goroutine 152
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/protocol/identify/id.go:277 +0x374

goroutine 140 gp=0xc00053afc0 m=nil [select]:
runtime.gopark(0xc0002d46a8?, 0x2?, 0x70?, 0x0?, 0xc0002d463c?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc0000a0cd8 sp=0xc0000a0cb8 pc=0x479f4e
runtime.selectgo(0xc0000a0ea8, 0xc0002d4638, 0x3727b8b382c6dbc7?, 0x0, 0x7c1079ca64ce2e44?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc0000a0e00 sp=0xc0000a0cd8 pc=0x4573c5
github.com/libp2p/go-libp2p/p2p/transport/quicreuse.(*reuse).gc(0xc000760050)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/transport/quicreuse/reuse.go:219 +0xfe fp=0xc0000a0fc8 sp=0xc0000a0e00 pc=0xcec89e
github.com/libp2p/go-libp2p/p2p/transport/quicreuse.newReuse.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/transport/quicreuse/reuse.go:194 +0x25 fp=0xc0000a0fe0 sp=0xc0000a0fc8 pc=0xcec765
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc0000a0fe8 sp=0xc0000a0fe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p/p2p/transport/quicreuse.newReuse in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/transport/quicreuse/reuse.go:194 +0x145

goroutine 141 gp=0xc00053b180 m=nil [select]:
runtime.gopark(0xc0002d4ea8?, 0x2?, 0x70?, 0x0?, 0xc0002d4e3c?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000274cd8 sp=0xc000274cb8 pc=0x479f4e
runtime.selectgo(0xc000274ea8, 0xc0002d4e38, 0x5e2855968772dc86?, 0x0, 0x8c3a639c03349743?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc000274e00 sp=0xc000274cd8 pc=0x4573c5
github.com/libp2p/go-libp2p/p2p/transport/quicreuse.(*reuse).gc(0xc0007600a0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/transport/quicreuse/reuse.go:219 +0xfe fp=0xc000274fc8 sp=0xc000274e00 pc=0xcec89e
github.com/libp2p/go-libp2p/p2p/transport/quicreuse.newReuse.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/transport/quicreuse/reuse.go:194 +0x25 fp=0xc000274fe0 sp=0xc000274fc8 pc=0xcec765
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc000274fe8 sp=0xc000274fe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p/p2p/transport/quicreuse.newReuse in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/transport/quicreuse/reuse.go:194 +0x145

goroutine 95 gp=0xc00053b6c0 m=nil [chan receive]:
runtime.gopark(0x0?, 0x259de7f25675941c?, 0xd0?, 0x20?, 0xfeaa27254b5dbca2?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc0002d5710 sp=0xc0002d56f0 pc=0x479f4e
runtime.chanrecv(0xc000058a80, 0x0, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/chan.go:639 +0x41c fp=0xc0002d5788 sp=0xc0002d5710 pc=0x412b5c
runtime.chanrecv1(0x44d9bb?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/chan.go:489 +0x12 fp=0xc0002d57b0 sp=0xc0002d5788 pc=0x412712
github.com/libp2p/go-libp2p/config.(*Config).addAutoNAT.func6()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/config/config.go:686 +0x39 fp=0xc0002d57e0 sp=0xc0002d57b0 pc=0xfd7dd9
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc0002d57e8 sp=0xc0002d57e0 pc=0x481ec1
created by github.com/libp2p/go-libp2p/config.(*Config).addAutoNAT in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/config/config.go:685 +0xc56

goroutine 163 gp=0xc000278380 m=nil [select]:
runtime.gopark(0xc00080be60?, 0x7?, 0x3d?, 0x61?, 0xc00080bd62?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00080bbd8 sp=0xc00080bbb8 pc=0x479f4e
runtime.selectgo(0xc00080be60, 0xc00080bd54, 0x0?, 0x0, 0x70000c0000ea628?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc00080bd00 sp=0xc00080bbd8 pc=0x4573c5
github.com/libp2p/go-libp2p/p2p/host/autonat.(*AmbientAutoNAT).background(0xc00020de10)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/host/autonat/autonat.go:187 +0x33c fp=0xc00080bfc8 sp=0xc00080bd00 pc=0xefdb1c
github.com/libp2p/go-libp2p/p2p/host/autonat.New.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/host/autonat/autonat.go:138 +0x25 fp=0xc00080bfe0 sp=0xc00080bfc8 pc=0xefd545
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00080bfe8 sp=0xc00080bfe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p/p2p/host/autonat.New in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/host/autonat/autonat.go:138 +0x6e5

goroutine 164 gp=0xc000278540 m=nil [select]:
runtime.gopark(0xc0003f3dc8?, 0x5?, 0xb8?, 0x35?, 0xc0003f3c96?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc0003f3af0 sp=0xc0003f3ad0 pc=0x479f4e
runtime.selectgo(0xc0003f3dc8, 0xc0003f3c8c, 0x0?, 0x0, 0x0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc0003f3c18 sp=0xc0003f3af0 pc=0x4573c5
github.com/libp2p/go-libp2p-kad-dht/providers.(*ProviderManager).run.func1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-kad-dht@v0.25.2/providers/providers_manager.go:160 +0x27c fp=0xc0003f3fe0 sp=0xc0003f3c18 pc=0x10837fc
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc0003f3fe8 sp=0xc0003f3fe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p-kad-dht/providers.(*ProviderManager).run in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-kad-dht@v0.25.2/providers/providers_manager.go:139 +0x65

goroutine 317 gp=0xc000278a80 m=nil [sleep]:
runtime.gopark(0x8cf67e97f353?, 0xc00014ce80?, 0x0?, 0x0?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00056c6f8 sp=0xc00056c6d8 pc=0x479f4e
time.Sleep(0xbebc200)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/time.go:300 +0xf2 fp=0xc00056c730 sp=0xc00056c6f8 pc=0x47de32
main.(*GUI).createWebViewWindow.func4()
	/home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP_PROTO/altgui.go:674 +0x3f fp=0xc00056c7e0 sp=0xc00056c730 pc=0x11cf4bf
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00056c7e8 sp=0xc00056c7e0 pc=0x481ec1
created by main.(*GUI).createWebViewWindow in goroutine 299
	/home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP_PROTO/altgui.go:671 +0x3bc

goroutine 307 gp=0xc000278c40 m=nil [select]:
runtime.gopark(0xc000a33da0?, 0x2?, 0x10?, 0x0?, 0xc000a33d74?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000a33c18 sp=0xc000a33bf8 pc=0x479f4e
runtime.selectgo(0xc000a33da0, 0xc000a33d70, 0x458730?, 0x0, 0xe?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc000a33d40 sp=0xc000a33c18 pc=0x4573c5
github.com/libp2p/go-yamux/v4.(*Stream).Read(0xc0003ae380, {0xc000694b0a, 0x1, 0x1})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.2/stream.go:111 +0x1a5 fp=0xc000a33dd0 sp=0xc000a33d40 pc=0xa64d45
github.com/libp2p/go-libp2p/p2p/muxer/yamux.(*stream).Read(0x48f42c?, {0xc000694b0a?, 0x19848932a7211?, 0x8d53eb0a33c0436b?})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/muxer/yamux/stream.go:17 +0x18 fp=0xc000a33e18 sp=0xc000a33dd0 pc=0xa678d8
github.com/libp2p/go-libp2p/p2p/net/swarm.(*Stream).Read(0xc00017d800, {0xc000694b0a?, 0x448b01538a2be9a1?, 0x18a9f780766aca4b?})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/swarm_stream.go:58 +0x2d fp=0xc000a33e80 sp=0xc000a33e18 pc=0xbb0ecd
github.com/multiformats/go-multistream.(*lazyClientConn[...]).Read(0xc000092e08?, {0xc000694b0a?, 0x1?, 0x1?})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/multiformats/go-multistream@v0.6.0/lazyClient.go:68 +0x98 fp=0xc000a33ec8 sp=0xc000a33e80 pc=0xf79e18
github.com/multiformats/go-multistream.(*lazyClientConn[...]).Read({0xc000694b0a?, 0x134dd00?, 0x994704c7eb283101?})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/multiformats/go-multistream@v0.6.0/lazyClient.go:56 +0x31 fp=0xc000a33f00 sp=0xc000a33ec8 pc=0xf7a1b1
github.com/libp2p/go-libp2p/p2p/host/basic.(*streamWrapper).Read(0xaff9821f39078a9a?, {0xc000694b0a?, 0xeceea453d7a4c616?, 0x659bf535ef218b0?})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/host/basic/basic_host.go:1138 +0x22 fp=0xc000a33f30 sp=0xc000a33f00 pc=0xf76fe2
github.com/libp2p/go-libp2p-pubsub.(*PubSub).handlePeerDead(0xc0002ee248, {0x1b49070, 0xc000682380})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/comm.go:150 +0x73 fp=0xc000a33fb8 sp=0xc000a33f30 pc=0x11855d3
github.com/libp2p/go-libp2p-pubsub.(*PubSub).handleNewPeer.gowrap2()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/comm.go:131 +0x28 fp=0xc000a33fe0 sp=0xc000a33fb8 pc=0x1185388
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc000a33fe8 sp=0xc000a33fe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p-pubsub.(*PubSub).handleNewPeer in goroutine 129
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/comm.go:131 +0x345

goroutine 169 gp=0xc000278e00 m=nil [select]:
runtime.gopark(0xc0007a0f60?, 0x2?, 0x8?, 0x0?, 0xc0007a0e9c?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc0007a0d18 sp=0xc0007a0cf8 pc=0x479f4e
runtime.selectgo(0xc0007a0f60, 0xc0007a0e98, 0x22?, 0x0, 0x798835?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc0007a0e40 sp=0xc0007a0d18 pc=0x4573c5
github.com/libp2p/go-libp2p-kad-dht.(*IpfsDHT).startNetworkSubscriber.func1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-kad-dht@v0.25.2/subscriber_notifee.go:48 +0x151 fp=0xc0007a0fe0 sp=0xc0007a0e40 pc=0x10d6371
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc0007a0fe8 sp=0xc0007a0fe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p-kad-dht.(*IpfsDHT).startNetworkSubscriber in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-kad-dht@v0.25.2/subscriber_notifee.go:43 +0x36f

goroutine 170 gp=0xc000278fc0 m=nil [select]:
runtime.gopark(0xc00031b780?, 0x2?, 0xb8?, 0x35?, 0xc00031b75c?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00031b5f8 sp=0xc00031b5d8 pc=0x479f4e
runtime.selectgo(0xc00031b780, 0xc00031b758, 0xc0002ca000?, 0x0, 0x0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc00031b720 sp=0xc00031b5f8 pc=0x4573c5
github.com/libp2p/go-libp2p-kad-dht.(*IpfsDHT).persistRTPeersInPeerStore(0xc0004b4a88)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-kad-dht@v0.25.2/dht.go:558 +0xf0 fp=0xc00031b7c8 sp=0xc00031b720 pc=0x10ba1d0
github.com/libp2p/go-libp2p-kad-dht.New.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-kad-dht@v0.25.2/dht.go:240 +0x25 fp=0xc00031b7e0 sp=0xc00031b7c8 pc=0x10b8385
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00031b7e8 sp=0xc00031b7e0 pc=0x481ec1
created by github.com/libp2p/go-libp2p-kad-dht.New in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-kad-dht@v0.25.2/dht.go:240 +0x57b

goroutine 171 gp=0xc000279180 m=nil [select]:
runtime.gopark(0xc000884f60?, 0x4?, 0xbd?, 0xac?, 0xc000884f10?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000884da8 sp=0xc000884d88 pc=0x479f4e
runtime.selectgo(0xc000884f60, 0xc000884f08, 0x22?, 0x0, 0x7f000000000?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc000884ed0 sp=0xc000884da8 pc=0x4573c5
github.com/libp2p/go-libp2p-kad-dht.(*IpfsDHT).rtPeerLoop.func1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-kad-dht@v0.25.2/dht.go:613 +0x150 fp=0xc000884fe0 sp=0xc000884ed0 pc=0x10bab10
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc000884fe8 sp=0xc000884fe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p-kad-dht.(*IpfsDHT).rtPeerLoop in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-kad-dht@v0.25.2/dht.go:605 +0x65

goroutine 172 gp=0xc000279340 m=nil [select]:
runtime.gopark(0xc0005aff20?, 0x3?, 0x70?, 0x0?, 0xc0005afe7a?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc0007a1cf8 sp=0xc0007a1cd8 pc=0x479f4e
runtime.selectgo(0xc0007a1f20, 0xc0005afe74, 0x0?, 0x0, 0x1?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc0007a1e20 sp=0xc0007a1cf8 pc=0x4573c5
github.com/libp2p/go-libp2p-kad-dht/rtrefresh.(*RtRefreshManager).loop(0xc000686c80)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-kad-dht@v0.25.2/rtrefresh/rt_refresh_manager.go:197 +0x230 fp=0xc0007a1fc8 sp=0xc0007a1e20 pc=0x10b28b0
github.com/libp2p/go-libp2p-kad-dht/rtrefresh.(*RtRefreshManager).Start.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-kad-dht@v0.25.2/rtrefresh/rt_refresh_manager.go:93 +0x25 fp=0xc0007a1fe0 sp=0xc0007a1fc8 pc=0x10b1125
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc0007a1fe8 sp=0xc0007a1fe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p-kad-dht/rtrefresh.(*RtRefreshManager).Start in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-kad-dht@v0.25.2/rtrefresh/rt_refresh_manager.go:93 +0x65

goroutine 173 gp=0xc000279500 m=nil [select]:
runtime.gopark(0xc0000a59a0?, 0x2?, 0xc8?, 0x58?, 0xc0000a5974?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc0000a5818 sp=0xc0000a57f8 pc=0x479f4e
runtime.selectgo(0xc0000a59a0, 0xc0000a5970, 0x26456a0?, 0x0, 0xc00037b1b0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc0000a5940 sp=0xc0000a5818 pc=0x4573c5
github.com/libp2p/go-libp2p/p2p/net/swarm.(*activeDial).dial(0xc0004fc6c0, {0x1b34c50, 0xc00038b960})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/dial_sync.go:60 +0x23f fp=0xc0000a5a20 sp=0xc0000a5940 pc=0xb9c79f
github.com/libp2p/go-libp2p/p2p/net/swarm.(*dialSync).Dial(0xc00054e180, {0x1b34c50, 0xc00038b960}, {0xc00068e120, 0x22})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/dial_sync.go:98 +0x88 fp=0xc0000a5aa8 sp=0xc0000a5a20 pc=0xb9cc68
github.com/libp2p/go-libp2p/p2p/net/swarm.(*Swarm).dialPeer(0xc000152200, {0x1b34be0, 0xc0005f8b90}, {0xc00068e120, 0x22})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/swarm_dial.go:266 +0x3fd fp=0xc0000a5c20 sp=0xc0000a5aa8 pc=0xba97fd
github.com/libp2p/go-libp2p/p2p/net/swarm.(*Swarm).DialPeer(0x0?, {0x1b34be0?, 0xc0005f8b90?}, {0xc00068e120?, 0xc0004169ff?})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/swarm_dial.go:229 +0x25 fp=0xc0000a5c58 sp=0xc0000a5c20 pc=0xba9385
github.com/libp2p/go-libp2p/p2p/host/basic.(*BasicHost).dialPeer(0xc000394000, {0x1b34be0, 0xc0005f8b90}, {0xc00068e120, 0x22})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/host/basic/basic_host.go:815 +0x12c fp=0xc0000a5d40 sp=0xc0000a5c58 pc=0xf7572c
github.com/libp2p/go-libp2p/p2p/host/basic.(*BasicHost).Connect(0xc000394000, {0x1b34be0, 0xc0005f8b90}, {{0xc00068e120, 0x22}, {0xc000123a60, 0x1, 0x1}})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/host/basic/basic_host.go:808 +0x17f fp=0xc0000a5d98 sp=0xc0000a5d40 pc=0xf7557f
github.com/libp2p/go-libp2p/config.(*closableBasicHost).Connect(0x4?, {0x1b34be0?, 0xc0005f8b90?}, {{0xc00068e120, 0x22}, {0xc000123a60, 0x1, 0x1}})
	<autogenerated>:1 +0x57 fp=0xc0000a5de8 sp=0xc0000a5d98 pc=0xfdaf37
github.com/libp2p/go-libp2p-kad-dht.(*IpfsDHT).fixLowPeers(0xc0004b4a88)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-kad-dht@v0.25.2/dht.go:518 +0x264 fp=0xc0000a5f10 sp=0xc0000a5de8 pc=0x10b9f64
github.com/libp2p/go-libp2p-kad-dht.(*IpfsDHT).runFixLowPeersLoop.func1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-kad-dht@v0.25.2/dht.go:474 +0x65 fp=0xc0000a5fe0 sp=0xc0000a5f10 pc=0x10b9b05
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc0000a5fe8 sp=0xc0000a5fe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p-kad-dht.(*IpfsDHT).runFixLowPeersLoop in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-kad-dht@v0.25.2/dht.go:471 +0x65

goroutine 174 gp=0xc0002796c0 m=nil [select]:
runtime.gopark(0xc000270dc8?, 0x5?, 0xb8?, 0x35?, 0xc000270c96?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000270af0 sp=0xc000270ad0 pc=0x479f4e
runtime.selectgo(0xc000270dc8, 0xc000270c8c, 0x0?, 0x0, 0x0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc000270c18 sp=0xc000270af0 pc=0x4573c5
github.com/libp2p/go-libp2p-kad-dht/providers.(*ProviderManager).run.func1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-kad-dht@v0.25.2/providers/providers_manager.go:160 +0x27c fp=0xc000270fe0 sp=0xc000270c18 pc=0x10837fc
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc000270fe8 sp=0xc000270fe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p-kad-dht/providers.(*ProviderManager).run in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-kad-dht@v0.25.2/providers/providers_manager.go:139 +0x65

goroutine 306 gp=0xc000279880 m=nil [sync.Cond.Wait]:
runtime.gopark(0x0?, 0x0?, 0xa0?, 0xd5?, 0xc000934dd0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000934d90 sp=0xc000934d70 pc=0x479f4e
runtime.goparkunlock(...)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:430
sync.runtime_notifyListWait(0xc000001ed0, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/sema.go:587 +0x159 fp=0xc000934de0 sp=0xc000934d90 pc=0x47b679
sync.(*Cond).Wait(0x1b34be0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/sync/cond.go:71 +0x85 fp=0xc000934e20 sp=0xc000934de0 pc=0x48d1a5
github.com/libp2p/go-libp2p-pubsub.(*rpcQueue).Pop(0xc000001ec0, {0x1b34be0, 0xc0008c0460})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/rpc_queue.go:129 +0x1cb fp=0xc000934ea8 sp=0xc000934e20 pc=0x11a32ab
github.com/libp2p/go-libp2p-pubsub.(*PubSub).handleSendingMessages(0xef6e58?, {0x1b34be0, 0xc0008c0460}, {0x1b49070, 0xc000682380}, 0xc000001ec0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/comm.go:178 +0xe8 fp=0xc000934fa0 sp=0xc000934ea8 pc=0x11857a8
github.com/libp2p/go-libp2p-pubsub.(*PubSub).handleNewPeer.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/comm.go:130 +0x34 fp=0xc000934fe0 sp=0xc000934fa0 pc=0x11853f4
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc000934fe8 sp=0xc000934fe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p-pubsub.(*PubSub).handleNewPeer in goroutine 129
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/comm.go:130 +0x2e5

goroutine 125 gp=0xc000279a40 m=nil [sleep]:
runtime.gopark(0x8cf63f62ad04?, 0x0?, 0x0?, 0x0?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc0006b56e0 sp=0xc0006b56c0 pc=0x479f4e
time.Sleep(0x37e11d600)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/time.go:300 +0xf2 fp=0xc0006b5718 sp=0xc0006b56e0 pc=0x47de32
main.main.func3()
	/home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP_PROTO/main.go:883 +0x4c fp=0xc0006b57e0 sp=0xc0006b5718 pc=0x1252acc
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc0006b57e8 sp=0xc0006b57e0 pc=0x481ec1
created by main.main in goroutine 1
	/home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP_PROTO/main.go:877 +0x48f5

goroutine 143 gp=0xc000279c00 m=nil [select]:
runtime.gopark(0xc0006b5e40?, 0x2?, 0x0?, 0x0?, 0xc0006b5e2c?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00043fcd0 sp=0xc00043fcb0 pc=0x479f4e
runtime.selectgo(0xc00043fe40, 0xc0006b5e28, 0x0?, 0x0, 0x0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc00043fdf8 sp=0xc00043fcd0 pc=0x4573c5
github.com/libp2p/go-libp2p-pubsub.(*Subscription).Next(0xc0008c05f0, {0x1b34be0, 0xc0008c0460})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/subscription.go:26 +0x85 fp=0xc00043fe70 sp=0xc00043fdf8 pc=0x11a9985
main.(*P2PConsensusManager).handleBlocks(0xc000a28090)
	/home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP_PROTO/p2p_consensus.go:183 +0x45 fp=0xc00043ffc8 sp=0xc00043fe70 pc=0x122c145
main.(*P2PConsensusManager).Start.gowrap1()
	/home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP_PROTO/p2p_consensus.go:169 +0x25 fp=0xc00043ffe0 sp=0xc00043ffc8 pc=0x122c065
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00043ffe8 sp=0xc00043ffe0 pc=0x481ec1
created by main.(*P2PConsensusManager).Start in goroutine 295
	/home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP_PROTO/p2p_consensus.go:169 +0x132

goroutine 179 gp=0xc0006b6000 m=nil [select]:
runtime.gopark(0xc0006e1f60?, 0x2?, 0xb0?, 0x1c?, 0xc0006e1e9c?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc0006e1d18 sp=0xc0006e1cf8 pc=0x479f4e
runtime.selectgo(0xc0006e1f60, 0xc0006e1e98, 0x22?, 0x0, 0x0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc0006e1e40 sp=0xc0006e1d18 pc=0x4573c5
github.com/libp2p/go-libp2p-kad-dht.(*IpfsDHT).startNetworkSubscriber.func1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-kad-dht@v0.25.2/subscriber_notifee.go:48 +0x151 fp=0xc0006e1fe0 sp=0xc0006e1e40 pc=0x10d6371
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc0006e1fe8 sp=0xc0006e1fe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p-kad-dht.(*IpfsDHT).startNetworkSubscriber in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-kad-dht@v0.25.2/subscriber_notifee.go:43 +0x36f

goroutine 180 gp=0xc0006b61c0 m=nil [select]:
runtime.gopark(0xc00056b780?, 0x2?, 0xb8?, 0x35?, 0xc00056b75c?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00056b5f8 sp=0xc00056b5d8 pc=0x479f4e
runtime.selectgo(0xc00056b780, 0xc00056b758, 0x0?, 0x0, 0x0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc00056b720 sp=0xc00056b5f8 pc=0x4573c5
github.com/libp2p/go-libp2p-kad-dht.(*IpfsDHT).persistRTPeersInPeerStore(0xc0004b5508)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-kad-dht@v0.25.2/dht.go:558 +0xf0 fp=0xc00056b7c8 sp=0xc00056b720 pc=0x10ba1d0
github.com/libp2p/go-libp2p-kad-dht.New.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-kad-dht@v0.25.2/dht.go:240 +0x25 fp=0xc00056b7e0 sp=0xc00056b7c8 pc=0x10b8385
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00056b7e8 sp=0xc00056b7e0 pc=0x481ec1
created by github.com/libp2p/go-libp2p-kad-dht.New in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-kad-dht@v0.25.2/dht.go:240 +0x57b

goroutine 181 gp=0xc0006b6380 m=nil [select]:
runtime.gopark(0xc000937f60?, 0x4?, 0x50?, 0x7e?, 0xc000937f10?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000937da8 sp=0xc000937d88 pc=0x479f4e
runtime.selectgo(0xc000937f60, 0xc000937f08, 0x26?, 0x0, 0x0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc000937ed0 sp=0xc000937da8 pc=0x4573c5
github.com/libp2p/go-libp2p-kad-dht.(*IpfsDHT).rtPeerLoop.func1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-kad-dht@v0.25.2/dht.go:613 +0x150 fp=0xc000937fe0 sp=0xc000937ed0 pc=0x10bab10
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc000937fe8 sp=0xc000937fe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p-kad-dht.(*IpfsDHT).rtPeerLoop in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-kad-dht@v0.25.2/dht.go:605 +0x65

goroutine 182 gp=0xc0006b6540 m=nil [select]:
runtime.gopark(0xc0005b1f20?, 0x3?, 0x1?, 0x0?, 0xc0005b1e7a?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc0007a2cf8 sp=0xc0007a2cd8 pc=0x479f4e
runtime.selectgo(0xc0007a2f20, 0xc0005b1e74, 0x2?, 0x0, 0x1?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc0007a2e20 sp=0xc0007a2cf8 pc=0x4573c5
github.com/libp2p/go-libp2p-kad-dht/rtrefresh.(*RtRefreshManager).loop(0xc000686d20)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-kad-dht@v0.25.2/rtrefresh/rt_refresh_manager.go:197 +0x230 fp=0xc0007a2fc8 sp=0xc0007a2e20 pc=0x10b28b0
github.com/libp2p/go-libp2p-kad-dht/rtrefresh.(*RtRefreshManager).Start.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-kad-dht@v0.25.2/rtrefresh/rt_refresh_manager.go:93 +0x25 fp=0xc0007a2fe0 sp=0xc0007a2fc8 pc=0x10b1125
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc0007a2fe8 sp=0xc0007a2fe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p-kad-dht/rtrefresh.(*RtRefreshManager).Start in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-kad-dht@v0.25.2/rtrefresh/rt_refresh_manager.go:93 +0x65

goroutine 183 gp=0xc0006b6700 m=nil [select]:
runtime.gopark(0xc0003ef9a0?, 0x2?, 0xc8?, 0xf8?, 0xc0003ef974?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc0003ef818 sp=0xc0003ef7f8 pc=0x479f4e
runtime.selectgo(0xc0003ef9a0, 0xc0003ef970, 0x26456a0?, 0x0, 0xc0004fc6f0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc0003ef940 sp=0xc0003ef818 pc=0x4573c5
github.com/libp2p/go-libp2p/p2p/net/swarm.(*activeDial).dial(0xc0004fc6c0, {0x1b34c50, 0xc00027b3b0})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/dial_sync.go:60 +0x23f fp=0xc0003efa20 sp=0xc0003ef940 pc=0xb9c79f
github.com/libp2p/go-libp2p/p2p/net/swarm.(*dialSync).Dial(0xc00054e180, {0x1b34c50, 0xc00027b3b0}, {0xc00068e120, 0x22})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/dial_sync.go:98 +0x88 fp=0xc0003efaa8 sp=0xc0003efa20 pc=0xb9cc68
github.com/libp2p/go-libp2p/p2p/net/swarm.(*Swarm).dialPeer(0xc000152200, {0x1b34be0, 0xc0005f8d20}, {0xc00068e120, 0x22})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/swarm_dial.go:266 +0x3fd fp=0xc0003efc20 sp=0xc0003efaa8 pc=0xba97fd
github.com/libp2p/go-libp2p/p2p/net/swarm.(*Swarm).DialPeer(0x0?, {0x1b34be0?, 0xc0005f8d20?}, {0xc00068e120?, 0xc0004169ff?})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/swarm_dial.go:229 +0x25 fp=0xc0003efc58 sp=0xc0003efc20 pc=0xba9385
github.com/libp2p/go-libp2p/p2p/host/basic.(*BasicHost).dialPeer(0xc000394000, {0x1b34be0, 0xc0005f8d20}, {0xc00068e120, 0x22})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/host/basic/basic_host.go:815 +0x12c fp=0xc0003efd40 sp=0xc0003efc58 pc=0xf7572c
github.com/libp2p/go-libp2p/p2p/host/basic.(*BasicHost).Connect(0xc000394000, {0x1b34be0, 0xc0005f8d20}, {{0xc00068e120, 0x22}, {0xc000123a60, 0x1, 0x1}})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/host/basic/basic_host.go:808 +0x17f fp=0xc0003efd98 sp=0xc0003efd40 pc=0xf7557f
github.com/libp2p/go-libp2p/config.(*closableBasicHost).Connect(0x4?, {0x1b34be0?, 0xc0005f8d20?}, {{0xc00068e120, 0x22}, {0xc000123a60, 0x1, 0x1}})
	<autogenerated>:1 +0x57 fp=0xc0003efde8 sp=0xc0003efd98 pc=0xfdaf37
github.com/libp2p/go-libp2p-kad-dht.(*IpfsDHT).fixLowPeers(0xc0004b5508)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-kad-dht@v0.25.2/dht.go:518 +0x264 fp=0xc0003eff10 sp=0xc0003efde8 pc=0x10b9f64
github.com/libp2p/go-libp2p-kad-dht.(*IpfsDHT).runFixLowPeersLoop.func1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-kad-dht@v0.25.2/dht.go:474 +0x65 fp=0xc0003effe0 sp=0xc0003eff10 pc=0x10b9b05
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc0003effe8 sp=0xc0003effe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p-kad-dht.(*IpfsDHT).runFixLowPeersLoop in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-kad-dht@v0.25.2/dht.go:471 +0x65

goroutine 271 gp=0xc0004f6540 m=nil [IO wait]:
runtime.gopark(0x441849?, 0x5?, 0x0?, 0x1b?, 0xb?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc0003b1880 sp=0xc0003b1860 pc=0x479f4e
runtime.netpollblock(0x49d3b8?, 0x40ff66?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/netpoll.go:575 +0xf7 fp=0xc0003b18b8 sp=0xc0003b1880 pc=0x43ddd7
internal/poll.runtime_pollWait(0x7f0270667dc0, 0x72)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/netpoll.go:351 +0x85 fp=0xc0003b18d8 sp=0xc0003b18b8 pc=0x479245
internal/poll.(*pollDesc).wait(0xc000169300?, 0xc0007ca000?, 0x0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_poll_runtime.go:84 +0x27 fp=0xc0003b1900 sp=0xc0003b18d8 pc=0x50a5c7
internal/poll.(*pollDesc).waitRead(...)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_poll_runtime.go:89
internal/poll.(*FD).Read(0xc000169300, {0xc0007ca000, 0x1b00, 0x1b00})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_unix.go:165 +0x27a fp=0xc0003b1998 sp=0xc0003b1900 pc=0x50b8ba
net.(*netFD).Read(0xc000169300, {0xc0007ca000?, 0x59?, 0xc0003b1a38?})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/fd_posix.go:55 +0x25 fp=0xc0003b19e0 sp=0xc0003b1998 pc=0x61be05
net.(*conn).Read(0xc000a30040, {0xc0007ca000?, 0xc0003b1ab0?, 0x41f425?})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/net.go:189 +0x45 fp=0xc0003b1a28 sp=0xc0003b19e0 pc=0x62dc45
go:(*struct { *net.TCPConn; github.com/multiformats/go-multiaddr/net.maEndpoints }).Read(0x0?, {0xc0007ca000?, 0x18?, 0x1000000001b00?})
	<autogenerated>:1 +0x26 fp=0xc0003b1a58 sp=0xc0003b1a28 pc=0x9f03a6
crypto/tls.(*atLeastReader).Read(0xc000769c68, {0xc0007ca000?, 0x0?, 0xc000769c68?})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/crypto/tls/conn.go:809 +0x3b fp=0xc0003b1aa0 sp=0xc0003b1a58 pc=0x69d43b
bytes.(*Buffer).ReadFrom(0xc000852638, {0x1b271a0, 0xc000769c68})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/bytes/buffer.go:211 +0x98 fp=0xc0003b1af8 sp=0xc0003b1aa0 pc=0x53bcb8
crypto/tls.(*Conn).readFromUntil(0xc000852388, {0x7f02703060b8, 0xc0008b29f0}, 0xc0003b1b90?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/crypto/tls/conn.go:831 +0xde fp=0xc0003b1b30 sp=0xc0003b1af8 pc=0x69d61e
crypto/tls.(*Conn).readRecordOrCCS(0xc000852388, 0x0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/crypto/tls/conn.go:629 +0x3cf fp=0xc0003b1da8 sp=0xc0003b1b30 pc=0x69a72f
crypto/tls.(*Conn).readRecord(...)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/crypto/tls/conn.go:591
crypto/tls.(*Conn).Read(0xc000852388, {0xc0008a0ab0, 0xc, 0xc000533280?})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/crypto/tls/conn.go:1385 +0x150 fp=0xc0003b1e18 sp=0xc0003b1da8 pc=0x6a0f90
github.com/libp2p/go-libp2p/p2p/security/tls.(*conn).Read(0xc000533250?, {0xc0008a0ab0?, 0xc0003b1e80?, 0xa615b2?})
	<autogenerated>:1 +0x25 fp=0xc0003b1e48 sp=0xc0003b1e18 pc=0xbdede5
io.ReadAtLeast({0x7f0270446020, 0xc0000e8a10}, {0xc0008a0ab0, 0xc, 0xc}, 0xc)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/io/io.go:335 +0x90 fp=0xc0003b1e90 sp=0xc0003b1e48 pc=0x4ba2d0
io.ReadFull(...)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/io/io.go:354
github.com/libp2p/go-yamux/v4.(*Session).recvLoop(0xc000533200)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.2/session.go:674 +0xe5 fp=0xc0003b1fa0 sp=0xc0003b1e90 pc=0xa62725
github.com/libp2p/go-yamux/v4.(*Session).recv(0xc000533200)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.2/session.go:646 +0x18 fp=0xc0003b1fc8 sp=0xc0003b1fa0 pc=0xa625f8
github.com/libp2p/go-yamux/v4.newSession.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.2/session.go:162 +0x25 fp=0xc0003b1fe0 sp=0xc0003b1fc8 pc=0xa5f525
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc0003b1fe8 sp=0xc0003b1fe0 pc=0x481ec1
created by github.com/libp2p/go-yamux/v4.newSession in goroutine 158
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.2/session.go:162 +0x516

goroutine 160 gp=0xc0004f6700 m=nil [select]:
runtime.gopark(0xc00080fd80?, 0x3?, 0xfd?, 0x4c?, 0xc00080fa1a?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00080f878 sp=0xc00080f858 pc=0x479f4e
runtime.selectgo(0xc00080fd80, 0xc00080fa14, 0xc0005fdcb0?, 0x0, 0x12f95c0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc00080f9a0 sp=0xc00080f878 pc=0x4573c5
github.com/libp2p/go-libp2p/p2p/net/swarm.(*dialWorker).loop(0xc000800900)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/dial_worker.go:161 +0x36a fp=0xc00080ff50 sp=0xc00080f9a0 pc=0xb9d1ea
github.com/libp2p/go-libp2p/p2p/net/swarm.(*Swarm).dialWorkerLoop(0xc000152200, {0xc00068e120, 0x22}, 0xc000059030)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/swarm_dial.go:296 +0xf8 fp=0xc00080ff88 sp=0xc00080ff50 pc=0xba9df8
github.com/libp2p/go-libp2p/p2p/net/swarm.(*Swarm).dialWorkerLoop-fm({0xc00068e120?, 0xc0006b0f70?}, 0x0?)
	<autogenerated>:1 +0x31 fp=0xc00080ffb8 sp=0xc00080ff88 pc=0xbb7a71
github.com/libp2p/go-libp2p/p2p/net/swarm.(*dialSync).getActiveDial.gowrap2()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/dial_sync.go:82 +0x2c fp=0xc00080ffe0 sp=0xc00080ffb8 pc=0xb9cb4c
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00080ffe8 sp=0xc00080ffe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p/p2p/net/swarm.(*dialSync).getActiveDial in goroutine 183
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/dial_sync.go:82 +0x1ec

goroutine 161 gp=0xc0004f68c0 m=nil [IO wait]:
runtime.gopark(0x7f02b78e3a68?, 0x40?, 0x40?, 0x93?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000808ca0 sp=0xc000808c80 pc=0x479f4e
runtime.netpollblock(0xc000808d10?, 0x44ef5f?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/netpoll.go:575 +0xf7 fp=0xc000808cd8 sp=0xc000808ca0 pc=0x43ddd7
internal/poll.runtime_pollWait(0x7f0270667ca8, 0x77)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/netpoll.go:351 +0x85 fp=0xc000808cf8 sp=0xc000808cd8 pc=0x479245
internal/poll.(*pollDesc).wait(0xc000169480?, 0x0?, 0x0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_poll_runtime.go:84 +0x27 fp=0xc000808d20 sp=0xc000808cf8 pc=0x50a5c7
internal/poll.(*pollDesc).waitWrite(...)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_poll_runtime.go:93
internal/poll.(*FD).WaitWrite(...)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_unix.go:692
net.(*netFD).connect(0xc000169480, {0x1b34c50, 0xc00027bb90}, {0x0?, 0xc0004ea660?}, {0x1b265e0?, 0xc00004a880?})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/fd_unix.go:141 +0x6eb fp=0xc000808e98 sp=0xc000808d20 pc=0x61d6cb
net.(*netFD).dial(0xc000169480, {0x1b34c50, 0xc00027bb90}, {0x1b3b780?, 0xc00047bcb0?}, {0x1b3b780, 0xc0004fc870}, 0xc000809120?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/sock_posix.go:124 +0x3bc fp=0xc000808f70 sp=0xc000808e98 pc=0x63365c
net.socket({0x1b34c50, 0xc00027bb90}, {0x15c7cea, 0x4}, 0x2, 0x1, 0x41dc8b?, 0x0, {0x1b3b780, 0xc00047bcb0}, ...)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/sock_posix.go:70 +0x29b fp=0xc000809018 sp=0xc000808f70 pc=0x63319b
net.internetSocket({0x1b34c50, 0xc00027bb90}, {0x15c7cea, 0x4}, {0x1b3b780, 0xc00047bcb0}, {0x1b3b780, 0xc0004fc870}, 0x1, 0x0, ...)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/ipsock_posix.go:167 +0xf8 fp=0xc000809090 sp=0xc000809018 pc=0x627fd8
net.(*sysDialer).doDialTCPProto(0xc0008100c0, {0x1b34c50, 0xc00027bb90}, 0xc00047bcb0, 0xc0004fc870, 0x0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/tcpsock_posix.go:85 +0xec fp=0xc000809140 sp=0xc000809090 pc=0x63786c
net.(*sysDialer).doDialTCP(...)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/tcpsock_posix.go:75
net.(*sysDialer).dialTCP(0xc0008091a8?, {0x1b34c50?, 0xc00027bb90?}, 0x1385700?, 0xc000809218?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/tcpsock_posix.go:71 +0x69 fp=0xc000809180 sp=0xc000809140 pc=0x637709
net.(*sysDialer).dialSingle(0xc0008100c0, {0x1b34c50, 0xc00027bb90}, {0x1b2d140, 0xc0004fc870})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/dial.go:670 +0x27d fp=0xc000809250 sp=0xc000809180 pc=0x61269d
net.(*sysDialer).dialSerial(0xc0008100c0, {0x1b34c50, 0xc00027bb90}, {0xc00047df60?, 0x1, 0xc0004fc840?})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/dial.go:635 +0x24e fp=0xc000809358 sp=0xc000809250 pc=0x611fce
net.(*sysDialer).dialParallel(0x1b2d140?, {0x1b34c50?, 0xc00027bb90?}, {0xc00047df60?, 0xc00027bb90?, 0x15c7b2a?}, {0x0?, 0x15c7cea?, 0x1?})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/dial.go:536 +0x3a7 fp=0xc000809570 sp=0xc000809358 pc=0x6116a7
net.(*Dialer).DialContext(0xc0008096f0, {0x1b34c50, 0xc00027bb90}, {0x15c7cea, 0x4}, {0xc0004ea648, 0x14})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/dial.go:527 +0x6a5 fp=0xc000809690 sp=0xc000809570 pc=0x611125
github.com/libp2p/go-libp2p/p2p/net/reuseport.reuseDial({0x1b34c50, 0xc00027bb90}, 0x1b46600?, {0x15c7cea, 0x4}, {0xc0004ea648, 0x14})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/reuseport/reuseport.go:23 +0xd1 fp=0xc0008097b0 sp=0xc000809690 pc=0xcff511
github.com/libp2p/go-libp2p/p2p/net/reuseport.(*dialer).DialContext(0xc0004e87d0, {0x1b34c50, 0xc00027bb90}, {0x15c7cea, 0x4}, {0xc0004ea648, 0x14})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/reuseport/dialer.go:86 +0x255 fp=0xc0008098f8 sp=0xc0008097b0 pc=0xcfe8d5
github.com/libp2p/go-libp2p/p2p/net/reuseport.(*Transport).DialContext(0xc0004185e0, {0x1b34c50, 0xc00027bb90}, {0x1b46600?, 0xc0002c5710?})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/reuseport/dial.go:36 +0xe8 fp=0xc000809980 sp=0xc0008098f8 pc=0xcfe368
github.com/libp2p/go-libp2p/p2p/transport/tcp.(*TcpTransport).maDial(0xc0004185a0?, {0x1b34c50?, 0xc00027bab0?}, {0x1b46600?, 0xc0002c5710?})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/transport/tcp/tcp.go:243 +0x17f fp=0xc000809a90 sp=0xc000809980 pc=0xd084df
github.com/libp2p/go-libp2p/p2p/transport/tcp.(*TcpTransport).dialWithScope(0xc0004185a0, {0x1b34c50, 0xc00027bab0}, {0x1b46600, 0xc0002c5710}, {0xc00068e120, 0x22}, {0x1b3db70, 0xc0004e8910}, 0xc000059110)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/transport/tcp/tcp.go:274 +0x229 fp=0xc000809bd0 sp=0xc000809a90 pc=0xd08c09
github.com/libp2p/go-libp2p/p2p/transport/tcp.(*TcpTransport).DialWithUpdates(0xc0004185a0, {0x1b34c50, 0xc00027bab0}, {0x1b46600, 0xc0002c5710}, {0xc00068e120, 0x22}, 0xc000059110)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/transport/tcp/tcp.go:261 +0x245 fp=0xc000809cb8 sp=0xc000809bd0 pc=0xd08925
github.com/libp2p/go-libp2p/p2p/net/swarm.(*Swarm).dialAddr(0xc000152200, {0x1b34c50, 0xc00027bab0}, {0xc00068e120, 0x22}, {0x1b46600, 0xc0002c5710}, 0xc000059110)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/swarm_dial.go:604 +0x3dd fp=0xc000809e80 sp=0xc000809cb8 pc=0xbac4fd
github.com/libp2p/go-libp2p/p2p/net/swarm.(*Swarm).dialAddr-fm({0x1b34c50?, 0xc00027bab0?}, {0xc00068e120?, 0x0?}, {0x1b46600?, 0xc0002c5710?}, 0x0?)
	<autogenerated>:1 +0x55 fp=0xc000809ed0 sp=0xc000809e80 pc=0xbb7b15
github.com/libp2p/go-libp2p/p2p/net/swarm.(*dialLimiter).executeDial(0xc00018e4b0, 0xc0004e7640)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/limiter.go:213 +0xee fp=0xc000809fc0 sp=0xc000809ed0 pc=0xba0fce
github.com/libp2p/go-libp2p/p2p/net/swarm.(*dialLimiter).addCheckFdLimit.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/limiter.go:163 +0x25 fp=0xc000809fe0 sp=0xc000809fc0 pc=0xba07e5
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc000809fe8 sp=0xc000809fe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p/p2p/net/swarm.(*dialLimiter).addCheckFdLimit in goroutine 160
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/limiter.go:163 +0x485

goroutine 194 gp=0xc0004f6a80 m=nil [select]:
runtime.gopark(0xc0006b0fa0?, 0x2?, 0x80?, 0xde?, 0xc0006b0f7c?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc0006b0e20 sp=0xc0006b0e00 pc=0x479f4e
runtime.selectgo(0xc0006b0fa0, 0xc0006b0f78, 0xc00018e4b0?, 0x0, 0xc0004fc750?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc0006b0f48 sp=0xc0006b0e20 pc=0x4573c5
net.(*netFD).connect.func2()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/fd_unix.go:118 +0x7a fp=0xc0006b0fe0 sp=0xc0006b0f48 pc=0x61db5a
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc0006b0fe8 sp=0xc0006b0fe0 pc=0x481ec1
created by net.(*netFD).connect in goroutine 161
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/fd_unix.go:117 +0x367

goroutine 184 gp=0xc0006b68c0 m=nil [IO wait]:
runtime.gopark(0xc000940eb0?, 0xc0003ecd80?, 0x80?, 0x6d?, 0x5?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc0003ecd40 sp=0xc0003ecd20 pc=0x479f4e
runtime.netpollblock(0x1b26000?, 0x258ed70?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/netpoll.go:575 +0xf7 fp=0xc0003ecd78 sp=0xc0003ecd40 pc=0x43ddd7
internal/poll.runtime_pollWait(0x7f0270667b90, 0x72)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/netpoll.go:351 +0x85 fp=0xc0003ecd98 sp=0xc0003ecd78 pc=0x479245
internal/poll.(*pollDesc).wait(0xc0005fb800?, 0x50?, 0x0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_poll_runtime.go:84 +0x27 fp=0xc0003ecdc0 sp=0xc0003ecd98 pc=0x50a5c7
internal/poll.(*pollDesc).waitRead(...)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_poll_runtime.go:89
internal/poll.(*FD).RawRead(0xc0005fb800, 0xc000940eb0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_unix.go:717 +0x125 fp=0xc0003ece20 sp=0xc0003ecdc0 pc=0x510605
net.(*rawConn).Read(0xc0002a54b0, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/rawconn.go:44 +0x36 fp=0xc0003ece58 sp=0xc0003ece20 pc=0x632416
golang.org/x/net/internal/socket.(*Conn).recvMsg(0xc000682800, 0xc000795260, 0x0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/x/net@v0.35.0/internal/socket/rawconn_msg.go:27 +0x144 fp=0xc0003eceb0 sp=0xc0003ece58 pc=0xb32404
golang.org/x/net/internal/socket.(*Conn).RecvMsg(...)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/x/net@v0.35.0/internal/socket/socket.go:247
golang.org/x/net/ipv4.(*payloadHandler).ReadFrom(0xc0005f8f10, {0xc0007a8000, 0x10000, 0x10000})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/x/net@v0.35.0/ipv4/payload_cmsg.go:31 +0x4ae fp=0xc0003ecf50 sp=0xc0003eceb0 pc=0xb3c32e
github.com/libp2p/zeroconf/v2.(*Server).recv4(0xc0005f2cc0, 0xc0005f8f00)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/zeroconf/v2@v2.2.0/server.go:275 +0xb5 fp=0xc0003ecfc0 sp=0xc0003ecf50 pc=0x10e2af5
github.com/libp2p/zeroconf/v2.(*Server).start.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/zeroconf/v2@v2.2.0/server.go:213 +0x25 fp=0xc0003ecfe0 sp=0xc0003ecfc0 pc=0x10e26a5
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc0003ecfe8 sp=0xc0003ecfe0 pc=0x481ec1
created by github.com/libp2p/zeroconf/v2.(*Server).start in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/zeroconf/v2@v2.2.0/server.go:213 +0x8b

goroutine 185 gp=0xc0006b6a80 m=nil [IO wait]:
runtime.gopark(0xc000940dc0?, 0xc000887d30?, 0xc0?, 0xd5?, 0xc000887da8?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000887cf0 sp=0xc000887cd0 pc=0x479f4e
runtime.netpollblock(0x1b26000?, 0x258ed70?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/netpoll.go:575 +0xf7 fp=0xc000887d28 sp=0xc000887cf0 pc=0x43ddd7
internal/poll.runtime_pollWait(0x7f0270667a78, 0x72)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/netpoll.go:351 +0x85 fp=0xc000887d48 sp=0xc000887d28 pc=0x479245
internal/poll.(*pollDesc).wait(0xc0005fb880?, 0x50?, 0x0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_poll_runtime.go:84 +0x27 fp=0xc000887d70 sp=0xc000887d48 pc=0x50a5c7
internal/poll.(*pollDesc).waitRead(...)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_poll_runtime.go:89
internal/poll.(*FD).RawRead(0xc0005fb880, 0xc000940dc0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_unix.go:717 +0x125 fp=0xc000887dd0 sp=0xc000887d70 pc=0x510605
net.(*rawConn).Read(0xc0002a54c0, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/rawconn.go:44 +0x36 fp=0xc000887e08 sp=0xc000887dd0 pc=0x632416
golang.org/x/net/internal/socket.(*Conn).recvMsg(0xc000682820, 0xc000795140, 0x0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/x/net@v0.35.0/internal/socket/rawconn_msg.go:27 +0x144 fp=0xc000887e60 sp=0xc000887e08 pc=0xb32404
golang.org/x/net/internal/socket.(*Conn).RecvMsg(...)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/x/net@v0.35.0/internal/socket/socket.go:247
golang.org/x/net/ipv6.(*payloadHandler).ReadFrom(0xc0005f8f60, {0xc0007ba000, 0x10000, 0x10000})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/x/net@v0.35.0/ipv6/payload_cmsg.go:31 +0x38d fp=0xc000887f50 sp=0xc000887e60 pc=0xb43aad
github.com/libp2p/zeroconf/v2.(*Server).recv6(0xc0005f2cc0, 0xc0005f8f50)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/zeroconf/v2@v2.2.0/server.go:300 +0xb5 fp=0xc000887fc0 sp=0xc000887f50 pc=0x10e2c95
github.com/libp2p/zeroconf/v2.(*Server).start.gowrap2()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/zeroconf/v2@v2.2.0/server.go:217 +0x25 fp=0xc000887fe0 sp=0xc000887fc0 pc=0x10e2645
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc000887fe8 sp=0xc000887fe0 pc=0x481ec1
created by github.com/libp2p/zeroconf/v2.(*Server).start in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/zeroconf/v2@v2.2.0/server.go:217 +0x107

goroutine 186 gp=0xc0006b6c40 m=nil [select]:
runtime.gopark(0xc000271ec8?, 0x2?, 0x18?, 0xd1?, 0xc000271e04?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000271ca0 sp=0xc000271c80 pc=0x479f4e
runtime.selectgo(0xc000271ec8, 0xc000271e00, 0xb?, 0x0, 0x1?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc000271dc8 sp=0xc000271ca0 pc=0x4573c5
github.com/libp2p/zeroconf/v2.(*Server).probe(0xc0005f2cc0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/zeroconf/v2@v2.2.0/server.go:625 +0x5af fp=0xc000271fc8 sp=0xc000271dc8 pc=0x10e49cf
github.com/libp2p/zeroconf/v2.(*Server).start.gowrap3()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/zeroconf/v2@v2.2.0/server.go:220 +0x25 fp=0xc000271fe0 sp=0xc000271fc8 pc=0x10e25e5
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc000271fe8 sp=0xc000271fe0 pc=0x481ec1
created by github.com/libp2p/zeroconf/v2.(*Server).start in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/zeroconf/v2@v2.2.0/server.go:220 +0x159

goroutine 187 gp=0xc0006b6e00 m=nil [chan receive]:
runtime.gopark(0x1?, 0x997f687200000003?, 0x80?, 0x3c?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000883dd0 sp=0xc000883db0 pc=0x479f4e
runtime.chanrecv(0xc000359b20, 0xc000883f70, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/chan.go:639 +0x41c fp=0xc000883e48 sp=0xc000883dd0 pc=0x412b5c
runtime.chanrecv2(0xc000912840?, 0x6?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/chan.go:494 +0x12 fp=0xc000883e70 sp=0xc000883e48 pc=0x412732
github.com/libp2p/go-libp2p/p2p/discovery/mdns.(*mdnsService).startResolver.func1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/discovery/mdns/mdns.go:158 +0x99 fp=0xc000883fe0 sp=0xc000883e70 pc=0x10e7979
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc000883fe8 sp=0xc000883fe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p/p2p/discovery/mdns.(*mdnsService).startResolver in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/discovery/mdns/mdns.go:156 +0x98

goroutine 188 gp=0xc0006b6fc0 m=nil [select]:
runtime.gopark(0xc000503e30?, 0x3?, 0x70?, 0x0?, 0xc000503df2?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000503c90 sp=0xc000503c70 pc=0x479f4e
runtime.selectgo(0xc000503e30, 0xc000503dec, 0x7f02b78e3a68?, 0x0, 0x7f02b78ecc48?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc000503db8 sp=0xc000503c90 pc=0x4573c5
github.com/libp2p/zeroconf/v2.(*client).periodicQuery(0xc0004fcc30, {0x1b34be0, 0xc0004e8af0}, 0xc00035ebe0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/zeroconf/v2@v2.2.0/client.go:382 +0x165 fp=0xc000503e88 sp=0xc000503db8 pc=0x10e08a5
github.com/libp2p/zeroconf/v2.(*client).run(0xc0004fcc30, {0x1b34be0?, 0xc0005f8dc0?}, 0xc00035ebe0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/zeroconf/v2@v2.2.0/client.go:123 +0xe7 fp=0xc000503ee8 sp=0xc000503e88 pc=0x10de5e7
github.com/libp2p/zeroconf/v2.Browse({0x1b34be0, 0xc0005f8dc0}, {0x15cc222, 0xa}, {0x15c81fd, 0x5}, 0xc000359b20, {0x0, 0x0, 0x0})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/zeroconf/v2@v2.2.0/client.go:80 +0x1a6 fp=0xc000503f50 sp=0xc000503ee8 pc=0x10de486
github.com/libp2p/go-libp2p/p2p/discovery/mdns.(*mdnsService).startResolver.func2()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/discovery/mdns/mdns.go:189 +0x85 fp=0xc000503fe0 sp=0xc000503f50 pc=0x10e77e5
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc000503fe8 sp=0xc000503fe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p/p2p/discovery/mdns.(*mdnsService).startResolver in goroutine 1
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/discovery/mdns/mdns.go:187 +0x105

goroutine 195 gp=0xc0004f6c40 m=nil [select]:
runtime.gopark(0xc0003f0db0?, 0x3?, 0xfd?, 0x4c?, 0xc0003f0c82?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc0003f0b10 sp=0xc0003f0af0 pc=0x479f4e
runtime.selectgo(0xc0003f0db0, 0xc0003f0c7c, 0xc000a10a00?, 0x0, 0x4d378d65?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc0003f0c38 sp=0xc0003f0b10 pc=0x4573c5
github.com/libp2p/zeroconf/v2.(*client).mainloop(0xc0004fcc30, {0x1b34be0, 0xc0004e8af0}, 0xc00035ebe0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/zeroconf/v2@v2.2.0/client.go:187 +0x34d fp=0xc0003f0f90 sp=0xc0003f0c38 pc=0x10debed
github.com/libp2p/zeroconf/v2.(*client).run.func1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/zeroconf/v2@v2.2.0/client.go:118 +0x53 fp=0xc0003f0fe0 sp=0xc0003f0f90 pc=0x10de6b3
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc0003f0fe8 sp=0xc0003f0fe0 pc=0x481ec1
created by github.com/libp2p/zeroconf/v2.(*client).run in goroutine 188
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/zeroconf/v2@v2.2.0/client.go:116 +0xce

goroutine 196 gp=0xc0004f6e00 m=nil [IO wait]:
runtime.gopark(0xc000940fa0?, 0xc0003f2ce8?, 0xa0?, 0x6e?, 0xb5a88d?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc0003f2ca8 sp=0xc0003f2c88 pc=0x479f4e
runtime.netpollblock(0x1b26000?, 0x258ed70?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/netpoll.go:575 +0xf7 fp=0xc0003f2ce0 sp=0xc0003f2ca8 pc=0x43ddd7
internal/poll.runtime_pollWait(0x7f0270667960, 0x72)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/netpoll.go:351 +0x85 fp=0xc0003f2d00 sp=0xc0003f2ce0 pc=0x479245
internal/poll.(*pollDesc).wait(0xc000169980?, 0x50?, 0x0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_poll_runtime.go:84 +0x27 fp=0xc0003f2d28 sp=0xc0003f2d00 pc=0x50a5c7
internal/poll.(*pollDesc).waitRead(...)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_poll_runtime.go:89
internal/poll.(*FD).RawRead(0xc000169980, 0xc000940fa0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_unix.go:717 +0x125 fp=0xc0003f2d88 sp=0xc0003f2d28 pc=0x510605
net.(*rawConn).Read(0xc000090750, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/rawconn.go:44 +0x36 fp=0xc0003f2dc0 sp=0xc0003f2d88 pc=0x632416
golang.org/x/net/internal/socket.(*Conn).recvMsg(0xc0004f2660, 0xc000795380, 0x0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/x/net@v0.35.0/internal/socket/rawconn_msg.go:27 +0x144 fp=0xc0003f2e18 sp=0xc0003f2dc0 pc=0xb32404
golang.org/x/net/internal/socket.(*Conn).RecvMsg(...)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/x/net@v0.35.0/internal/socket/socket.go:247
golang.org/x/net/ipv4.(*payloadHandler).ReadFrom(0xc0004e8a60, {0xc00082a000, 0x10000, 0x10000})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/x/net@v0.35.0/ipv4/payload_cmsg.go:31 +0x4ae fp=0xc0003f2eb8 sp=0xc0003f2e18 pc=0xb3c32e
github.com/libp2p/zeroconf/v2.(*client).recv.func2({0xc00082a000?, 0xc0003f2f18?, 0x35a?})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/zeroconf/v2@v2.2.0/client.go:328 +0x2e fp=0xc0003f2ee8 sp=0xc0003f2eb8 pc=0x10e066e
github.com/libp2p/zeroconf/v2.(*client).recv(0xc0004e8af0?, {0x1b34be0, 0xc0004e8af0}, {0x15a1260?, 0xc0004e8a50?}, 0xc00027bce0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/zeroconf/v2@v2.2.0/client.go:347 +0x14e fp=0xc0003f2fa0 sp=0xc0003f2ee8 pc=0x10e050e
github.com/libp2p/zeroconf/v2.(*client).mainloop.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/zeroconf/v2@v2.2.0/client.go:173 +0x34 fp=0xc0003f2fe0 sp=0xc0003f2fa0 pc=0x10e0334
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc0003f2fe8 sp=0xc0003f2fe0 pc=0x481ec1
created by github.com/libp2p/zeroconf/v2.(*client).mainloop in goroutine 195
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/zeroconf/v2@v2.2.0/client.go:173 +0x125

goroutine 197 gp=0xc0004f6fc0 m=nil [IO wait]:
runtime.gopark(0xc000940d20?, 0xc0003f1c98?, 0x60?, 0xd5?, 0x1b3fa08?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc0003f1c58 sp=0xc0003f1c38 pc=0x479f4e
runtime.netpollblock(0x1b26000?, 0x258ed70?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/netpoll.go:575 +0xf7 fp=0xc0003f1c90 sp=0xc0003f1c58 pc=0x43ddd7
internal/poll.runtime_pollWait(0x7f0270667848, 0x72)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/netpoll.go:351 +0x85 fp=0xc0003f1cb0 sp=0xc0003f1c90 pc=0x479245
internal/poll.(*pollDesc).wait(0xc000169a00?, 0x50?, 0x0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_poll_runtime.go:84 +0x27 fp=0xc0003f1cd8 sp=0xc0003f1cb0 pc=0x50a5c7
internal/poll.(*pollDesc).waitRead(...)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_poll_runtime.go:89
internal/poll.(*FD).RawRead(0xc000169a00, 0xc000940d20)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_unix.go:717 +0x125 fp=0xc0003f1d38 sp=0xc0003f1cd8 pc=0x510605
net.(*rawConn).Read(0xc000090760, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/rawconn.go:44 +0x36 fp=0xc0003f1d70 sp=0xc0003f1d38 pc=0x632416
golang.org/x/net/internal/socket.(*Conn).recvMsg(0xc0004f2680, 0xc000795080, 0x0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/x/net@v0.35.0/internal/socket/rawconn_msg.go:27 +0x144 fp=0xc0003f1dc8 sp=0xc0003f1d70 pc=0xb32404
golang.org/x/net/internal/socket.(*Conn).RecvMsg(...)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/x/net@v0.35.0/internal/socket/socket.go:247
golang.org/x/net/ipv6.(*payloadHandler).ReadFrom(0xc0004e8ab0, {0xc00081a000, 0x10000, 0x10000})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/x/net@v0.35.0/ipv6/payload_cmsg.go:31 +0x38d fp=0xc0003f1eb8 sp=0xc0003f1dc8 pc=0xb43aad
github.com/libp2p/zeroconf/v2.(*client).recv.func1({0xc00081a000?, 0xc0003f1f18?, 0x35a?})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/zeroconf/v2@v2.2.0/client.go:323 +0x2e fp=0xc0003f1ee8 sp=0xc0003f1eb8 pc=0x10e06ee
github.com/libp2p/zeroconf/v2.(*client).recv(0x0?, {0x1b34be0, 0xc0004e8af0}, {0x15a49c0?, 0xc0004e8aa0?}, 0xc00027bce0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/zeroconf/v2@v2.2.0/client.go:347 +0x14e fp=0xc0003f1fa0 sp=0xc0003f1ee8 pc=0x10e050e
github.com/libp2p/zeroconf/v2.(*client).mainloop.gowrap2()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/zeroconf/v2@v2.2.0/client.go:176 +0x34 fp=0xc0003f1fe0 sp=0xc0003f1fa0 pc=0x10e02d4
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc0003f1fe8 sp=0xc0003f1fe0 pc=0x481ec1
created by github.com/libp2p/zeroconf/v2.(*client).mainloop in goroutine 195
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/zeroconf/v2@v2.2.0/client.go:176 +0x1f2

goroutine 233 gp=0xc0004f7340 m=nil [select]:
runtime.gopark(0xc0003b5660?, 0x2?, 0xf0?, 0x8?, 0xc0003b5634?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc0003b54d8 sp=0xc0003b54b8 pc=0x479f4e
runtime.selectgo(0xc0003b5660, 0xc0003b5630, 0xc0003b5601?, 0x0, 0x1000200000002?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc0003b5600 sp=0xc0003b54d8 pc=0x4573c5
github.com/libp2p/go-yamux/v4.(*Stream).Read(0xc0008f81c0, {0xc000920df0, 0x1, 0x1})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.2/stream.go:111 +0x1a5 fp=0xc0003b5690 sp=0xc0003b5600 pc=0xa64d45
github.com/libp2p/go-libp2p/p2p/muxer/yamux.(*stream).Read(0x0?, {0xc000920df0?, 0x5000000?, 0x3feef00000022?})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/muxer/yamux/stream.go:17 +0x18 fp=0xc0003b56d8 sp=0xc0003b5690 pc=0xa678d8
github.com/libp2p/go-libp2p/p2p/net/swarm.(*Stream).Read(0xc000786080, {0xc000920df0?, 0xa64eda?, 0xc000945f80?})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/swarm_stream.go:58 +0x2d fp=0xc0003b5740 sp=0xc0003b56d8 pc=0xbb0ecd
io.ReadAtLeast({0x7f0270446100, 0xc000786080}, {0xc000920df0, 0x1, 0x1}, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/io/io.go:335 +0x90 fp=0xc0003b5788 sp=0xc0003b5740 pc=0x4ba2d0
io.ReadFull(...)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/io/io.go:354
github.com/libp2p/go-msgio.(*simpleByteReader).ReadByte(0xc000920de0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-msgio@v0.3.0/varint.go:185 +0x31 fp=0xc0003b57c8 sp=0xc0003b5788 pc=0xe9d7b1
github.com/multiformats/go-varint.ReadUvarint({0x1b264a0, 0xc000920de0})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/multiformats/go-varint@v0.0.7/varint.go:80 +0x51 fp=0xc0003b5810 sp=0xc0003b57c8 pc=0x94b331
github.com/libp2p/go-msgio.(*varintReader).nextMsgLen(0xc00083ad00)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-msgio@v0.3.0/varint.go:119 +0x2a fp=0xc0003b5830 sp=0xc0003b5810 pc=0xe9d02a
github.com/libp2p/go-msgio.(*varintReader).ReadMsg(0xc00083ad00)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-msgio@v0.3.0/varint.go:149 +0xb2 fp=0xc0003b58d8 sp=0xc0003b5830 pc=0xe9d3d2
github.com/libp2p/go-libp2p-kad-dht.(*IpfsDHT).handleNewMessage(0xc0004b5508, {0x1b48fe0, 0xc000786080})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-kad-dht@v0.25.2/dht_net.go:52 +0x27a fp=0xc0003b5d08 sp=0xc0003b58d8 pc=0x10bf17a
github.com/libp2p/go-libp2p-kad-dht.(*IpfsDHT).handleNewStream(0xc000918a10?, {0x1b48fe0, 0xc000786080})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-kad-dht@v0.25.2/dht_net.go:26 +0x1d fp=0xc0003b5d30 sp=0xc0003b5d08 pc=0x10bee9d
github.com/libp2p/go-libp2p-kad-dht.(*IpfsDHT).handleNewStream-fm({0x1b48fe0?, 0xc000786080?})
	<autogenerated>:1 +0x33 fp=0xc0003b5d58 sp=0xc0003b5d30 pc=0x10d74b3
github.com/libp2p/go-libp2p/p2p/host/basic.(*BasicHost).SetStreamHandler.func1({0x256abe0?, 0x15633c0?}, {0x7f02704460d0?, 0xc000786080?})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/host/basic/basic_host.go:659 +0x82 fp=0xc0003b5d88 sp=0xc0003b5d58 pc=0xf73ee2
github.com/libp2p/go-libp2p/p2p/host/basic.(*BasicHost).newStreamHandler(0xc000394000, {0x1b48fe0, 0xc000786080})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/host/basic/basic_host.go:487 +0x7e9 fp=0xc0003b5f50 sp=0xc0003b5d88 pc=0xf72a89
github.com/libp2p/go-libp2p/p2p/host/basic.(*BasicHost).newStreamHandler-fm({0x1b48fe0?, 0xc000786080?})
	<autogenerated>:1 +0x33 fp=0xc0003b5f78 sp=0xc0003b5f50 pc=0xf7ae53
github.com/libp2p/go-libp2p/p2p/net/swarm.(*Conn).start.func1.1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/swarm_conn.go:142 +0xa7 fp=0xc0003b5fe0 sp=0xc0003b5f78 pc=0xba75e7
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc0003b5fe8 sp=0xc0003b5fe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p/p2p/net/swarm.(*Conn).start.func1 in goroutine 230
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/swarm_conn.go:128 +0x1ab

goroutine 232 gp=0xc0004f7880 m=nil [select]:
runtime.gopark(0xc0003b3660?, 0x2?, 0x45?, 0x0?, 0xc0003b3634?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc0003b34d8 sp=0xc0003b34b8 pc=0x479f4e
runtime.selectgo(0xc0003b3660, 0xc0003b3630, 0xba4401?, 0x0, 0x1000200000002?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc0003b3600 sp=0xc0003b34d8 pc=0x4573c5
github.com/libp2p/go-yamux/v4.(*Stream).Read(0xc0008f80e0, {0xc000920d60, 0x1, 0x1})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.2/stream.go:111 +0x1a5 fp=0xc0003b3690 sp=0xc0003b3600 pc=0xa64d45
github.com/libp2p/go-libp2p/p2p/muxer/yamux.(*stream).Read(0x0?, {0xc000920d60?, 0x6f000000?, 0x3ff6c00000002?})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/muxer/yamux/stream.go:17 +0x18 fp=0xc0003b36d8 sp=0xc0003b3690 pc=0xa678d8
github.com/libp2p/go-libp2p/p2p/net/swarm.(*Stream).Read(0xc000786180, {0xc000920d60?, 0xa64eda?, 0xc000a23f80?})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/swarm_stream.go:58 +0x2d fp=0xc0003b3740 sp=0xc0003b36d8 pc=0xbb0ecd
io.ReadAtLeast({0x7f0270446100, 0xc000786180}, {0xc000920d60, 0x1, 0x1}, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/io/io.go:335 +0x90 fp=0xc0003b3788 sp=0xc0003b3740 pc=0x4ba2d0
io.ReadFull(...)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/io/io.go:354
github.com/libp2p/go-msgio.(*simpleByteReader).ReadByte(0xc000920d50)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-msgio@v0.3.0/varint.go:185 +0x31 fp=0xc0003b37c8 sp=0xc0003b3788 pc=0xe9d7b1
github.com/multiformats/go-varint.ReadUvarint({0x1b264a0, 0xc000920d50})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/multiformats/go-varint@v0.0.7/varint.go:80 +0x51 fp=0xc0003b3810 sp=0xc0003b37c8 pc=0x94b331
github.com/libp2p/go-msgio.(*varintReader).nextMsgLen(0xc00083aa80)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-msgio@v0.3.0/varint.go:119 +0x2a fp=0xc0003b3830 sp=0xc0003b3810 pc=0xe9d02a
github.com/libp2p/go-msgio.(*varintReader).ReadMsg(0xc00083aa80)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-msgio@v0.3.0/varint.go:149 +0xb2 fp=0xc0003b38d8 sp=0xc0003b3830 pc=0xe9d3d2
github.com/libp2p/go-libp2p-kad-dht.(*IpfsDHT).handleNewMessage(0xc0004b4a88, {0x1b48fe0, 0xc000786180})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-kad-dht@v0.25.2/dht_net.go:52 +0x27a fp=0xc0003b3d08 sp=0xc0003b38d8 pc=0x10bf17a
github.com/libp2p/go-libp2p-kad-dht.(*IpfsDHT).handleNewStream(0xc0009189e0?, {0x1b48fe0, 0xc000786180})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-kad-dht@v0.25.2/dht_net.go:26 +0x1d fp=0xc0003b3d30 sp=0xc0003b3d08 pc=0x10bee9d
github.com/libp2p/go-libp2p-kad-dht.(*IpfsDHT).handleNewStream-fm({0x1b48fe0?, 0xc000786180?})
	<autogenerated>:1 +0x33 fp=0xc0003b3d58 sp=0xc0003b3d30 pc=0x10d74b3
github.com/libp2p/go-libp2p/p2p/host/basic.(*BasicHost).SetStreamHandler.func1({0x256abe0?, 0x15633c0?}, {0x7f02704460d0?, 0xc000786180?})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/host/basic/basic_host.go:659 +0x82 fp=0xc0003b3d88 sp=0xc0003b3d58 pc=0xf73ee2
github.com/libp2p/go-libp2p/p2p/host/basic.(*BasicHost).newStreamHandler(0xc000394000, {0x1b48fe0, 0xc000786180})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/host/basic/basic_host.go:487 +0x7e9 fp=0xc0003b3f50 sp=0xc0003b3d88 pc=0xf72a89
github.com/libp2p/go-libp2p/p2p/host/basic.(*BasicHost).newStreamHandler-fm({0x1b48fe0?, 0xc000786180?})
	<autogenerated>:1 +0x33 fp=0xc0003b3f78 sp=0xc0003b3f50 pc=0xf7ae53
github.com/libp2p/go-libp2p/p2p/net/swarm.(*Conn).start.func1.1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/swarm_conn.go:142 +0xa7 fp=0xc0003b3fe0 sp=0xc0003b3f78 pc=0xba75e7
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc0003b3fe8 sp=0xc0003b3fe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p/p2p/net/swarm.(*Conn).start.func1 in goroutine 230
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/swarm_conn.go:128 +0x1ab

goroutine 226 gp=0xc0004f7a40 m=nil [IO wait]:
runtime.gopark(0xc0008c2000?, 0x5?, 0x80?, 0xa?, 0xb?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000235880 sp=0xc000235860 pc=0x479f4e
runtime.netpollblock(0x49d3b8?, 0x40ff66?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/netpoll.go:575 +0xf7 fp=0xc0002358b8 sp=0xc000235880 pc=0x43ddd7
internal/poll.runtime_pollWait(0x7f02703c78f0, 0x72)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/netpoll.go:351 +0x85 fp=0xc0002358d8 sp=0xc0002358b8 pc=0x479245
internal/poll.(*pollDesc).wait(0xc00083c580?, 0xc0008c2000?, 0x0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_poll_runtime.go:84 +0x27 fp=0xc000235900 sp=0xc0002358d8 pc=0x50a5c7
internal/poll.(*pollDesc).waitRead(...)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_poll_runtime.go:89
internal/poll.(*FD).Read(0xc00083c580, {0xc0008c2000, 0xa80, 0xa80})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_unix.go:165 +0x27a fp=0xc000235998 sp=0xc000235900 pc=0x50b8ba
net.(*netFD).Read(0xc00083c580, {0xc0008c2000?, 0xc0008be140?, 0xc000235b20?})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/fd_posix.go:55 +0x25 fp=0xc0002359e0 sp=0xc000235998 pc=0x61be05
net.(*conn).Read(0xc000916000, {0xc0008c2000?, 0xc0008be140?, 0xc0008c2005?})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/net.go:189 +0x45 fp=0xc000235a28 sp=0xc0002359e0 pc=0x62dc45
go:(*struct { *net.TCPConn; github.com/multiformats/go-multiaddr/net.maEndpoints }).Read(0x8521c8?, {0xc0008c2000?, 0x18?, 0x10000000000?})
	<autogenerated>:1 +0x26 fp=0xc000235a58 sp=0xc000235a28 pc=0x9f03a6
crypto/tls.(*atLeastReader).Read(0xc000768c78, {0xc0008c2000?, 0x0?, 0xc000768c78?})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/crypto/tls/conn.go:809 +0x3b fp=0xc000235aa0 sp=0xc000235a58 pc=0x69d43b
bytes.(*Buffer).ReadFrom(0xc0008522b8, {0x1b271a0, 0xc000768c78})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/bytes/buffer.go:211 +0x98 fp=0xc000235af8 sp=0xc000235aa0 pc=0x53bcb8
crypto/tls.(*Conn).readFromUntil(0xc000852008, {0x7f02703060b8, 0xc0009140c0}, 0xc000235b90?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/crypto/tls/conn.go:831 +0xde fp=0xc000235b30 sp=0xc000235af8 pc=0x69d61e
crypto/tls.(*Conn).readRecordOrCCS(0xc000852008, 0x0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/crypto/tls/conn.go:629 +0x3cf fp=0xc000235da8 sp=0xc000235b30 pc=0x69a72f
crypto/tls.(*Conn).readRecord(...)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/crypto/tls/conn.go:591
crypto/tls.(*Conn).Read(0xc000852008, {0xc0008a0470, 0xc, 0xc0008e6080?})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/crypto/tls/conn.go:1385 +0x150 fp=0xc000235e18 sp=0xc000235da8 pc=0x6a0f90
github.com/libp2p/go-libp2p/p2p/security/tls.(*conn).Read(0xc0008e6050?, {0xc0008a0470?, 0xc000235e80?, 0xa615b2?})
	<autogenerated>:1 +0x25 fp=0xc000235e48 sp=0xc000235e18 pc=0xbdede5
io.ReadAtLeast({0x7f0270446020, 0xc0008bc770}, {0xc0008a0470, 0xc, 0xc}, 0xc)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/io/io.go:335 +0x90 fp=0xc000235e90 sp=0xc000235e48 pc=0x4ba2d0
io.ReadFull(...)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/io/io.go:354
github.com/libp2p/go-yamux/v4.(*Session).recvLoop(0xc0008e6000)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.2/session.go:674 +0xe5 fp=0xc000235fa0 sp=0xc000235e90 pc=0xa62725
github.com/libp2p/go-yamux/v4.(*Session).recv(0xc0008e6000)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.2/session.go:646 +0x18 fp=0xc000235fc8 sp=0xc000235fa0 pc=0xa625f8
github.com/libp2p/go-yamux/v4.newSession.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.2/session.go:162 +0x25 fp=0xc000235fe0 sp=0xc000235fc8 pc=0xa5f525
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc000235fe8 sp=0xc000235fe0 pc=0x481ec1
created by github.com/libp2p/go-yamux/v4.newSession in goroutine 200
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.2/session.go:162 +0x516

goroutine 190 gp=0xc000904380 m=nil [select]:
runtime.gopark(0xc000882ee8?, 0x4?, 0x20?, 0x2d?, 0xc000882dfc?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000882c38 sp=0xc000882c18 pc=0x479f4e
runtime.selectgo(0xc000882ee8, 0xc000882df4, 0xc?, 0x0, 0x0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc000882d60 sp=0xc000882c38 pc=0x4573c5
github.com/libp2p/go-yamux/v4.(*Session).sendLoop(0xc000688000)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.2/session.go:589 +0x587 fp=0xc000882fa0 sp=0xc000882d60 pc=0xa61f07
github.com/libp2p/go-yamux/v4.(*Session).send(0xc000688000)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.2/session.go:518 +0x18 fp=0xc000882fc8 sp=0xc000882fa0 pc=0xa61938
github.com/libp2p/go-yamux/v4.newSession.gowrap2()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.2/session.go:163 +0x25 fp=0xc000882fe0 sp=0xc000882fc8 pc=0xa5f4c5
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc000882fe8 sp=0xc000882fe0 pc=0x481ec1
created by github.com/libp2p/go-yamux/v4.newSession in goroutine 214
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.2/session.go:163 +0x556

goroutine 296 gp=0xc0004f7c00 m=nil [IO wait]:
runtime.gopark(0x0?, 0x0?, 0x0?, 0x0?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc0003edb70 sp=0xc0003edb50 pc=0x479f4e
runtime.netpollblock(0xc0003edbe0?, 0x40ff66?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/netpoll.go:575 +0xf7 fp=0xc0003edba8 sp=0xc0003edb70 pc=0x43ddd7
internal/poll.runtime_pollWait(0x7f02703c75a8, 0x72)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/netpoll.go:351 +0x85 fp=0xc0003edbc8 sp=0xc0003edba8 pc=0x479245
internal/poll.(*pollDesc).wait(0xc0008c5400?, 0x10?, 0x0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_poll_runtime.go:84 +0x27 fp=0xc0003edbf0 sp=0xc0003edbc8 pc=0x50a5c7
internal/poll.(*pollDesc).waitRead(...)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_poll_runtime.go:89
internal/poll.(*FD).Accept(0xc0008c5400)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_unix.go:620 +0x295 fp=0xc0003edc98 sp=0xc0003edbf0 pc=0x50f995
net.(*netFD).accept(0xc0008c5400)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/fd_unix.go:172 +0x29 fp=0xc0003edd50 sp=0xc0003edc98 pc=0x61ddc9
net.(*TCPListener).accept(0xc0008d5b80)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/tcpsock_posix.go:159 +0x1e fp=0xc0003edda0 sp=0xc0003edd50 pc=0x637d3e
net.(*TCPListener).Accept(0xc0008d5b80)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/tcpsock.go:372 +0x30 fp=0xc0003eddd0 sp=0xc0003edda0 pc=0x636bf0
net/http.(*onceCloseListener).Accept(0x1b34b70?)
	<autogenerated>:1 +0x24 fp=0xc0003edde8 sp=0xc0003eddd0 pc=0x794584
net/http.(*Server).Serve(0xc0004b23c0, {0x1b30da8, 0xc0008d5b80})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/http/server.go:3330 +0x30c fp=0xc0003edf18 sp=0xc0003edde8 pc=0x76c24c
net/http.(*Server).ListenAndServe(0xc0004b23c0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/http/server.go:3259 +0x71 fp=0xc0003edf48 sp=0xc0003edf18 pc=0x76bf11
main.(*PaymentProcessor).Start.func2()
	/home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP_PROTO/payment_processor.go:80 +0xbb fp=0xc0003edfe0 sp=0xc0003edf48 pc=0x123441b
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc0003edfe8 sp=0xc0003edfe0 pc=0x481ec1
created by main.(*PaymentProcessor).Start in goroutine 1
	/home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP_PROTO/payment_processor.go:78 +0x1be

goroutine 189 gp=0xc0009048c0 m=nil [IO wait]:
runtime.gopark(0xc0008fa000?, 0x5?, 0x0?, 0x8?, 0xb?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc0006e5880 sp=0xc0006e5860 pc=0x479f4e
runtime.netpollblock(0x49d3b8?, 0x40ff66?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/netpoll.go:575 +0xf7 fp=0xc0006e58b8 sp=0xc0006e5880 pc=0x43ddd7
internal/poll.runtime_pollWait(0x7f02703c76c0, 0x72)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/netpoll.go:351 +0x85 fp=0xc0006e58d8 sp=0xc0006e58b8 pc=0x479245
internal/poll.(*pollDesc).wait(0xc000944980?, 0xc0008fa000?, 0x0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_poll_runtime.go:84 +0x27 fp=0xc0006e5900 sp=0xc0006e58d8 pc=0x50a5c7
internal/poll.(*pollDesc).waitRead(...)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_poll_runtime.go:89
internal/poll.(*FD).Read(0xc000944980, {0xc0008fa000, 0x800, 0x800})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_unix.go:165 +0x27a fp=0xc0006e5998 sp=0xc0006e5900 pc=0x50b8ba
net.(*netFD).Read(0xc000944980, {0xc0008fa000?, 0xc0003ce220?, 0xc0006e5b20?})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/fd_posix.go:55 +0x25 fp=0xc0006e59e0 sp=0xc0006e5998 pc=0x61be05
net.(*conn).Read(0xc000916010, {0xc0008fa000?, 0xc0003ce220?, 0xc0008fa005?})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/net.go:189 +0x45 fp=0xc0006e5a28 sp=0xc0006e59e0 pc=0x62dc45
go:(*struct { *net.TCPConn; github.com/multiformats/go-multiaddr/net.maEndpoints }).Read(0x94e548?, {0xc0008fa000?, 0x18?, 0x26474a0?})
	<autogenerated>:1 +0x26 fp=0xc0006e5a58 sp=0xc0006e5a28 pc=0x9f03a6
crypto/tls.(*atLeastReader).Read(0xc000768c60, {0xc0008fa000?, 0x0?, 0xc000768c60?})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/crypto/tls/conn.go:809 +0x3b fp=0xc0006e5aa0 sp=0xc0006e5a58 pc=0x69d43b
bytes.(*Buffer).ReadFrom(0xc00094e638, {0x1b271a0, 0xc000768c60})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/bytes/buffer.go:211 +0x98 fp=0xc0006e5af8 sp=0xc0006e5aa0 pc=0x53bcb8
crypto/tls.(*Conn).readFromUntil(0xc00094e388, {0x7f02703060b8, 0xc000914bd0}, 0xc0006e5b90?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/crypto/tls/conn.go:831 +0xde fp=0xc0006e5b30 sp=0xc0006e5af8 pc=0x69d61e
crypto/tls.(*Conn).readRecordOrCCS(0xc00094e388, 0x0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/crypto/tls/conn.go:629 +0x3cf fp=0xc0006e5da8 sp=0xc0006e5b30 pc=0x69a72f
crypto/tls.(*Conn).readRecord(...)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/crypto/tls/conn.go:591
crypto/tls.(*Conn).Read(0xc00094e388, {0xc0003ac290, 0xc, 0xc000688080?})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/crypto/tls/conn.go:1385 +0x150 fp=0xc0006e5e18 sp=0xc0006e5da8 pc=0x6a0f90
github.com/libp2p/go-libp2p/p2p/security/tls.(*conn).Read(0xc000688050?, {0xc0003ac290?, 0xc0006e5e80?, 0xa615b2?})
	<autogenerated>:1 +0x25 fp=0xc0006e5e48 sp=0xc0006e5e18 pc=0xbdede5
io.ReadAtLeast({0x7f0270446020, 0xc000358850}, {0xc0003ac290, 0xc, 0xc}, 0xc)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/io/io.go:335 +0x90 fp=0xc0006e5e90 sp=0xc0006e5e48 pc=0x4ba2d0
io.ReadFull(...)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/io/io.go:354
github.com/libp2p/go-yamux/v4.(*Session).recvLoop(0xc000688000)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.2/session.go:674 +0xe5 fp=0xc0006e5fa0 sp=0xc0006e5e90 pc=0xa62725
github.com/libp2p/go-yamux/v4.(*Session).recv(0xc000688000)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.2/session.go:646 +0x18 fp=0xc0006e5fc8 sp=0xc0006e5fa0 pc=0xa625f8
github.com/libp2p/go-yamux/v4.newSession.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.2/session.go:162 +0x25 fp=0xc0006e5fe0 sp=0xc0006e5fc8 pc=0xa5f525
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc0006e5fe8 sp=0xc0006e5fe0 pc=0x481ec1
created by github.com/libp2p/go-yamux/v4.newSession in goroutine 214
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.2/session.go:162 +0x516

goroutine 191 gp=0xc000904c40 m=nil [select]:
runtime.gopark(0xc00092e790?, 0x2?, 0x70?, 0x0?, 0xc00092e774?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00092e610 sp=0xc00092e5f0 pc=0x479f4e
runtime.selectgo(0xc00092e790, 0xc00092e770, 0xc00092e770?, 0x0, 0x0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc00092e738 sp=0xc00092e610 pc=0x4573c5
github.com/libp2p/go-yamux/v4.(*Session).startMeasureRTT(0xc000688000)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.2/session.go:356 +0xc5 fp=0xc00092e7c8 sp=0xc00092e738 pc=0xa60725
github.com/libp2p/go-yamux/v4.newSession.gowrap3()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.2/session.go:164 +0x25 fp=0xc00092e7e0 sp=0xc00092e7c8 pc=0xa5f465
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00092e7e8 sp=0xc00092e7e0 pc=0x481ec1
created by github.com/libp2p/go-yamux/v4.newSession in goroutine 214
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.2/session.go:164 +0x596

goroutine 227 gp=0xc00088c380 m=nil [select]:
runtime.gopark(0xc000935ee8?, 0x4?, 0x20?, 0x5d?, 0xc000935dfc?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000935c38 sp=0xc000935c18 pc=0x479f4e
runtime.selectgo(0xc000935ee8, 0xc000935df4, 0xc?, 0x0, 0x0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc000935d60 sp=0xc000935c38 pc=0x4573c5
github.com/libp2p/go-yamux/v4.(*Session).sendLoop(0xc0008e6000)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.2/session.go:589 +0x587 fp=0xc000935fa0 sp=0xc000935d60 pc=0xa61f07
github.com/libp2p/go-yamux/v4.(*Session).send(0xc0008e6000)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.2/session.go:518 +0x18 fp=0xc000935fc8 sp=0xc000935fa0 pc=0xa61938
github.com/libp2p/go-yamux/v4.newSession.gowrap2()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.2/session.go:163 +0x25 fp=0xc000935fe0 sp=0xc000935fc8 pc=0xa5f4c5
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc000935fe8 sp=0xc000935fe0 pc=0x481ec1
created by github.com/libp2p/go-yamux/v4.newSession in goroutine 200
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.2/session.go:163 +0x556

goroutine 228 gp=0xc00088c540 m=nil [select]:
runtime.gopark(0xc000933f90?, 0x2?, 0x70?, 0x0?, 0xc000933f74?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000933e10 sp=0xc000933df0 pc=0x479f4e
runtime.selectgo(0xc000933f90, 0xc000933f70, 0x0?, 0x0, 0x0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc000933f38 sp=0xc000933e10 pc=0x4573c5
github.com/libp2p/go-yamux/v4.(*Session).startMeasureRTT(0xc0008e6000)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.2/session.go:356 +0xc5 fp=0xc000933fc8 sp=0xc000933f38 pc=0xa60725
github.com/libp2p/go-yamux/v4.newSession.gowrap3()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.2/session.go:164 +0x25 fp=0xc000933fe0 sp=0xc000933fc8 pc=0xa5f465
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc000933fe8 sp=0xc000933fe0 pc=0x481ec1
created by github.com/libp2p/go-yamux/v4.newSession in goroutine 200
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.2/session.go:164 +0x596

goroutine 246 gp=0xc00088c700 m=nil [select]:
runtime.gopark(0xc00039a760?, 0x5?, 0xb8?, 0xf3?, 0xc00039a6d6?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00039a570 sp=0xc00039a550 pc=0x479f4e
runtime.selectgo(0xc00039a760, 0xc00039a6cc, 0x15e31df?, 0x0, 0xc0008ea900?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc00039a698 sp=0xc00039a570 pc=0x4573c5
main.(*SystrayManager).onReady.func1()
	/home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP_PROTO/systray.go:119 +0xe7 fp=0xc00039a7e0 sp=0xc00039a698 pc=0x1243547
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00039a7e8 sp=0xc00039a7e0 pc=0x481ec1
created by main.(*SystrayManager).onReady in goroutine 340
	/home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP_PROTO/systray.go:117 +0x2da

goroutine 230 gp=0xc00088c8c0 m=nil [select]:
runtime.gopark(0xc000185e98?, 0x2?, 0xa?, 0x0?, 0xc000185e74?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000185d18 sp=0xc000185cf8 pc=0x479f4e
runtime.selectgo(0xc000185e98, 0xc000185e70, 0x1?, 0x0, 0xa3eb99?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc000185e40 sp=0xc000185d18 pc=0x4573c5
github.com/libp2p/go-yamux/v4.(*Session).AcceptStream(0xc0008e6000)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.2/session.go:272 +0x106 fp=0xc000185ef8 sp=0xc000185e40 pc=0xa5ffa6
github.com/libp2p/go-libp2p/p2p/muxer/yamux.(*conn).AcceptStream(0x44efa0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/muxer/yamux/conn.go:43 +0x13 fp=0xc000185f10 sp=0xc000185ef8 pc=0xa67893
github.com/libp2p/go-libp2p/p2p/net/upgrader.(*transportConn).AcceptStream(0xba74eb?)
	<autogenerated>:1 +0x24 fp=0xc000185f28 sp=0xc000185f10 pc=0xbd6524
github.com/libp2p/go-libp2p/p2p/net/swarm.(*connWithMetrics).AcceptStream(0xc000000000?)
	<autogenerated>:1 +0x24 fp=0xc000185f40 sp=0xc000185f28 pc=0xbb6b24
github.com/libp2p/go-libp2p/p2p/net/swarm.(*Conn).start.func1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/swarm_conn.go:118 +0x96 fp=0xc000185fe0 sp=0xc000185f40 pc=0xba73d6
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc000185fe8 sp=0xc000185fe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p/p2p/net/swarm.(*Conn).start in goroutine 199
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/swarm_conn.go:114 +0x4f

goroutine 242 gp=0xc0006b7340 m=nil [select]:
runtime.gopark(0xc000881e98?, 0x2?, 0xa?, 0x0?, 0xc000881e74?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000881d18 sp=0xc000881cf8 pc=0x479f4e
runtime.selectgo(0xc000881e98, 0xc000881e70, 0x1?, 0x0, 0xa3eb99?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc000881e40 sp=0xc000881d18 pc=0x4573c5
github.com/libp2p/go-yamux/v4.(*Session).AcceptStream(0xc000688000)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.2/session.go:272 +0x106 fp=0xc000881ef8 sp=0xc000881e40 pc=0xa5ffa6
github.com/libp2p/go-libp2p/p2p/muxer/yamux.(*conn).AcceptStream(0x44efa0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/muxer/yamux/conn.go:43 +0x13 fp=0xc000881f10 sp=0xc000881ef8 pc=0xa67893
github.com/libp2p/go-libp2p/p2p/net/upgrader.(*transportConn).AcceptStream(0xba74eb?)
	<autogenerated>:1 +0x24 fp=0xc000881f28 sp=0xc000881f10 pc=0xbd6524
github.com/libp2p/go-libp2p/p2p/net/swarm.(*connWithMetrics).AcceptStream(0xc000000000?)
	<autogenerated>:1 +0x24 fp=0xc000881f40 sp=0xc000881f28 pc=0xbb6b24
github.com/libp2p/go-libp2p/p2p/net/swarm.(*Conn).start.func1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/swarm_conn.go:118 +0x96 fp=0xc000881fe0 sp=0xc000881f40 pc=0xba73d6
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc000881fe8 sp=0xc000881fe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p/p2p/net/swarm.(*Conn).start in goroutine 192
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/swarm_conn.go:114 +0x4f

goroutine 266 gp=0xc000286fc0 m=nil [select]:
runtime.gopark(0xc00030d908?, 0x2?, 0x0?, 0x0?, 0xc00030d8dc?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00030d780 sp=0xc00030d760 pc=0x479f4e
runtime.selectgo(0xc00030d908, 0xc00030d8d8, 0xc00076df80?, 0x0, 0x10?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc00030d8a8 sp=0xc00030d780 pc=0x4573c5
github.com/libp2p/go-yamux/v4.(*Stream).Read(0xc0001fc540, {0xc000768ac0, 0x1, 0x1})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.2/stream.go:111 +0x1a5 fp=0xc00030d938 sp=0xc00030d8a8 pc=0xa64d45
github.com/libp2p/go-libp2p/p2p/muxer/yamux.(*stream).Read(0x47cc6b?, {0xc000768ac0?, 0xc000286fc0?, 0xc00030d9c0?})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/muxer/yamux/stream.go:17 +0x18 fp=0xc00030d980 sp=0xc00030d938 pc=0xa678d8
github.com/libp2p/go-libp2p/p2p/net/swarm.(*Stream).Read(0xc0008c5480, {0xc000768ac0?, 0x47fef2?, 0xc00030da48?})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/swarm_stream.go:58 +0x2d fp=0xc00030d9e8 sp=0xc00030d980 pc=0xbb0ecd
io.ReadAtLeast({0x7f0270446100, 0xc0008c5480}, {0xc000768ac0, 0x1, 0x1}, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/io/io.go:335 +0x90 fp=0xc00030da30 sp=0xc00030d9e8 pc=0x4ba2d0
io.ReadFull(...)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/io/io.go:354
github.com/libp2p/go-msgio.(*simpleByteReader).ReadByte(0xc000768ab0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-msgio@v0.3.0/varint.go:185 +0x31 fp=0xc00030da70 sp=0xc00030da30 pc=0xe9d7b1
github.com/multiformats/go-varint.ReadUvarint({0x1b264a0, 0xc000768ab0})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/multiformats/go-varint@v0.0.7/varint.go:80 +0x51 fp=0xc00030dab8 sp=0xc00030da70 pc=0x94b331
github.com/libp2p/go-msgio.(*varintReader).nextMsgLen(0xc0008d5d00)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-msgio@v0.3.0/varint.go:119 +0x2a fp=0xc00030dad8 sp=0xc00030dab8 pc=0xe9d02a
github.com/libp2p/go-msgio.(*varintReader).ReadMsg(0xc0008d5d00)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-msgio@v0.3.0/varint.go:149 +0xb2 fp=0xc00030db80 sp=0xc00030dad8 pc=0xe9d3d2
github.com/libp2p/go-libp2p-pubsub.(*PubSub).handleNewStream(0xc0002ee248, {0x1b48fe0, 0xc0008c5480})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/comm.go:66 +0x3d7 fp=0xc00030dd30 sp=0xc00030db80 pc=0x11847b7
github.com/libp2p/go-libp2p-pubsub.(*PubSub).handleNewStream-fm({0x1b48fe0?, 0xc0008c5480?})
	<autogenerated>:1 +0x33 fp=0xc00030dd58 sp=0xc00030dd30 pc=0x11b74b3
github.com/libp2p/go-libp2p/p2p/host/basic.(*BasicHost).SetStreamHandler.func1({0x256abe0?, 0x15633c0?}, {0x7f02704460d0?, 0xc0008c5480?})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/host/basic/basic_host.go:659 +0x82 fp=0xc00030dd88 sp=0xc00030dd58 pc=0xf73ee2
github.com/libp2p/go-libp2p/p2p/host/basic.(*BasicHost).newStreamHandler(0xc000394000, {0x1b48fe0, 0xc0008c5480})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/host/basic/basic_host.go:487 +0x7e9 fp=0xc00030df50 sp=0xc00030dd88 pc=0xf72a89
github.com/libp2p/go-libp2p/p2p/host/basic.(*BasicHost).newStreamHandler-fm({0x1b48fe0?, 0xc0008c5480?})
	<autogenerated>:1 +0x33 fp=0xc00030df78 sp=0xc00030df50 pc=0xf7ae53
github.com/libp2p/go-libp2p/p2p/net/swarm.(*Conn).start.func1.1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/swarm_conn.go:142 +0xa7 fp=0xc00030dfe0 sp=0xc00030df78 pc=0xba75e7
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00030dfe8 sp=0xc00030dfe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p/p2p/net/swarm.(*Conn).start.func1 in goroutine 230
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/swarm_conn.go:128 +0x1ab

goroutine 208 gp=0xc00053b880 m=nil [select]:
runtime.gopark(0xc00008ff68?, 0x2?, 0x78?, 0x48?, 0xc00008ff44?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00008fde0 sp=0xc00008fdc0 pc=0x479f4e
runtime.selectgo(0xc00008ff68, 0xc00008ff40, 0xbd5f85?, 0x0, 0xc0004fd650?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc00008ff08 sp=0xc00008fde0 pc=0x4573c5
github.com/libp2p/go-libp2p/p2p/host/devstore/pstoremem.(*memoryAddrBook).background(0xc000944d80, {0x1b34be0, 0xc0008c04b0})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/host/devstore/pstoremem/addr_book.go:242 +0x114 fp=0xc00008ffb8 sp=0xc00008ff08 pc=0x9f8f14
github.com/libp2p/go-libp2p/p2p/host/devstore/pstoremem.NewAddrBook.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/host/devstore/pstoremem/addr_book.go:205 +0x28 fp=0xc00008ffe0 sp=0xc00008ffb8 pc=0x9f8dc8
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00008ffe8 sp=0xc00008ffe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p/p2p/host/devstore/pstoremem.NewAddrBook in goroutine 207
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/host/devstore/pstoremem/addr_book.go:205 +0x1c5

goroutine 209 gp=0xc00053ba40 m=nil [select]:
runtime.gopark(0xc00092b770?, 0x2?, 0x78?, 0x48?, 0xc00092b764?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00092b600 sp=0xc00092b5e0 pc=0x479f4e
runtime.selectgo(0xc00092b770, 0xc00092b760, 0x0?, 0x0, 0x0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc00092b728 sp=0xc00092b600 pc=0x4573c5
github.com/libp2p/go-libp2p-pubsub.(*backoff).cleanupLoop(0xc000790570, {0x1b34be0, 0xc0008c0460})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/backoff.go:99 +0xcd fp=0xc00092b7b8 sp=0xc00092b728 pc=0x118426d
github.com/libp2p/go-libp2p-pubsub.newBackoff.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/backoff.go:46 +0x28 fp=0xc00092b7e0 sp=0xc00092b7b8 pc=0x1183c08
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00092b7e8 sp=0xc00092b7e0 pc=0x481ec1
created by github.com/libp2p/go-libp2p-pubsub.newBackoff in goroutine 207
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/backoff.go:46 +0xdd

goroutine 274 gp=0xc00053bc00 m=nil [select]:
runtime.gopark(0xc00092bf60?, 0x2?, 0x78?, 0x48?, 0xc00092bf34?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00092bdd0 sp=0xc00092bdb0 pc=0x479f4e
runtime.selectgo(0xc00092bf60, 0xc00092bf30, 0x0?, 0x0, 0x0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc00092bef8 sp=0xc00092bdd0 pc=0x4573c5
github.com/libp2p/go-libp2p-pubsub/timecache.background({0x1b34be0, 0xc0008c05a0}, {0x1b2e8b0, 0xc000790780}, 0xc000790750)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/timecache/util.go:16 +0x13d fp=0xc00092bfa8 sp=0xc00092bef8 pc=0x11543fd
github.com/libp2p/go-libp2p-pubsub/timecache.newFirstSeenCache.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/timecache/first_seen_cache.go:28 +0x30 fp=0xc00092bfe0 sp=0xc00092bfa8 pc=0x11538b0
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00092bfe8 sp=0xc00092bfe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p-pubsub/timecache.newFirstSeenCache in goroutine 207
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/timecache/first_seen_cache.go:28 +0x125

goroutine 275 gp=0xc00053bdc0 m=nil [select]:
runtime.gopark(0xc00092c760?, 0x2?, 0xb0?, 0xa?, 0xc00092c714?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00092c5a8 sp=0xc00092c588 pc=0x479f4e
runtime.selectgo(0xc00092c760, 0xc00092c710, 0x0?, 0x0, 0x0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc00092c6d0 sp=0xc00092c5a8 pc=0x4573c5
github.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).heartbeatTimer(0xc0002ee008)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/gossipsub.go:1527 +0x1d8 fp=0xc00092c7c8 sp=0xc00092c6d0 pc=0x1192d18
github.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).Attach.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/gossipsub.go:551 +0x25 fp=0xc00092c7e0 sp=0xc00092c7c8 pc=0x11899e5
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00092c7e8 sp=0xc00092c7e0 pc=0x481ec1
created by github.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).Attach in goroutine 207
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/gossipsub.go:551 +0x1b6

goroutine 276 gp=0xc00088ca80 m=nil [select]:
runtime.gopark(0xc00092cf80?, 0x2?, 0x0?, 0x0?, 0xc00092ceb4?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00092cd38 sp=0xc00092cd18 pc=0x479f4e
runtime.selectgo(0xc00092cf80, 0xc00092ceb0, 0x0?, 0x0, 0x0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc00092ce60 sp=0xc00092cd38 pc=0x4573c5
github.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).connector(0xc0002ee008)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/gossipsub.go:1114 +0xc5 fp=0xc00092cfc8 sp=0xc00092ce60 pc=0x118e145
github.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).Attach.gowrap2()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/gossipsub.go:555 +0x25 fp=0xc00092cfe0 sp=0xc00092cfc8 pc=0x1189985
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00092cfe8 sp=0xc00092cfe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).Attach in goroutine 207
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/gossipsub.go:555 +0x1c5

goroutine 277 gp=0xc00088cc40 m=nil [select]:
runtime.gopark(0xc00092d780?, 0x2?, 0x0?, 0x0?, 0xc00092d6b4?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00092d538 sp=0xc00092d518 pc=0x479f4e
runtime.selectgo(0xc00092d780, 0xc00092d6b0, 0x0?, 0x0, 0x0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc00092d660 sp=0xc00092d538 pc=0x4573c5
github.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).connector(0xc0002ee008)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/gossipsub.go:1114 +0xc5 fp=0xc00092d7c8 sp=0xc00092d660 pc=0x118e145
github.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).Attach.gowrap2()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/gossipsub.go:555 +0x25 fp=0xc00092d7e0 sp=0xc00092d7c8 pc=0x1189985
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00092d7e8 sp=0xc00092d7e0 pc=0x481ec1
created by github.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).Attach in goroutine 207
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/gossipsub.go:555 +0x1c5

goroutine 278 gp=0xc00088ce00 m=nil [select]:
runtime.gopark(0xc00092df80?, 0x2?, 0x0?, 0x0?, 0xc00092deb4?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00092dd38 sp=0xc00092dd18 pc=0x479f4e
runtime.selectgo(0xc00092df80, 0xc00092deb0, 0x0?, 0x0, 0x0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc00092de60 sp=0xc00092dd38 pc=0x4573c5
github.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).connector(0xc0002ee008)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/gossipsub.go:1114 +0xc5 fp=0xc00092dfc8 sp=0xc00092de60 pc=0x118e145
github.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).Attach.gowrap2()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/gossipsub.go:555 +0x25 fp=0xc00092dfe0 sp=0xc00092dfc8 pc=0x1189985
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00092dfe8 sp=0xc00092dfe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).Attach in goroutine 207
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/gossipsub.go:555 +0x1c5

goroutine 279 gp=0xc00088cfc0 m=nil [select]:
runtime.gopark(0xc00092f780?, 0x2?, 0x0?, 0x0?, 0xc00092f6b4?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00092f538 sp=0xc00092f518 pc=0x479f4e
runtime.selectgo(0xc00092f780, 0xc00092f6b0, 0x0?, 0x0, 0x0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc00092f660 sp=0xc00092f538 pc=0x4573c5
github.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).connector(0xc0002ee008)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/gossipsub.go:1114 +0xc5 fp=0xc00092f7c8 sp=0xc00092f660 pc=0x118e145
github.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).Attach.gowrap2()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/gossipsub.go:555 +0x25 fp=0xc00092f7e0 sp=0xc00092f7c8 pc=0x1189985
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00092f7e8 sp=0xc00092f7e0 pc=0x481ec1
created by github.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).Attach in goroutine 207
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/gossipsub.go:555 +0x1c5

goroutine 280 gp=0xc00088d180 m=nil [select]:
runtime.gopark(0xc00092ff80?, 0x2?, 0x0?, 0x0?, 0xc00092feb4?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00092fd38 sp=0xc00092fd18 pc=0x479f4e
runtime.selectgo(0xc00092ff80, 0xc00092feb0, 0x0?, 0x0, 0x0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc00092fe60 sp=0xc00092fd38 pc=0x4573c5
github.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).connector(0xc0002ee008)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/gossipsub.go:1114 +0xc5 fp=0xc00092ffc8 sp=0xc00092fe60 pc=0x118e145
github.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).Attach.gowrap2()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/gossipsub.go:555 +0x25 fp=0xc00092ffe0 sp=0xc00092ffc8 pc=0x1189985
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00092ffe8 sp=0xc00092ffe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).Attach in goroutine 207
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/gossipsub.go:555 +0x1c5

goroutine 281 gp=0xc00088d340 m=nil [select]:
runtime.gopark(0xc00087d780?, 0x2?, 0x0?, 0x0?, 0xc00087d6b4?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00087d538 sp=0xc00087d518 pc=0x479f4e
runtime.selectgo(0xc00087d780, 0xc00087d6b0, 0x0?, 0x0, 0x0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc00087d660 sp=0xc00087d538 pc=0x4573c5
github.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).connector(0xc0002ee008)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/gossipsub.go:1114 +0xc5 fp=0xc00087d7c8 sp=0xc00087d660 pc=0x118e145
github.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).Attach.gowrap2()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/gossipsub.go:555 +0x25 fp=0xc00087d7e0 sp=0xc00087d7c8 pc=0x1189985
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00087d7e8 sp=0xc00087d7e0 pc=0x481ec1
created by github.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).Attach in goroutine 207
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/gossipsub.go:555 +0x1c5

goroutine 282 gp=0xc00088d500 m=nil [select]:
runtime.gopark(0xc00087df80?, 0x2?, 0x0?, 0x0?, 0xc00087deb4?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00087dd38 sp=0xc00087dd18 pc=0x479f4e
runtime.selectgo(0xc00087df80, 0xc00087deb0, 0x0?, 0x0, 0x0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc00087de60 sp=0xc00087dd38 pc=0x4573c5
github.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).connector(0xc0002ee008)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/gossipsub.go:1114 +0xc5 fp=0xc00087dfc8 sp=0xc00087de60 pc=0x118e145
github.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).Attach.gowrap2()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/gossipsub.go:555 +0x25 fp=0xc00087dfe0 sp=0xc00087dfc8 pc=0x1189985
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00087dfe8 sp=0xc00087dfe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).Attach in goroutine 207
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/gossipsub.go:555 +0x1c5

goroutine 283 gp=0xc00088d6c0 m=nil [select]:
runtime.gopark(0xc000928780?, 0x2?, 0x0?, 0x0?, 0xc0009286b4?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000928538 sp=0xc000928518 pc=0x479f4e
runtime.selectgo(0xc000928780, 0xc0009286b0, 0x0?, 0x0, 0x0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc000928660 sp=0xc000928538 pc=0x4573c5
github.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).connector(0xc0002ee008)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/gossipsub.go:1114 +0xc5 fp=0xc0009287c8 sp=0xc000928660 pc=0x118e145
github.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).Attach.gowrap2()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/gossipsub.go:555 +0x25 fp=0xc0009287e0 sp=0xc0009287c8 pc=0x1189985
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc0009287e8 sp=0xc0009287e0 pc=0x481ec1
created by github.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).Attach in goroutine 207
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/gossipsub.go:555 +0x1c5

goroutine 284 gp=0xc00088d880 m=nil [select]:
runtime.gopark(0xc0005cbf38?, 0x2?, 0x60?, 0x11?, 0xc0005cbe5c?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc0005cbcd8 sp=0xc0005cbcb8 pc=0x479f4e
runtime.selectgo(0xc0005cbf38, 0xc0005cbe58, 0x7ffffffffffffffe?, 0x0, 0x0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc0005cbe00 sp=0xc0005cbcd8 pc=0x4573c5
github.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).manageAddrBook(0xc0002ee008)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/gossipsub.go:586 +0x269 fp=0xc0005cbfc8 sp=0xc0005cbe00 pc=0x1189c89
github.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).Attach.gowrap3()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/gossipsub.go:559 +0x25 fp=0xc0005cbfe0 sp=0xc0005cbfc8 pc=0x1189925
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc0005cbfe8 sp=0xc0005cbfe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p-pubsub.(*GossipSubRouter).Attach in goroutine 207
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/gossipsub.go:559 +0x256

goroutine 285 gp=0xc00088da40 m=nil [select]:
runtime.gopark(0xc000a37e28?, 0x2?, 0x40?, 0xda?, 0xc000a37d34?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000a37bb0 sp=0xc000a37b90 pc=0x479f4e
runtime.selectgo(0xc000a37e28, 0xc000a37d30, 0x15d21a0?, 0x0, 0x0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc000a37cd8 sp=0xc000a37bb0 pc=0x4573c5
github.com/libp2p/go-libp2p-pubsub.(*PubSub).watchForNewPeers(0xc0002ee248, {0x1b34be0, 0xc0008c0460})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/dev_notify.go:69 +0x55e fp=0xc000a37fb8 sp=0xc000a37cd8 pc=0x119b69e
github.com/libp2p/go-libp2p-pubsub.NewPubSub.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/pubsub.go:333 +0x28 fp=0xc000a37fe0 sp=0xc000a37fb8 pc=0x119cc88
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc000a37fe8 sp=0xc000a37fe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p-pubsub.NewPubSub in goroutine 207
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/pubsub.go:333 +0xddf

goroutine 286 gp=0xc00088dc00 m=nil [select]:
runtime.gopark(0xc000929f88?, 0x2?, 0x0?, 0x0?, 0xc000929f84?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000929e18 sp=0xc000929df8 pc=0x479f4e
runtime.selectgo(0xc000929f88, 0xc000929f80, 0x0?, 0x0, 0x0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc000929f40 sp=0xc000929e18 pc=0x4573c5
github.com/libp2p/go-libp2p-pubsub.(*validation).validateWorker(0xc0008c0500)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/validation.go:283 +0xbf fp=0xc000929fc8 sp=0xc000929f40 pc=0x11b2aff
github.com/libp2p/go-libp2p-pubsub.(*validation).Start.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/validation.go:136 +0x25 fp=0xc000929fe0 sp=0xc000929fc8 pc=0x11b1ba5
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc000929fe8 sp=0xc000929fe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p-pubsub.(*validation).Start in goroutine 207
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/validation.go:136 +0x66

goroutine 287 gp=0xc00088ddc0 m=nil [select]:
runtime.gopark(0xc00092a788?, 0x2?, 0x0?, 0x0?, 0xc00092a784?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00092a618 sp=0xc00092a5f8 pc=0x479f4e
runtime.selectgo(0xc00092a788, 0xc00092a780, 0x1b46780?, 0x0, 0xa62460?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc00092a740 sp=0xc00092a618 pc=0x4573c5
github.com/libp2p/go-libp2p-pubsub.(*validation).validateWorker(0xc0008c0500)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/validation.go:283 +0xbf fp=0xc00092a7c8 sp=0xc00092a740 pc=0x11b2aff
github.com/libp2p/go-libp2p-pubsub.(*validation).Start.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/validation.go:136 +0x25 fp=0xc00092a7e0 sp=0xc00092a7c8 pc=0x11b1ba5
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00092a7e8 sp=0xc00092a7e0 pc=0x481ec1
created by github.com/libp2p/go-libp2p-pubsub.(*validation).Start in goroutine 207
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/validation.go:136 +0x66

goroutine 288 gp=0xc000904fc0 m=nil [select]:
runtime.gopark(0xc00092af88?, 0x2?, 0x10?, 0xca?, 0xc00092af84?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00092ae18 sp=0xc00092adf8 pc=0x479f4e
runtime.selectgo(0xc00092af88, 0xc00092af80, 0x0?, 0x0, 0x0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc00092af40 sp=0xc00092ae18 pc=0x4573c5
github.com/libp2p/go-libp2p-pubsub.(*validation).validateWorker(0xc0008c0500)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/validation.go:283 +0xbf fp=0xc00092afc8 sp=0xc00092af40 pc=0x11b2aff
github.com/libp2p/go-libp2p-pubsub.(*validation).Start.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/validation.go:136 +0x25 fp=0xc00092afe0 sp=0xc00092afc8 pc=0x11b1ba5
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00092afe8 sp=0xc00092afe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p-pubsub.(*validation).Start in goroutine 207
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/validation.go:136 +0x66

goroutine 289 gp=0xc000905180 m=nil [select]:
runtime.gopark(0xc000879788?, 0x2?, 0x0?, 0x0?, 0xc000879784?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000879618 sp=0xc0008795f8 pc=0x479f4e
runtime.selectgo(0xc000879788, 0xc000879780, 0x0?, 0x0, 0x0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc000879740 sp=0xc000879618 pc=0x4573c5
github.com/libp2p/go-libp2p-pubsub.(*validation).validateWorker(0xc0008c0500)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/validation.go:283 +0xbf fp=0xc0008797c8 sp=0xc000879740 pc=0x11b2aff
github.com/libp2p/go-libp2p-pubsub.(*validation).Start.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/validation.go:136 +0x25 fp=0xc0008797e0 sp=0xc0008797c8 pc=0x11b1ba5
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc0008797e8 sp=0xc0008797e0 pc=0x481ec1
created by github.com/libp2p/go-libp2p-pubsub.(*validation).Start in goroutine 207
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/validation.go:136 +0x66

goroutine 290 gp=0xc000905340 m=nil [select]:
runtime.gopark(0xc000879f88?, 0x2?, 0x0?, 0x0?, 0xc000879f84?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000879e18 sp=0xc000879df8 pc=0x479f4e
runtime.selectgo(0xc000879f88, 0xc000879f80, 0x0?, 0x0, 0x0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc000879f40 sp=0xc000879e18 pc=0x4573c5
github.com/libp2p/go-libp2p-pubsub.(*validation).validateWorker(0xc0008c0500)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/validation.go:283 +0xbf fp=0xc000879fc8 sp=0xc000879f40 pc=0x11b2aff
github.com/libp2p/go-libp2p-pubsub.(*validation).Start.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/validation.go:136 +0x25 fp=0xc000879fe0 sp=0xc000879fc8 pc=0x11b1ba5
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc000879fe8 sp=0xc000879fe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p-pubsub.(*validation).Start in goroutine 207
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/validation.go:136 +0x66

goroutine 291 gp=0xc000905500 m=nil [select]:
runtime.gopark(0xc00087a788?, 0x2?, 0x0?, 0x0?, 0xc00087a784?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00087a618 sp=0xc00087a5f8 pc=0x479f4e
runtime.selectgo(0xc00087a788, 0xc00087a780, 0x0?, 0x0, 0x0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc00087a740 sp=0xc00087a618 pc=0x4573c5
github.com/libp2p/go-libp2p-pubsub.(*validation).validateWorker(0xc0008c0500)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/validation.go:283 +0xbf fp=0xc00087a7c8 sp=0xc00087a740 pc=0x11b2aff
github.com/libp2p/go-libp2p-pubsub.(*validation).Start.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/validation.go:136 +0x25 fp=0xc00087a7e0 sp=0xc00087a7c8 pc=0x11b1ba5
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00087a7e8 sp=0xc00087a7e0 pc=0x481ec1
created by github.com/libp2p/go-libp2p-pubsub.(*validation).Start in goroutine 207
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/validation.go:136 +0x66

goroutine 292 gp=0xc0009056c0 m=nil [select]:
runtime.gopark(0xc00087af88?, 0x2?, 0x0?, 0x0?, 0xc00087af84?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00087ae18 sp=0xc00087adf8 pc=0x479f4e
runtime.selectgo(0xc00087af88, 0xc00087af80, 0x0?, 0x0, 0x0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc00087af40 sp=0xc00087ae18 pc=0x4573c5
github.com/libp2p/go-libp2p-pubsub.(*validation).validateWorker(0xc0008c0500)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/validation.go:283 +0xbf fp=0xc00087afc8 sp=0xc00087af40 pc=0x11b2aff
github.com/libp2p/go-libp2p-pubsub.(*validation).Start.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/validation.go:136 +0x25 fp=0xc00087afe0 sp=0xc00087afc8 pc=0x11b1ba5
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00087afe8 sp=0xc00087afe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p-pubsub.(*validation).Start in goroutine 207
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/validation.go:136 +0x66

goroutine 293 gp=0xc000905880 m=nil [select]:
runtime.gopark(0xc00087b788?, 0x2?, 0x0?, 0x0?, 0xc00087b784?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00087b618 sp=0xc00087b5f8 pc=0x479f4e
runtime.selectgo(0xc00087b788, 0xc00087b780, 0x0?, 0x0, 0x0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc00087b740 sp=0xc00087b618 pc=0x4573c5
github.com/libp2p/go-libp2p-pubsub.(*validation).validateWorker(0xc0008c0500)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/validation.go:283 +0xbf fp=0xc00087b7c8 sp=0xc00087b740 pc=0x11b2aff
github.com/libp2p/go-libp2p-pubsub.(*validation).Start.gowrap1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/validation.go:136 +0x25 fp=0xc00087b7e0 sp=0xc00087b7c8 pc=0x11b1ba5
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00087b7e8 sp=0xc00087b7e0 pc=0x481ec1
created by github.com/libp2p/go-libp2p-pubsub.(*validation).Start in goroutine 207
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/validation.go:136 +0x66

goroutine 294 gp=0xc000905a40 m=nil [select]:
runtime.gopark(0xc0005cfdb8?, 0x13?, 0x0?, 0x0?, 0xc0005cfb22?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc0005cf980 sp=0xc0005cf960 pc=0x479f4e
runtime.selectgo(0xc0005cfdb8, 0xc0005cfafc, 0x26?, 0x0, 0xe?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc0005cfaa8 sp=0xc0005cf980 pc=0x4573c5
github.com/libp2p/go-libp2p-pubsub.(*PubSub).processLoop(0xc0002ee248, {0x1b34be0, 0xc0008c0460})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/pubsub.go:574 +0x4bc fp=0xc0005cffb8 sp=0xc0005cfaa8 pc=0x119d17c
github.com/libp2p/go-libp2p-pubsub.NewPubSub.gowrap2()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/pubsub.go:337 +0x28 fp=0xc0005cffe0 sp=0xc0005cffb8 pc=0x119cc28
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc0005cffe8 sp=0xc0005cffe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p-pubsub.NewPubSub in goroutine 207
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/pubsub.go:337 +0xe52

goroutine 295 gp=0xc000905c00 m=nil [chan receive]:
runtime.gopark(0x1385700?, 0x7f0270383cc8?, 0x50?, 0x8d?, 0x4b6d79?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000a38cc0 sp=0xc000a38ca0 pc=0x479f4e
runtime.chanrecv(0xc00084f030, 0x0, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/chan.go:639 +0x41c fp=0xc000a38d38 sp=0xc000a38cc0 pc=0x412b5c
runtime.chanrecv1(0xc00036c000?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/chan.go:489 +0x12 fp=0xc000a38d60 sp=0xc000a38d38 pc=0x412712
main.startNodeWithComponents.func1()
	/home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP_PROTO/main.go:1302 +0xc12 fp=0xc000a38fe0 sp=0xc000a38d60 pc=0x1217f32
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc000a38fe8 sp=0xc000a38fe0 pc=0x481ec1
created by main.startNodeWithComponents in goroutine 207
	/home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP_PROTO/main.go:1102 +0x24c

goroutine 144 gp=0xc0001ffc00 m=nil [select]:
runtime.gopark(0xc00087be40?, 0x2?, 0x18?, 0x3f?, 0xc00087be2c?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000440cd0 sp=0xc000440cb0 pc=0x479f4e
runtime.selectgo(0xc000440e40, 0xc00087be28, 0xc00012a108?, 0x0, 0x0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc000440df8 sp=0xc000440cd0 pc=0x4573c5
github.com/libp2p/go-libp2p-pubsub.(*Subscription).Next(0xc0008c06e0, {0x1b34be0, 0xc0008c0460})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p-pubsub@v0.13.1/subscription.go:26 +0x85 fp=0xc000440e70 sp=0xc000440df8 pc=0x11a9985
main.(*P2PConsensusManager).handleTransactions(0xc000a28090)
	/home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP_PROTO/p2p_consensus.go:215 +0x45 fp=0xc000440fc8 sp=0xc000440e70 pc=0x122c7c5
main.(*P2PConsensusManager).Start.gowrap2()
	/home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP_PROTO/p2p_consensus.go:172 +0x25 fp=0xc000440fe0 sp=0xc000440fc8 pc=0x122c005
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc000440fe8 sp=0xc000440fe0 pc=0x481ec1
created by main.(*P2PConsensusManager).Start in goroutine 295
	/home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP_PROTO/p2p_consensus.go:172 +0x179

goroutine 145 gp=0xc0001ffdc0 m=nil [select]:
runtime.gopark(0xc000929780?, 0x2?, 0x70?, 0x0?, 0xc000929774?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000929610 sp=0xc0009295f0 pc=0x479f4e
runtime.selectgo(0xc000929780, 0xc000929770, 0x0?, 0x0, 0x0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc000929738 sp=0xc000929610 pc=0x4573c5
main.(*P2PConsensusManager).runForkResolution(0xc000a28090)
	/home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP_PROTO/p2p_consensus.go:339 +0xaa fp=0xc0009297c8 sp=0xc000929738 pc=0x122e20a
main.(*P2PConsensusManager).Start.gowrap3()
	/home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP_PROTO/p2p_consensus.go:175 +0x25 fp=0xc0009297e0 sp=0xc0009297c8 pc=0x122bfa5
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc0009297e8 sp=0xc0009297e0 pc=0x481ec1
created by main.(*P2PConsensusManager).Start in goroutine 295
	/home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP_PROTO/p2p_consensus.go:175 +0x1bb

goroutine 297 gp=0xc000905dc0 m=nil [chan receive (nil chan)]:
runtime.gopark(0x0?, 0x0?, 0x0?, 0x0?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00087cea0 sp=0xc00087ce80 pc=0x479f4e
runtime.chanrecv(0x0?, 0x0?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/chan.go:516 +0x199 fp=0xc00087cf18 sp=0xc00087cea0 pc=0x4128d9
runtime.chanrecv2(0x0?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/chan.go:494 +0x12 fp=0xc00087cf40 sp=0xc00087cf18 pc=0x412732
main.NewGUI.func1()
	/home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP_PROTO/altgui.go:85 +0xdb fp=0xc00087cfe0 sp=0xc00087cf40 pc=0x11ca0bb
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00087cfe8 sp=0xc00087cfe0 pc=0x481ec1
created by main.NewGUI in goroutine 1
	/home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP_PROTO/altgui.go:84 +0x2cd

goroutine 298 gp=0xc00044a000 m=nil [chan receive]:
runtime.gopark(0xc00091da50?, 0x8cf89418b403?, 0x0?, 0xe4?, 0x19a13d0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000399688 sp=0xc000399668 pc=0x479f4e
runtime.chanrecv(0xc00091d9d0, 0xc000399758, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/chan.go:639 +0x41c fp=0xc000399700 sp=0xc000399688 pc=0x412b5c
runtime.chanrecv2(0x2540be400?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/chan.go:494 +0x12 fp=0xc000399728 sp=0xc000399700 pc=0x412732
main.NewGUI.func2()
	/home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP_PROTO/altgui.go:97 +0x96 fp=0xc0003997e0 sp=0xc000399728 pc=0x11c9e16
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc0003997e8 sp=0xc0003997e0 pc=0x481ec1
created by main.NewGUI in goroutine 1
	/home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP_PROTO/altgui.go:93 +0x309

goroutine 299 gp=0xc00044a1c0 m=9 mp=0xc000282708 [syscall]:
runtime.cgocall(0x1256580, 0xc00079de80)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/cgocall.go:167 +0x4b fp=0xc00079de58 sp=0xc00079de20 pc=0x473b0b
github.com/webview/webview_go._Cfunc_webview_run(0x7f0250000b70)
	_cgo_gotypes.go:270 +0x3f fp=0xc00079de80 sp=0xc00079de58 pc=0x87739f
github.com/webview/webview_go.(*webview).Run.func1(0x11cf41c?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/webview/webview_go@v0.0.0-20240831120633-6173450d4dd6/webview.go:166 +0x34 fp=0xc00079deb8 sp=0xc00079de80 pc=0x877994
github.com/webview/webview_go.(*webview).Run(0xc000916158?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/webview/webview_go@v0.0.0-20240831120633-6173450d4dd6/webview.go:166 +0x13 fp=0xc00079ded0 sp=0xc00079deb8 pc=0x877933
main.(*GUI).createWebViewWindow(0xc0004d60b0)
	/home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP_PROTO/altgui.go:686 +0x3cc fp=0xc00079df58 sp=0xc00079ded0 pc=0x11cf42c
main.(*GUI).Run(0xc0004d60b0)
	/home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP_PROTO/altgui.go:127 +0x15f fp=0xc00079dfc8 sp=0xc00079df58 pc=0x11ca31f
main.InitializeGUI.gowrap1()
	/home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP_PROTO/altgui.go:727 +0x25 fp=0xc00079dfe0 sp=0xc00079dfc8 pc=0x11cff25
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00079dfe8 sp=0xc00079dfe0 pc=0x481ec1
created by main.InitializeGUI in goroutine 1
	/home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP_PROTO/altgui.go:727 +0x1a9

goroutine 300 gp=0xc00044a380 m=nil [IO wait]:
runtime.gopark(0x0?, 0x0?, 0x0?, 0x0?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc00079cb10 sp=0xc00079caf0 pc=0x479f4e
runtime.netpollblock(0x10?, 0x40ff66?, 0x0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/netpoll.go:575 +0xf7 fp=0xc00079cb48 sp=0xc00079cb10 pc=0x43ddd7
internal/poll.runtime_pollWait(0x7f02703c77d8, 0x72)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/netpoll.go:351 +0x85 fp=0xc00079cb68 sp=0xc00079cb48 pc=0x479245
internal/poll.(*pollDesc).wait(0xc000944f00?, 0x10?, 0x0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_poll_runtime.go:84 +0x27 fp=0xc00079cb90 sp=0xc00079cb68 pc=0x50a5c7
internal/poll.(*pollDesc).waitRead(...)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_poll_runtime.go:89
internal/poll.(*FD).Accept(0xc000944f00)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/internal/poll/fd_unix.go:620 +0x295 fp=0xc00079cc38 sp=0xc00079cb90 pc=0x50f995
net.(*netFD).accept(0xc000944f00)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/fd_unix.go:172 +0x29 fp=0xc00079ccf0 sp=0xc00079cc38 pc=0x61ddc9
net.(*TCPListener).accept(0xc000a12500)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/tcpsock_posix.go:159 +0x1e fp=0xc00079cd40 sp=0xc00079ccf0 pc=0x637d3e
net.(*TCPListener).Accept(0xc000a12500)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/tcpsock.go:372 +0x30 fp=0xc00079cd70 sp=0xc00079cd40 pc=0x636bf0
net/http.(*onceCloseListener).Accept(0x1b34b70?)
	<autogenerated>:1 +0x24 fp=0xc00079cd88 sp=0xc00079cd70 pc=0x794584
net/http.(*Server).Serve(0xc0004b33b0, {0x1b30da8, 0xc000a12500})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/http/server.go:3330 +0x30c fp=0xc00079ceb8 sp=0xc00079cd88 pc=0x76c24c
net/http.(*Server).ListenAndServe(0xc0004b33b0)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/net/http/server.go:3259 +0x71 fp=0xc00079cee8 sp=0xc00079ceb8 pc=0x76bf11
main.(*GUI).startAPIServer(0xc0004d60b0)
	/home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP_PROTO/altgui.go:502 +0x5e5 fp=0xc00079cfc8 sp=0xc00079cee8 pc=0x11cb925
main.(*GUI).Run.gowrap1()
	/home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP_PROTO/altgui.go:114 +0x25 fp=0xc00079cfe0 sp=0xc00079cfc8 pc=0x11ca425
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc00079cfe8 sp=0xc00079cfe0 pc=0x481ec1
created by main.(*GUI).Run in goroutine 299
	/home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP_PROTO/altgui.go:114 +0x56

goroutine 339 gp=0xc00044aa80 m=nil [select]:
runtime.gopark(0xc000399f78?, 0x2?, 0x78?, 0x48?, 0xc000399f74?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000399e18 sp=0xc000399df8 pc=0x479f4e
runtime.selectgo(0xc000399f78, 0xc000399f70, 0x0?, 0x0, 0xc000399f98?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc000399f40 sp=0xc000399e18 pc=0x4573c5
main.(*SystrayManager).Run.func1()
	/home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP_PROTO/systray.go:54 +0x6e fp=0xc000399fe0 sp=0xc000399f40 pc=0x124302e
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc000399fe8 sp=0xc000399fe0 pc=0x481ec1
created by main.(*SystrayManager).Run in goroutine 316
	/home/gperry/Documents/GitHub/cloud-equities/KNIRVCHAIN_GO_ROOT_MCP_PROTO/systray.go:53 +0xb6

goroutine 272 gp=0xc0004a41c0 m=nil [select]:
runtime.gopark(0xc000272ee8?, 0x4?, 0x20?, 0x2d?, 0xc000272dfc?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000272c38 sp=0xc000272c18 pc=0x479f4e
runtime.selectgo(0xc000272ee8, 0xc000272df4, 0x5a?, 0x0, 0xa64eda?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc000272d60 sp=0xc000272c38 pc=0x4573c5
github.com/libp2p/go-yamux/v4.(*Session).sendLoop(0xc000533200)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.2/session.go:589 +0x587 fp=0xc000272fa0 sp=0xc000272d60 pc=0xa61f07
github.com/libp2p/go-yamux/v4.(*Session).send(0xc000533200)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.2/session.go:518 +0x18 fp=0xc000272fc8 sp=0xc000272fa0 pc=0xa61938
github.com/libp2p/go-yamux/v4.newSession.gowrap2()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.2/session.go:163 +0x25 fp=0xc000272fe0 sp=0xc000272fc8 pc=0xa5f4c5
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc000272fe8 sp=0xc000272fe0 pc=0x481ec1
created by github.com/libp2p/go-yamux/v4.newSession in goroutine 158
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.2/session.go:163 +0x556

goroutine 273 gp=0xc0004a4380 m=nil [select]:
runtime.gopark(0xc0002d9790?, 0x2?, 0x70?, 0x0?, 0xc0002d9774?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc0002d9610 sp=0xc0002d95f0 pc=0x479f4e
runtime.selectgo(0xc0002d9790, 0xc0002d9770, 0x0?, 0x0, 0x7f02704460d0?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc0002d9738 sp=0xc0002d9610 pc=0x4573c5
github.com/libp2p/go-yamux/v4.(*Session).startMeasureRTT(0xc000533200)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.2/session.go:356 +0xc5 fp=0xc0002d97c8 sp=0xc0002d9738 pc=0xa60725
github.com/libp2p/go-yamux/v4.newSession.gowrap3()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.2/session.go:164 +0x25 fp=0xc0002d97e0 sp=0xc0002d97c8 pc=0xa5f465
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc0002d97e8 sp=0xc0002d97e0 pc=0x481ec1
created by github.com/libp2p/go-yamux/v4.newSession in goroutine 158
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.2/session.go:164 +0x596

goroutine 355 gp=0xc0004a4540 m=nil [select]:
runtime.gopark(0xc000931e98?, 0x2?, 0xa?, 0x0?, 0xc000931e74?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/proc.go:424 +0xce fp=0xc000931d18 sp=0xc000931cf8 pc=0x479f4e
runtime.selectgo(0xc000931e98, 0xc000931e70, 0x1?, 0x0, 0xa3eb99?, 0x1)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/select.go:335 +0x7a5 fp=0xc000931e40 sp=0xc000931d18 pc=0x4573c5
github.com/libp2p/go-yamux/v4.(*Session).AcceptStream(0xc000533200)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-yamux/v4@v4.0.2/session.go:272 +0x106 fp=0xc000931ef8 sp=0xc000931e40 pc=0xa5ffa6
github.com/libp2p/go-libp2p/p2p/muxer/yamux.(*conn).AcceptStream(0x44efa0?)
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/muxer/yamux/conn.go:43 +0x13 fp=0xc000931f10 sp=0xc000931ef8 pc=0xa67893
github.com/libp2p/go-libp2p/p2p/net/upgrader.(*transportConn).AcceptStream(0xba74eb?)
	<autogenerated>:1 +0x24 fp=0xc000931f28 sp=0xc000931f10 pc=0xbd6524
github.com/libp2p/go-libp2p/p2p/net/swarm.(*connWithMetrics).AcceptStream(0xc000000000?)
	<autogenerated>:1 +0x24 fp=0xc000931f40 sp=0xc000931f28 pc=0xbb6b24
github.com/libp2p/go-libp2p/p2p/net/swarm.(*Conn).start.func1()
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/swarm_conn.go:118 +0x96 fp=0xc000931fe0 sp=0xc000931f40 pc=0xba73d6
runtime.goexit({})
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/golang.org/toolchain@v0.0.1-go1.23.3.linux-amd64/src/runtime/asm_amd64.s:1700 +0x1 fp=0xc000931fe8 sp=0xc000931fe0 pc=0x481ec1
created by github.com/libp2p/go-libp2p/p2p/net/swarm.(*Conn).start in goroutine 157
	/home/gperry/.gvm/pkgsets/go1.23.1/global/pkg/mod/github.com/libp2p/go-libp2p@v0.39.1/p2p/net/swarm/swarm_conn.go:114 +0x4f

rax    0x0
rbx    0x7f02697fa640
rcx    0x7f02bca059fc
rdx    0x6
rdi    0x2a5f1a
rsi    0x2a5f21
rbp    0x2a5f21
rsp    0x7f02697f9570
r8     0x7f02697f9640
r9     0x2
r10    0x8
r11    0x246
r12    0x6
r13    0x16
r14    0x7f02be91cac0
r15    0x7f023a0194b0
rip    0x7f02bca059fc
rflags 0x246
cs     0x33
fs     0x0
gs     0x0
exit status 2


---

<div class="footer-links">


© 2025 KNIRV Network
</div>
