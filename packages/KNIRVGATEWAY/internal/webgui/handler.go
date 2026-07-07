package webgui

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/KNIRV/KNIRV_NETWORK/KNIRVGATEWAY/internal/config"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// Handler handles explorer API requests.
type Handler struct {
	config      *config.Config
	logger      *zap.Logger
	httpClient  *http.Client
	gatewayPort int
}

// NewHandler creates a new explorer handler.
func NewHandler(cfg *config.Config, logger *zap.Logger, gatewayPort ...int) *Handler {
	port := 8080
	if len(gatewayPort) > 0 && gatewayPort[0] > 0 {
		port = gatewayPort[0]
	} else if cfg.Port > 0 {
		port = cfg.Port
	}
	return &Handler{
		config:      cfg,
		logger:      logger,
		httpClient:  &http.Client{},
		gatewayPort: port,
	}
}

// RegisterRoutes registers the explorer API routes.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	// These routes are now handled directly by the gateway's setupRoutes()
	// with proper proxy destinations (chain.sock or oracle). We keep only
	// /api/view/{id} which is deferred to Phase 8.
	r.HandleFunc("/api/view/{id}", h.handleView).Methods("GET")
}

// getBackendURL returns the gateway's own /api/v1 base URL for proxying requests
// through the socket proxy to the real backend.
func (h *Handler) getBackendURL() string {
	return fmt.Sprintf("http://localhost:%d/api/v1", h.gatewayPort)
}

// proxyRequest proxies a request to the backend
func (h *Handler) proxyRequest(w http.ResponseWriter, r *http.Request, endpoint string) {
	backendURL := h.getBackendURL() + endpoint

	h.logger.Debug("Proxying request",
		zap.String("method", r.Method),
		zap.String("url", r.URL.Path),
		zap.String("backend", backendURL),
	)

	// Create request to backend
	req, err := http.NewRequest(r.Method, backendURL, r.Body)
	if err != nil {
		h.logger.Error("Failed to create backend request", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Copy headers
	for key, values := range r.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// Make request to backend
	resp, err := h.httpClient.Do(req)
	if err != nil {
		h.logger.Error("Failed to proxy request to backend",
			zap.String("backend", backendURL),
			zap.Error(err),
		)

		// Return mock data as fallback (similar to TypeScript implementation)
		h.returnMockData(w, endpoint)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Set status code
	w.WriteHeader(resp.StatusCode)

	// Copy response body
	io.Copy(w, resp.Body)
}

// returnMockData returns mock data when backend is unavailable
func (h *Handler) returnMockData(w http.ResponseWriter, endpoint string) {
	w.Header().Set("Content-Type", "application/json")

	switch endpoint {
	case "/api/objects":
		mockData := []map[string]interface{}{
			{"id": "mock-1", "name": "Mock Cube", "object_type": "glb"},
			{"id": "mock-2", "name": "Mock Sphere", "object_type": "glb"},
		}
		json.NewEncoder(w).Encode(mockData)

	case "/api/transactions", "/api/blocks", "/api/assets":
		// Return empty array for these endpoints
		json.NewEncoder(w).Encode([]interface{}{})

	default:
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Backend unavailable",
			"message": "Failed to fetch from backend, no mock data available",
		})
	}
}

func (h *Handler) handleObjects(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, "/api/objects")
}

func (h *Handler) handleTransactions(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, "/api/transactions")
}

func (h *Handler) handleBlocks(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, "/api/blocks")
}

func (h *Handler) handleAssets(w http.ResponseWriter, r *http.Request) {
	h.proxyRequest(w, r, "/api/assets")
}

