// Copyright 2026 KNIRV-NEXUS
// SPDX-License-Identifier: GPL-3.0-or-later

package viewport

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"sync"
)

// ObjectType represents the type of object being served
type ObjectType string

const (
	ObjectTypeWebApp     ObjectType = "webapp"
	ObjectTypeAPI        ObjectType = "api"
	ObjectTypeBlockchain ObjectType = "blockchain"
	ObjectTypeOracle     ObjectType = "oracle"
	ObjectTypeModel      ObjectType = "model"
	ObjectType3D         ObjectType = "3d"
	ObjectTypeP2P        ObjectType = "p2p"
	ObjectTypeFile       ObjectType = "file"
	ObjectTypeCustom     ObjectType = "custom"
)

// ViewportProxy manages browser access to a container with content-aware rendering
type ViewportProxy struct {
	containerID    string
	objectType     ObjectType
	containerPorts map[string]int
	httpProxy      *httputil.ReverseProxy
	webrtcPeer     *WebRTCPeer
	vncSession     *VNCSession
	glbRenderer    *GLBRenderer
	renderers      []string // Enabled renderers: http, webrtc, webgl, vnc
	running        bool
	mu             sync.RWMutex
}

// NewViewportProxy creates a new viewport proxy for a container
func NewViewportProxy(containerID string, objectType ObjectType, ports map[string]int, renderers []string) *ViewportProxy {
	return &ViewportProxy{
		containerID:    containerID,
		objectType:     objectType,
		containerPorts: ports,
		renderers:      renderers,
		running:        false,
	}
}

// Start initializes the viewport proxy with content-aware renderers
func (vp *ViewportProxy) Start(ctx context.Context) error {
	vp.mu.Lock()
	defer vp.mu.Unlock()

	if vp.running {
		return fmt.Errorf("viewport proxy already running")
	}

	// Initialize HTTP reverse proxy
	targetPort := vp.selectPort("")
	if targetPort > 0 {
		vp.httpProxy = &httputil.ReverseProxy{
			Director: func(req *http.Request) {
				req.URL.Scheme = "http"
				req.URL.Host = fmt.Sprintf("localhost:%d", targetPort)
			},
		}
	}

	// Initialize renderers based on object type
	for _, renderer := range vp.renderers {
		switch renderer {
		case "webrtc":
			if err := vp.initializeWebRTC(); err != nil {
				log.Printf("Warning: WebRTC initialization failed: %v", err)
			}

		case "webgl":
			if vp.objectType == ObjectType3D {
				glbRenderer := NewGLBRenderer()
				vp.glbRenderer = glbRenderer
			}

		case "vnc":
			vncSession, err := NewVNCSession(vp.containerID)
			if err != nil {
				log.Printf("Warning: VNC session creation failed: %v", err)
			} else {
				vp.vncSession = vncSession
			}
		}
	}

	vp.running = true

	log.Printf("Viewport proxy started for container %s (type: %s, renderers: %v)",
		vp.containerID, vp.objectType, vp.renderers)

	return nil
}

// Stop stops the viewport proxy
func (vp *ViewportProxy) Stop() error {
	vp.mu.Lock()
	defer vp.mu.Unlock()

	if !vp.running {
		return nil
	}

	// Cleanup WebRTC peer
	if vp.webrtcPeer != nil {
		if err := vp.webrtcPeer.Close(); err != nil {
			log.Printf("Error closing WebRTC peer: %v", err)
		}
	}

	// Cleanup VNC session
	if vp.vncSession != nil {
		if err := vp.vncSession.Close(); err != nil {
			log.Printf("Error closing VNC session: %v", err)
		}
	}

	// Cleanup GLB renderer
	if vp.glbRenderer != nil {
		if err := vp.glbRenderer.Close(); err != nil {
			log.Printf("Error closing GLB renderer: %v", err)
		}
	}

	vp.running = false

	log.Printf("Viewport proxy stopped for container %s", vp.containerID)

	return nil
}

