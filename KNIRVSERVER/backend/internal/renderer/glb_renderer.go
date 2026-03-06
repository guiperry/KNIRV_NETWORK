// Copyright 2026 KNIRV-SERVER
// SPDX-License-Identifier: GPL-3.0-or-later

package renderer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// GLBConfig defines GLB renderer configuration
type GLBConfig struct {
	Port           int    `json:"port"`
	MaxPolycount   int    `json:"max_polycount"`
	EnablePBR      bool   `json:"enable_pbr"`
	EnableShadows  bool   `json:"enable_shadows"`
	TextureQuality string `json:"texture_quality"` // low, medium, high, ultra
}

// GLBRenderer provides WebGL-based rendering for 3D GLB objects
type GLBRenderer struct {
	config        *GLBConfig
	assetRegistry *AssetRegistry
	httpServer    *http.Server
	running       bool
}

// NewGLBRenderer creates a new GLB renderer
func NewGLBRenderer() *GLBRenderer {
	return &GLBRenderer{
		config: &GLBConfig{
			Port:           8080,
			MaxPolycount:   1000000,
			EnablePBR:      true,
			EnableShadows:  true,
			TextureQuality: "high",
		},
		assetRegistry: NewAssetRegistry(),
		running:       false,
	}
}

// NewGLBRendererWithConfig creates a new GLB renderer with custom config
func NewGLBRendererWithConfig(config *GLBConfig) *GLBRenderer {
	return &GLBRenderer{
		config:        config,
		assetRegistry: NewAssetRegistry(),
		running:       false,
	}
}

// Start starts the GLB renderer HTTP server
func (gr *GLBRenderer) Start(ctx context.Context) error {
	if gr.running {
		return fmt.Errorf("renderer already running")
	}

	router := http.NewServeMux()
	router.HandleFunc("/", gr.serveViewer)
	router.HandleFunc("/asset.glb", gr.serveAsset)
	router.HandleFunc("/metadata", gr.serveMetadata)
	router.HandleFunc("/health", gr.serveHealth)

	gr.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", gr.config.Port),
		Handler: router,
	}

	gr.running = true

	go func() {
		log.Printf("GLB renderer starting on port %d", gr.config.Port)
		if err := gr.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("GLB renderer error: %v", err)
			gr.running = false
		}
	}()

	return nil
}

// Stop stops the GLB renderer HTTP server
func (gr *GLBRenderer) Stop(ctx context.Context) error {
	if !gr.running {
		return nil
	}

	if gr.httpServer != nil {
		if err := gr.httpServer.Shutdown(ctx); err != nil {
			return fmt.Errorf("server shutdown failed: %w", err)
		}
	}

	gr.running = false
	log.Println("GLB renderer stopped")

	return nil
}

// LoadAsset loads a GLB asset into the registry
func (gr *GLBRenderer) LoadAsset(path string) (*AssetMetadata, error) {
	// Parse GLB file and extract metadata
	metadata, err := parseGLBFile(path)
	if err != nil {
		return nil, fmt.Errorf("GLB parsing failed: %w", err)
	}

	// Validate polycount
	if metadata.Polycount > gr.config.MaxPolycount {
		return nil, fmt.Errorf("polycount %d exceeds limit %d",
			metadata.Polycount, gr.config.MaxPolycount)
	}

	// Register asset
	if err := gr.assetRegistry.Register(metadata); err != nil {
		return nil, fmt.Errorf("asset registration failed: %w", err)
	}

	return metadata, nil
}

// serveViewer serves the WebGL viewer interface
func (gr *GLBRenderer) serveViewer(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(glbViewerHTML))
}

// serveAsset serves the GLB binary data
func (gr *GLBRenderer) serveAsset(w http.ResponseWriter, r *http.Request) {
	assetID := r.URL.Query().Get("id")
	if assetID == "" {
		http.Error(w, "asset ID required", http.StatusBadRequest)
		return
	}

	asset, err := gr.assetRegistry.Get(assetID)
	if err != nil {
		http.Error(w, "asset not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "model/gltf-binary")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", asset.FileSize))
	w.Header().Set("Cache-Control", "public, max-age=31536000")

	http.ServeFile(w, r, asset.FilePath)
}