func (h *Handler) handleView(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if id == "" {
		http.Error(w, "Invalid or missing object ID", http.StatusBadRequest)
		return
	}

	// First, try to get object data from backend
	backendURL := h.getBackendURL()
	objectURL := backendURL + "/api/objects/" + id

	h.logger.Debug("Fetching object data",
		zap.String("id", id),
		zap.String("url", objectURL),
	)

	req, err := http.NewRequest("GET", objectURL, nil)
	if err != nil {
		h.logger.Error("Failed to create object request", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	resp, err := h.httpClient.Do(req)
	if err != nil {
		h.logger.Error("Failed to fetch object data",
			zap.String("id", id),
			zap.Error(err),
		)
		http.Error(w, "Failed to fetch object data", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		http.Error(w, fmt.Sprintf("Object with ID %s not found", id), http.StatusNotFound)
		return
	}

	if resp.StatusCode != http.StatusOK {
		h.logger.Error("Backend returned error for object",
			zap.String("id", id),
			zap.Int("status", resp.StatusCode),
		)
		http.Error(w, "Failed to fetch object data", http.StatusInternalServerError)
		return
	}

	// Parse object data
	var objectData map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&objectData); err != nil {
		h.logger.Error("Failed to parse object data", zap.Error(err))
		http.Error(w, "Invalid object data", http.StatusInternalServerError)
		return
	}

	// Generate HTML viewer
	h.generateViewerHTML(w, id, objectData)
}

// generateViewerHTML generates the 3D viewer HTML page
func (h *Handler) generateViewerHTML(w http.ResponseWriter, id string, objectData map[string]interface{}) {
	// Get model URL
	modelURL := "/assets/models/cube.gltf" // Default fallback
	if data, ok := objectData["file_path"].(string); ok && data != "" {
		modelURL = data
	}

	// Get object name
	objectName := "Object"
	if name, ok := objectData["name"].(string); ok {
		objectName = name
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>KNIRVCHAIN - %s</title>
    <script type="importmap">
        {
            "imports": {
                "three": "https://unpkg.com/three@0.175.0/build/three.module.js",
                "three/addons/": "https://unpkg.com/three@0.175.0/examples/jsm/"
            }
        }
    </script>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
        }

        body {
            margin: 0;
            overflow: hidden;
            background-color: #0a0e17;
            color: #e0e0ff;
        }

        canvas {
            display: block;
        }

        .ui-container {
            position: absolute;
            top: 0;
            left: 0;
            width: 100%%;
            height: 100%%;
            pointer-events: none;
            z-index: 10;
        }

        .header {
            position: absolute;
            top: 0;
            left: 0;
            right: 0;
            padding: 15px 20px;
            background: rgba(16, 24, 48, 0.7);
            backdrop-filter: blur(10px);
            border-bottom: 1px solid rgba(100, 130, 255, 0.2);
            display: flex;
            justify-content: space-between;
            align-items: center;
            pointer-events: auto;
        }

        .logo {
            display: flex;
            align-items: center;
        }

        .logo h1 {
            font-size: 22px;
            font-weight: 700;
            background: linear-gradient(90deg, #64b5f6, #9575cd);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            margin-right: 15px;
        }

        .object-title {
            font-size: 16px;
            color: #d0d0ff;
            font-weight: 500;
        }

        .object-id {
            font-size: 12px;
            color: #8080c0;
            margin-left: 10px;
        }

        .controls {
            display: flex;
            gap: 10px;
        }

        .control-button {
            background: rgba(30, 40, 80, 0.6);
            border: 1px solid rgba(100, 130, 255, 0.3);
            color: #b0b0ff;
            padding: 6px 12px;
            border-radius: 6px;
            cursor: pointer;
            transition: all 0.2s ease;
            font-size: 13px;
            pointer-events: auto;
        }

        .control-button:hover {
            background: rgba(40, 60, 120, 0.7);
            border-color: rgba(120, 150, 255, 0.5);
        }

        .info-panel {
            position: absolute;
            bottom: 20px;
            left: 20px;
            width: 300px;
            background: rgba(16, 24, 48, 0.7);
            backdrop-filter: blur(10px);
            border: 1px solid rgba(100, 130, 255, 0.2);
            border-radius: 10px;
            padding: 15px;
            pointer-events: auto;
            box-shadow: 0 4px 20px rgba(0, 0, 0, 0.3);
            transition: transform 0.3s ease, opacity 0.3s ease;
        }

        .info-panel.hidden {
            transform: translateY(20px);
            opacity: 0;
            pointer-events: none;
        }

        .panel-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 10px;
            padding-bottom: 8px;
            border-bottom: 1px solid rgba(100, 130, 255, 0.2);
        }

        .panel-title {
            font-size: 16px;
            font-weight: 600;
            color: #d0d0ff;
        }

        .close-button {
            background: none;
            border: none;
            color: #8080c0;
            cursor: pointer;
            font-size: 18px;
            transition: color 0.2s ease;
        }

        .close-button:hover {
            color: #b0b0ff;
        }

        .info-content {
            font-size: 14px;
            line-height: 1.5;
        }

        .info-row {
            display: flex;
            margin-bottom: 8px;
        }

        .info-label {
            width: 100px;
            color: #8080c0;
        }

        .info-value {
            flex: 1;
            color: #d0d0ff;
        }

        #loading {
            position: absolute;
            top: 50%%;
            left: 50%%;
            transform: translate(-50%%, -50%%);
            background: rgba(16, 24, 48, 0.8);
            backdrop-filter: blur(5px);
            border: 1px solid rgba(100, 130, 255, 0.3);
            border-radius: 10px;
            padding: 20px 30px;
            color: #d0d0ff;
            font-size: 16px;
            box-shadow: 0 4px 20px rgba(0, 0, 0, 0.3);
            z-index: 20;
        }

        .chain-info {
            position: absolute;
            top: 80px;
            right: 20px;
            background: rgba(16, 24, 48, 0.7);
            backdrop-filter: blur(10px);
            border: 1px solid rgba(100, 130, 255, 0.2);
            border-radius: 10px;
            padding: 12px;
            pointer-events: auto;
            max-width: 250px;
            box-shadow: 0 4px 20px rgba(0, 0, 0, 0.3);
        }

        .chain-header {
            font-size: 14px;
            font-weight: 600;
            color: #d0d0ff;
            margin-bottom: 8px;
            padding-bottom: 6px;
            border-bottom: 1px solid rgba(100, 130, 255, 0.2);
        }

        .chain-data {
            font-size: 12px;
            line-height: 1.4;
        }

        .chain-row {
            display: flex;
            margin-bottom: 6px;
        }

        .chain-label {
            width: 80px;
            color: #8080c0;
        }

        .chain-value {
            flex: 1;
            color: #d0d0ff;
            word-break: break-all;
            font-family: monospace;
        }

        .help-tooltip {
            position: absolute;
            bottom: 20px;
            right: 20px;
            background: rgba(16, 24, 48, 0.7);
            backdrop-filter: blur(10px);
            border: 1px solid rgba(100, 130, 255, 0.2);
            border-radius: 50%%;
            width: 40px;
            height: 40px;
            display: flex;
            align-items: center;
            justify-content: center;
            color: #b0b0ff;
            font-size: 20px;
            cursor: pointer;
            pointer-events: auto;
            transition: all 0.2s ease;
            box-shadow: 0 4px 10px rgba(0, 0, 0, 0.2);
        }

        .help-tooltip:hover {
            background: rgba(30, 40, 80, 0.8);
            transform: translateY(-2px);
            box-shadow: 0 6px 15px rgba(0, 0, 0, 0.3);
        }

        .help-panel {
            position: absolute;
            bottom: 70px;
            right: 20px;
            width: 300px;
            background: rgba(16, 24, 48, 0.9);
            backdrop-filter: blur(10px);
            border: 1px solid rgba(100, 130, 255, 0.3);
            border-radius: 10px;
            padding: 15px;
            pointer-events: auto;
            box-shadow: 0 4px 20px rgba(0, 0, 0, 0.4);
            transform: translateY(20px);
            opacity: 0;
            visibility: hidden;
            transition: all 0.3s ease;
        }

        .help-panel.visible {
            transform: translateY(0);
            opacity: 1;
            visibility: visible;
        }

        .help-title {
            font-size: 16px;
            font-weight: 600;
            color: #d0d0ff;
            margin-bottom: 10px;
            padding-bottom: 8px;
            border-bottom: 1px solid rgba(100, 130, 255, 0.2);
        }

        .help-content {
            font-size: 13px;
            line-height: 1.5;
            color: #b0b0ff;
        }

        .help-item {
            margin-bottom: 8px;
        }

        .help-key {
            display: inline-block;
            background: rgba(60, 80, 170, 0.4);
            border: 1px solid rgba(100, 130, 255, 0.4);
            border-radius: 4px;
            padding: 2px 6px;
            margin-right: 6px;
            font-family: monospace;
            font-size: 12px;
            color: #d0d0ff;
        }

        @media (max-width: 768px) {
            .header {
                padding: 10px 15px;
            }

            .logo h1 {
                font-size: 18px;
            }

            .object-title {
                font-size: 14px;
            }

            .info-panel, .chain-info {
                width: calc(100%% - 40px);
                max-width: none;
            }

            .chain-info {
                top: auto;
                bottom: 70px;
                right: 20px;
                left: 20px;
            }
        }
    </style>
</head>
<body>
    <div id="loading">Loading model...</div>

    <div class="ui-container">
        <div class="header">
            <div class="logo">
                <h1>KNIRVCHAIN</h1>
                <span class="object-title">%s</span>
                <span class="object-id">ID: %s</span>
            </div>
            <div class="controls">
                <button class="control-button" id="toggle-info">Object Info</button>
                <button class="control-button" id="reset-camera">Reset View</button>
            </div>
        </div>

        <div class="info-panel hidden" id="info-panel">
            <div class="panel-header">
                <div class="panel-title">Object Information</div>
                <button class="close-button" id="close-info">×</button>
            </div>
            <div class="info-content">
                <div class="info-row">
                    <div class="info-label">Name:</div>
                    <div class="info-value">%s</div>
                </div>
                <div class="info-row">
                    <div class="info-label">ID:</div>
                    <div class="info-value">%s</div>
                </div>
                <div class="info-row">
                    <div class="info-label">Type:</div>
                    <div class="info-value">%s</div>
                </div>
                <div class="info-row">
                    <div class="info-label">Author:</div>
                    <div class="info-value">%s</div>
                </div>
                <div class="info-row">
                    <div class="info-label">Created:</div>
                    <div class="info-value">%s</div>
                </div>
                <div class="info-row">
                    <div class="info-label">License:</div>
                    <div class="info-value">%s</div>
                </div>
            </div>
        </div>

        <div class="chain-info">
            <div class="chain-header">Blockchain Data</div>
            <div class="chain-data">
                <div class="chain-row">
                    <div class="chain-label">TX Hash:</div>
                    <div class="chain-value">%s</div>
                </div>
                <div class="chain-row">
                    <div class="chain-label">Block:</div>
                    <div class="chain-value">#%s</div>
                </div>
                <div class="chain-row">
                    <div class="chain-label">Status:</div>
                    <div class="chain-value">Confirmed</div>
                </div>
            </div>
        </div>

        <div class="help-tooltip" id="help-button">?</div>

        <div class="help-panel" id="help-panel">
            <div class="help-title">Controls Help</div>
            <div class="help-content">
                <div class="help-item">
                    <span class="help-key">Left Click + Drag</span> Rotate model
                </div>
                <div class="help-item">
                    <span class="help-key">Right Click + Drag</span> Pan camera
                </div>
                <div class="help-item">
                    <span class="help-key">Scroll</span> Zoom in/out
                </div>
                <div class="help-item">
                    <span class="help-key">Double Click</span> Reset camera
                </div>
                <div class="help-item">
                    <span class="help-key">R</span> Reset view
                </div>
            </div>
        </div>
    </div>

    <script type="module">
        import * as THREE from 'three';
        import { OrbitControls } from 'three/addons/controls/OrbitControls.js';
        import { GLTFLoader } from 'three/addons/loaders/GLTFLoader.js';

        // UI Elements
        const loadingElement = document.getElementById('loading');
        const toggleInfoButton = document.getElementById('toggle-info');
        const resetCameraButton = document.getElementById('reset-camera');
        const infoPanel = document.getElementById('info-panel');
        const closeInfoButton = document.getElementById('close-info');
        const helpButton = document.getElementById('help-button');
        const helpPanel = document.getElementById('help-panel');

        // UI Event Listeners
        toggleInfoButton.addEventListener('click', () => {
            infoPanel.classList.toggle('hidden');
        });

        closeInfoButton.addEventListener('click', () => {
            infoPanel.classList.add('hidden');
        });

        helpButton.addEventListener('click', () => {
            helpPanel.classList.toggle('visible');
        });

        document.addEventListener('keydown', (event) => {
            if (event.key === 'r' || event.key === 'R') {
                resetCamera();
            }

            if (event.key === 'Escape') {
                infoPanel.classList.add('hidden');
                helpPanel.classList.remove('visible');
            }
        });

        // Three.js Setup
        const scene = new THREE.Scene();
        scene.background = new THREE.Color(0x0a0e17);

        // Camera setup
        const camera = new THREE.PerspectiveCamera(75, window.innerWidth / window.innerHeight, 0.1, 1000);
        camera.position.z = 5;

        // Renderer setup with anti-aliasing and better shadows
        const renderer = new THREE.WebGLRenderer({
            antialias: true,
            alpha: true
        });
        renderer.setSize(window.innerWidth, window.innerHeight);
        renderer.setPixelRatio(window.devicePixelRatio);
        renderer.shadowMap.enabled = true;
        renderer.shadowMap.type = THREE.PCFSoftShadowMap;
        renderer.outputEncoding = THREE.sRGBEncoding;
        document.body.appendChild(renderer.domElement);

        // Lighting setup for better visual quality
        // Ambient light for general illumination
        const ambientLight = new THREE.AmbientLight(0x404060, 1.0);
        scene.add(ambientLight);

        // Main directional light with shadows
        const mainLight = new THREE.DirectionalLight(0xffffff, 1.5);
        mainLight.position.set(5, 10, 7);
        mainLight.castShadow = true;
        mainLight.shadow.mapSize.width = 1024;
        mainLight.shadow.mapSize.height = 1024;
        scene.add(mainLight);

        // Fill light from opposite direction
        const fillLight = new THREE.DirectionalLight(0x8080ff, 0.7);
        fillLight.position.set(-5, 2, -7);
        scene.add(fillLight);

        // Rim light for edge highlighting
        const rimLight = new THREE.DirectionalLight(0x8080ff, 0.5);
        rimLight.position.set(0, -10, -7);
        scene.add(rimLight);

        // Controls setup
        const controls = new OrbitControls(camera, renderer.domElement);
        controls.enableDamping = true;
        controls.dampingFactor = 0.05;
        controls.screenSpacePanning = true;
        controls.minDistance = 1;
        controls.maxDistance = 50;
        controls.autoRotate = false;
        controls.autoRotateSpeed = 0.5;

        // Store initial camera position for reset
        let initialCameraPosition = null;
        let initialTarget = new THREE.Vector3();

        // Function to reset camera to initial position
        function resetCamera() {
            if (initialCameraPosition) {
                camera.position.copy(initialCameraPosition);
                controls.target.copy(initialTarget);
                controls.update();
            }
        }

        resetCameraButton.addEventListener('click', resetCamera);

        // Grid helper for better spatial awareness
        const gridHelper = new THREE.GridHelper(20, 20, 0x444466, 0x222233);
        gridHelper.position.y = -0.01; // Slightly below the object to avoid z-fighting
        scene.add(gridHelper);

        // Load the 3D model
        const loader = new GLTFLoader();
        const modelToLoad = '%s';
        console.log("Loading model from:", modelToLoad);

        loader.load(
            modelToLoad,
            (gltf) => {
                const model = gltf.scene;

                // Enable shadows for all meshes in the model
                model.traverse((node) => {
                    if (node.isMesh) {
                        node.castShadow = true;
                        node.receiveShadow = true;

                        // Improve material quality if needed
                        if (node.material) {
                            node.material.metalness = 0.5;
                            node.material.roughness = 0.5;
                        }
                    }
                });

                scene.add(model);

                // Center and scale the model appropriately
                const box = new THREE.Box3().setFromObject(model);
                const center = box.getCenter(new THREE.Vector3());
                const size = box.getSize(new THREE.Vector3());

                // Center the model
                model.position.sub(center);

                // Scale the model if it's too large or too small
                const maxDim = Math.max(size.x, size.y, size.z);
                if (maxDim > 10 || maxDim < 0.1) {
                    const scale = maxDim > 10 ? 10 / maxDim : 1 / maxDim;
                    model.scale.multiplyScalar(scale);
                }

                // Position camera based on model size
                const fov = camera.fov * (Math.PI / 180);
                let cameraZ = Math.abs(maxDim / 2 / Math.tan(fov / 2));
                cameraZ *= 1.5; // Add some padding

                camera.position.z = cameraZ;
                camera.lookAt(scene.position);
                controls.update();

                // Store initial camera position for reset
                initialCameraPosition = camera.position.clone();

                // Hide loading message
                loadingElement.style.display = 'none';

                // Start auto-rotation for a few seconds then stop
                controls.autoRotate = true;
                setTimeout(() => {
                    controls.autoRotate = false;
                }, 3000);
            },
            (xhr) => {
                const percentLoaded = Math.round(xhr.loaded / xhr.total * 100);
                loadingElement.textContent = 'Loading model: ' + percentLoaded + '%%';
                console.log((xhr.loaded / xhr.total * 100) + '%% loaded');
            },
            (error) => {
                console.error('Error loading model:', error);
                loadingElement.textContent = 'Error loading model: ' + error.message;

                // Create a fallback cube if model fails to load
                const geometry = new THREE.BoxGeometry(1, 1, 1);
                const material = new THREE.MeshStandardMaterial({
                    color: 0x6666ff,
                    metalness: 0.5,
                    roughness: 0.5
                });
                const cube = new THREE.Mesh(geometry, material);
                cube.castShadow = true;
                cube.receiveShadow = true;
                scene.add(cube);
            }
        );

        // Handle window resize
        window.addEventListener('resize', () => {
            camera.aspect = window.innerWidth / window.innerHeight;
            camera.updateProjectionMatrix();
            renderer.setSize(window.innerWidth, window.innerHeight);
        });

        // Animation loop
        function animate() {
            requestAnimationFrame(animate);
            controls.update(); // Required for damping
            renderer.render(scene, camera);
        }
        animate();
    </script>
</body>
</html>`, objectName, objectName, id, objectName, id, h.getObjectType(objectData), h.getAuthor(objectData), h.getCreatedAt(objectData), h.getLicense(objectData), h.getTransactionHash(objectData), h.getBlockNumber(), modelURL)

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// Helper functions to extract data from objectData
func (h *Handler) getObjectType(data map[string]interface{}) string {
	if val, ok := data["object_type"].(string); ok {
		return val
	}
	if val, ok := data["asset_type"].(string); ok {
		return val
	}
	return "Unknown"
}

func (h *Handler) getAuthor(data map[string]interface{}) string {
	if data, ok := data["data"].(map[string]interface{}); ok {
		if author, ok := data["author"].(string); ok {
			return author
		}
	}
	return "Unknown"
}

func (h *Handler) getCreatedAt(data map[string]interface{}) string {
	if createdAt, ok := data["created_at"].(string); ok {
		return createdAt
	}
	return "Unknown"
}

func (h *Handler) getLicense(data map[string]interface{}) string {
	if data, ok := data["data"].(map[string]interface{}); ok {
		if license, ok := data["license"].(string); ok {
			return license
		}
	}
	return "Unknown"
}

func (h *Handler) getTransactionHash(data map[string]interface{}) string {
	if transaction, ok := data["transaction"].(string); ok && len(transaction) > 10 {
		return transaction[:10] + "..."
	}
	return "Unknown"
}

func (h *Handler) getBlockNumber() string {
	// Mock block number for now
	return "12345"
}