// HandleHTTP handles HTTP requests to the container
func (vp *ViewportProxy) HandleHTTP(w http.ResponseWriter, r *http.Request) {
	vp.mu.RLock()
	defer vp.mu.RUnlock()

	if !vp.running {
		http.Error(w, "viewport proxy not running", http.StatusServiceUnavailable)
		return
	}

	// Add CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("X-Object-Type", string(vp.objectType))
	w.Header().Set("X-Container-ID", vp.containerID)

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Route to appropriate handler based on object type
	switch vp.objectType {
	case ObjectType3D:
		vp.handle3DObject(w, r)
	default:
		if vp.httpProxy != nil {
			vp.httpProxy.ServeHTTP(w, r)
		} else {
			http.Error(w, "no HTTP proxy available", http.StatusServiceUnavailable)
		}
	}
}

// handle3DObject handles requests for 3D GLB objects
func (vp *ViewportProxy) handle3DObject(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/asset.glb" {
		// Serve the GLB file directly
		vp.serveGLBAsset(w, r)
	} else if r.URL.Path == "/metadata" {
		// Serve asset metadata
		vp.serveGLBMetadata(w, r)
	} else {
		// Serve the WebGL viewer interface
		vp.serveGLBViewer(w, r)
	}
}

// serveGLBAsset serves the GLB file with proper headers
func (vp *ViewportProxy) serveGLBAsset(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "model/gltf-binary")
	w.Header().Set("Cache-Control", "public, max-age=31536000")

	// Proxy to container's asset endpoint if HTTP proxy available
	if vp.httpProxy != nil {
		vp.httpProxy.ServeHTTP(w, r)
	} else {
		http.Error(w, "no asset available", http.StatusNotFound)
	}
}

// serveGLBMetadata serves asset metadata as JSON
func (vp *ViewportProxy) serveGLBMetadata(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if vp.glbRenderer != nil {
		metadata := vp.glbRenderer.GetMetadata()
		w.Write([]byte(fmt.Sprintf(`{"polycount": %d, "format": "%s"}`,
			metadata.Polycount, metadata.Format)))
	} else {
		w.Write([]byte(`{"error": "no renderer available"}`))
	}
}

// serveGLBViewer serves the WebGL viewer interface
func (vp *ViewportProxy) serveGLBViewer(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(glbViewerHTML))
}

// initializeWebRTC initializes WebRTC peer connection
func (vp *ViewportProxy) initializeWebRTC() error {
	peer, err := NewWebRTCPeer()
	if err != nil {
		return fmt.Errorf("WebRTC peer creation failed: %w", err)
	}
	vp.webrtcPeer = peer

	return nil
}

// selectPort selects the appropriate container port based on request path
func (vp *ViewportProxy) selectPort(path string) int {
	// Return first available port
	for _, port := range vp.containerPorts {
		return port
	}
	return 0
}

// GetStatus returns the viewport proxy status
func (vp *ViewportProxy) GetStatus() map[string]interface{} {
	vp.mu.RLock()
	defer vp.mu.RUnlock()

	return map[string]interface{}{
		"container_id": vp.containerID,
		"object_type":  vp.objectType,
		"running":      vp.running,
		"renderers":    vp.renderers,
	}
}