// serveMetadata serves asset metadata as JSON
func (gr *GLBRenderer) serveMetadata(w http.ResponseWriter, r *http.Request) {
	assetID := r.URL.Query().Get("id")
	if assetID == "" {
		http.Error(w, "asset ID required", http.StatusBadRequest)
		return
	}

	asset, err := gr.assetRegistry.Get(assetID)
	if err != nil {
		http.Error(w, "asset not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(asset.Metadata)
}

// serveHealth serves health check endpoint
func (gr *GLBRenderer) serveHealth(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"status":      "healthy",
		"running":     gr.running,
		"asset_count": gr.assetRegistry.Count(),
		"config":      gr.config,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// parseGLBFile parses a GLB file and extracts metadata
func parseGLBFile(path string) (*AssetMetadata, error) {
	// TODO: Implement actual GLB parsing
	// This would involve:
	// - Reading GLB binary format
	// - Extracting GLTF JSON chunk
	// - Parsing mesh data, materials, textures
	// - Calculating polycount and bounding box

	metadata := &AssetMetadata{
		Format:   "glb",
		FilePath: path,
		FileSize: 0,
		Dimensions: &Dimensions3D{
			Width:  1.0,
			Height: 1.0,
			Depth:  1.0,
		},
		BoundingBox: &BoundingBox3D{
			Min: Vector3D{X: -1, Y: -1, Z: -1},
			Max: Vector3D{X: 1, Y: 1, Z: 1},
		},
		Polycount:  0,
		Materials:  []string{},
		Animations: []string{},
		Textures:   []TextureInfo{},
	}

	return metadata, nil
}

// GLB Viewer HTML template (comprehensive version)
const glbViewerHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>KNIRV 3D Object Renderer</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
            color: #e0e0e0;
            overflow: hidden;
        }
        #canvas { width: 100vw; height: 100vh; display: block; }
        #ui-overlay {
            position: fixed;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            pointer-events: none;
        }
        #controls {
            position: absolute;
            top: 20px;
            left: 20px;
            background: rgba(26, 26, 46, 0.95);
            padding: 20px;
            border-radius: 12px;
            box-shadow: 0 8px 32px rgba(0, 0, 0, 0.5);
            pointer-events: all;
            max-width: 300px;
        }
        h3 {
            margin: 0 0 15px 0;
            color: #4a90e2;
            font-size: 18px;
            border-bottom: 2px solid #4a90e2;
            padding-bottom: 8px;
        }
        .control-group {
            margin: 12px 0;
        }
        .control-group label {
            display: block;
            margin-bottom: 6px;
            font-size: 13px;
            color: #b0b0b0;
        }
        button {
            background: linear-gradient(135deg, #4a90e2 0%, #357abd 100%);
            color: white;
            border: none;
            padding: 10px 16px;
            margin: 4px 2px;
            border-radius: 6px;
            cursor: pointer;
            font-size: 13px;
            transition: all 0.3s ease;
            pointer-events: all;
        }
        button:hover {
            transform: translateY(-2px);
            box-shadow: 0 4px 12px rgba(74, 144, 226, 0.4);
        }
        button:active {
            transform: translateY(0);
        }
        #info-panel {
            position: absolute;
            bottom: 20px;
            left: 20px;
            background: rgba(26, 26, 46, 0.95);
            padding: 15px;
            border-radius: 8px;
            font-size: 12px;
            color: #a0a0a0;
            pointer-events: all;
            min-width: 200px;
        }
        .info-item {
            margin: 6px 0;
            display: flex;
            justify-content: space-between;
        }
        .info-label {
            color: #707070;
        }
        .info-value {
            color: #4a90e2;
            font-weight: bold;
        }
        #loading {
            position: absolute;
            top: 50%;
            left: 50%;
            transform: translate(-50%, -50%);
            text-align: center;
            pointer-events: all;
        }
        .loader {
            border: 4px solid #2a2a4e;
            border-top: 4px solid #4a90e2;
            border-radius: 50%;
            width: 50px;
            height: 50px;
            animation: spin 1s linear infinite;
            margin: 0 auto 15px;
        }
        @keyframes spin {
            0% { transform: rotate(0deg); }
            100% { transform: rotate(360deg); }
        }
    </style>