// GLB Viewer HTML template
const glbViewerHTML = `
<!DOCTYPE html>
<html>
<head>
    <title>3D Object Viewer - KNIRV</title>
    <style>
        body { margin: 0; overflow: hidden; font-family: Arial, sans-serif; background: #1a1a1a; }
        #canvas { width: 100vw; height: 100vh; }
        #controls {
            position: absolute;
            top: 10px;
            left: 10px;
            background: rgba(0,0,0,0.8);
            color: white;
            padding: 15px;
            border-radius: 8px;
            box-shadow: 0 4px 6px rgba(0,0,0,0.3);
        }
        button {
            background: #4a90e2;
            color: white;
            border: none;
            padding: 8px 16px;
            margin: 4px;
            border-radius: 4px;
            cursor: pointer;
            font-size: 14px;
        }
        button:hover { background: #357abd; }
        #info {
            position: absolute;
            bottom: 10px;
            left: 10px;
            background: rgba(0,0,0,0.8);
            color: #aaa;
            padding: 10px;
            border-radius: 4px;
            font-size: 12px;
        }
    </style>
</head>
<body>
    <div id="controls">
        <h3 style="margin: 0 0 10px 0;">KNIRV Object Viewer</h3>
        <button onclick="resetCamera()">Reset Camera</button>
        <button onclick="toggleWireframe()">Wireframe</button>
        <button onclick="toggleAnimation()">Animation</button>
    </div>
    <div id="info">
        <div>FPS: <span id="fps">0</span></div>
        <div>Polycount: <span id="polycount">0</span></div>
    </div>
    <canvas id="canvas"></canvas>
    <script src="https://cdn.jsdelivr.net/npm/three@0.150.0/build/three.min.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/three@0.150.0/examples/js/loaders/GLTFLoader.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/three@0.150.0/examples/js/controls/OrbitControls.js"></script>
    <script>
        const scene = new THREE.Scene();
        scene.background = new THREE.Color(0x1a1a1a);

        const camera = new THREE.PerspectiveCamera(75, window.innerWidth / window.innerHeight, 0.1, 1000);
        const renderer = new THREE.WebGLRenderer({
            canvas: document.getElementById('canvas'),
            antialias: true,
            alpha: true
        });
        renderer.setSize(window.innerWidth, window.innerHeight);
        renderer.shadowMap.enabled = true;
        renderer.shadowMap.type = THREE.PCFSoftShadowMap;

        const controls = new THREE.OrbitControls(camera, renderer.domElement);
        controls.enableDamping = true;
        controls.dampingFactor = 0.05;
        camera.position.set(0, 2, 5);

        const loader = new THREE.GLTFLoader();
        let mixer;
        let model;

        loader.load('/asset.glb', (gltf) => {
            model = gltf.scene;
            scene.add(model);

            // Calculate bounding box and center camera
            const box = new THREE.Box3().setFromObject(model);
            const center = box.getCenter(new THREE.Vector3());
            const size = box.getSize(new THREE.Vector3());
            const maxDim = Math.max(size.x, size.y, size.z);
            camera.position.set(center.x, center.y + maxDim, center.z + maxDim * 1.5);
            controls.target.copy(center);

            // Setup animations
            if (gltf.animations.length > 0) {
                mixer = new THREE.AnimationMixer(model);
                gltf.animations.forEach((clip) => {
                    const action = mixer.clipAction(clip);
                    action.play();
                });
            }

            // Count polygons
            let polycount = 0;
            model.traverse((child) => {
                if (child.isMesh) {
                    polycount += child.geometry.index ? child.geometry.index.count / 3 : child.geometry.attributes.position.count / 3;
                }
            });
            document.getElementById('polycount').textContent = Math.floor(polycount).toLocaleString();
        }, undefined, (error) => {
            console.error('Error loading GLB:', error);
            document.getElementById('info').innerHTML += '<br><span style="color: #f44;">Load Error</span>';
        });

        // Lighting
        const ambientLight = new THREE.AmbientLight(0xffffff, 0.6);
        scene.add(ambientLight);

        const directionalLight = new THREE.DirectionalLight(0xffffff, 0.8);
        directionalLight.position.set(5, 10, 7.5);
        directionalLight.castShadow = true;
        directionalLight.shadow.mapSize.width = 2048;
        directionalLight.shadow.mapSize.height = 2048;
        scene.add(directionalLight);

        const hemisphereLight = new THREE.HemisphereLight(0xffffff, 0x444444, 0.4);
        scene.add(hemisphereLight);

        // Animation loop
        const clock = new THREE.Clock();
        let lastTime = Date.now();
        let frames = 0;

        function animate() {
            requestAnimationFrame(animate);

            const delta = clock.getDelta();
            if (mixer) mixer.update(delta);

            controls.update();
            renderer.render(scene, camera);

            // FPS counter
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
            camera.position.set(0, 2, 5);
            controls.target.set(0, 0, 0);
            controls.update();
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
    </script>
</body>
</html>
`