</head>
<body>
    <canvas id="canvas"></canvas>
    <div id="ui-overlay">
        <div id="loading">
            <div class="loader"></div>
            <div>Loading 3D Model...</div>
        </div>
        <div id="controls" style="display: none;">
            <h3>KNIRV 3D Viewer</h3>
            <div class="control-group">
                <button onclick="resetCamera()">🔄 Reset View</button>
                <button onclick="toggleWireframe()">🔲 Wireframe</button>
            </div>
            <div class="control-group">
                <button onclick="toggleAnimation()">▶️ Animation</button>
                <button onclick="toggleLighting()">💡 Lighting</button>
            </div>
            <div class="control-group">
                <button onclick="takeScreenshot()">📸 Screenshot</button>
            </div>
        </div>
        <div id="info-panel">
            <div class="info-item">
                <span class="info-label">FPS:</span>
                <span class="info-value" id="fps">0</span>
            </div>
            <div class="info-item">
                <span class="info-label">Polygons:</span>
                <span class="info-value" id="polycount">0</span>
            </div>
            <div class="info-item">
                <span class="info-label">Renderer:</span>
                <span class="info-value">WebGL</span>
            </div>
        </div>
    </div>
    <script src="https://cdn.jsdelivr.net/npm/three@0.150.0/build/three.min.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/three@0.150.0/examples/js/loaders/GLTFLoader.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/three@0.150.0/examples/js/controls/OrbitControls.js"></script>
    <script>
        const scene = new THREE.Scene();
        scene.background = new THREE.Color(0x1a1a2e);

        const camera = new THREE.PerspectiveCamera(
            75,
            window.innerWidth / window.innerHeight,
            0.1,
            1000
        );

        const renderer = new THREE.WebGLRenderer({
            canvas: document.getElementById('canvas'),
            antialias: true,
            alpha: true
        });
        renderer.setSize(window.innerWidth, window.innerHeight);
        renderer.setPixelRatio(window.devicePixelRatio);
        renderer.shadowMap.enabled = true;
        renderer.shadowMap.type = THREE.PCFSoftShadowMap;
        renderer.toneMapping = THREE.ACESFilmicToneMapping;
        renderer.toneMappingExposure = 1.0;

        const controls = new THREE.OrbitControls(camera, renderer.domElement);
        controls.enableDamping = true;
        controls.dampingFactor = 0.05;
        controls.minDistance = 1;
        controls.maxDistance = 100;

        const loader = new THREE.GLTFLoader();
        let mixer, model, initialCameraPosition, initialControlsTarget;
        let lightingEnabled = true;

        loader.load('/asset.glb', (gltf) => {
            model = gltf.scene;
            scene.add(model);

            const box = new THREE.Box3().setFromObject(model);
            const center = box.getCenter(new THREE.Vector3());
            const size = box.getSize(new THREE.Vector3());
            const maxDim = Math.max(size.x, size.y, size.z);

            const distance = maxDim * 2;
            camera.position.set(
                center.x + distance * 0.7,
                center.y + distance * 0.5,
                center.z + distance * 0.7
            );
            controls.target.copy(center);
            controls.update();

            initialCameraPosition = camera.position.clone();
            initialControlsTarget = controls.target.clone();

            if (gltf.animations.length > 0) {
                mixer = new THREE.AnimationMixer(model);
                gltf.animations.forEach((clip) => {
                    mixer.clipAction(clip).play();
                });
            }

            let polycount = 0;
            model.traverse((child) => {
                if (child.isMesh) {
                    child.castShadow = true;
                    child.receiveShadow = true;
                    const count = child.geometry.index
                        ? child.geometry.index.count / 3
                        : child.geometry.attributes.position.count / 3;
                    polycount += count;
                }
            });
            document.getElementById('polycount').textContent = Math.floor(polycount).toLocaleString();

            document.getElementById('loading').style.display = 'none';
            document.getElementById('controls').style.display = 'block';
        }, undefined, (error) => {
            console.error('Error loading GLB:', error);
            document.getElementById('loading').innerHTML = '<div style="color: #f44;">❌ Load Error</div>';
        });

        const ambientLight = new THREE.AmbientLight(0xffffff, 0.5);
        scene.add(ambientLight);

        const directionalLight = new THREE.DirectionalLight(0xffffff, 1.0);
        directionalLight.position.set(5, 10, 7.5);
        directionalLight.castShadow = true;
        directionalLight.shadow.mapSize.width = 2048;
        directionalLight.shadow.mapSize.height = 2048;
        directionalLight.shadow.camera.near = 0.5;
        directionalLight.shadow.camera.far = 500;
        scene.add(directionalLight);

        const hemisphereLight = new THREE.HemisphereLight(0xffffbb, 0x080820, 0.3);
        scene.add(hemisphereLight);

        const clock = new THREE.Clock();
        let lastTime = Date.now();
        let frames = 0;

        function animate() {
            requestAnimationFrame(animate);
            const delta = clock.getDelta();
            if (mixer) mixer.update(delta);
            controls.update();
            renderer.render(scene, camera);

            frames++;
            const now = Date.now();
            if (now - lastTime >= 1000) {
                document.getElementById('fps').textContent = frames;
                frames = 0;
                lastTime = now;
            }
        }
        animate();

        window.addEventListener('resize', () => {
            camera.aspect = window.innerWidth / window.innerHeight;
            camera.updateProjectionMatrix();
            renderer.setSize(window.innerWidth, window.innerHeight);
        });

        function resetCamera() {
            if (initialCameraPosition && initialControlsTarget) {
                camera.position.copy(initialCameraPosition);
                controls.target.copy(initialControlsTarget);
                controls.update();
            }
        }

        function toggleWireframe() {
            if (model) {
                model.traverse((obj) => {
                    if (obj.isMesh) {
                        obj.material.wireframe = !obj.material.wireframe;
                    }
                });
            }
        }

        function toggleAnimation() {
            if (mixer) {
                mixer.timeScale = mixer.timeScale === 1 ? 0 : 1;
            }
        }

        function toggleLighting() {
            lightingEnabled = !lightingEnabled;
            ambientLight.intensity = lightingEnabled ? 0.5 : 0.1;
            directionalLight.intensity = lightingEnabled ? 1.0 : 0.2;
        }

        function takeScreenshot() {
            renderer.render(scene, camera);
            const dataURL = renderer.domElement.toDataURL('image/png');
            const link = document.createElement('a');
            link.download = 'knirv-3d-capture.png';
            link.href = dataURL;
            link.click();
        }
    </script>
</body>
</html>`
