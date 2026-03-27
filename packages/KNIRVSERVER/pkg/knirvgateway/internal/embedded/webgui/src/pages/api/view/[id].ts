// pages/api/view/[id].ts
import { NextApiRequest, NextApiResponse } from 'next';

// Use the environment variable for the backend URL
const BACKEND_URL = process.env.NEXT_PUBLIC_BACKEND_URL || 'http://localhost:3001'; // Default fallback

export default async function handler(req: NextApiRequest, res: NextApiResponse) {
  if (req.method !== 'GET') {
    res.setHeader('Allow', ['GET']);
    return res.status(405).json({ message: 'Method Not Allowed' });
  }

  const { id } = req.query;

  if (!id || typeof id !== 'string') {
    return res.status(400).json({ message: 'Invalid or missing object ID' });
  }

  try {
    // Fetch specific object details from the backend server API endpoint
    const response = await fetch(`${BACKEND_URL}/api/objects/${id}`, { // <-- Use BACKEND_URL and id
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        // Add any other necessary headers
      },
    });

    if (!response.ok) {
       // Log the error from the backend response for better debugging
       const errorText = await response.text();
       console.error(`Backend server error fetching object ${id}: ${response.status} ${response.statusText}`, errorText);

       // Handle specific errors like 404 Not Found from the backend
       if (response.status === 404) {
         return res.status(404).json({ message: `Object with ID "${id}" not found.` });
       }
       // Return a specific error status based on the backend response if desired, or a generic 502
       return res.status(response.status < 500 ? response.status : 502).json({ message: `Failed to fetch object ${id} from backend: ${response.statusText}` });
    }

    // Assuming the backend returns JSON data for the object, including its name and potentially the model URL
    const objectData = await response.json();

    // Log the object data for debugging
    console.log('Object data from backend:', objectData);

    // --- HTML Viewer Generation ---
    // Get the model URL from the object data
    // Use the file_path directly if available, otherwise use default
    let modelUrl = '/assets/models/cube.gltf'; // Default fallback

    if (objectData && objectData.file_path) {
      modelUrl = objectData.file_path;
    }

    console.log('Using model URL:', modelUrl);

    const viewerHtml = `
      <!DOCTYPE html>
      <html lang="en">
      <head>
        <meta charset="UTF-8">
        <meta name="viewport" content="width=device-width, initial-scale=1.0">
        <title>KNIRVCHAIN - ${objectData.name || 'Object'}</title>
        <script type="importmap">
          {
            "imports": {
              "three": "https://unpkg.com/three@0.175.0/build/three.module.js",
              "three/addons/": "https://unpkg.com/three@0.175.0/examples/jsm/"
            }
          }
        </script>
        <style>
          /* Global Styles */
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

          /* UI Elements */
          .ui-container {
            position: absolute;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
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

          /* Info Panel */
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

          /* Loading Indicator */
          #loading {
            position: absolute;
            top: 50%;
            left: 50%;
            transform: translate(-50%, -50%);
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

          /* Chain Info */
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

          /* Help Tooltip */
          .help-tooltip {
            position: absolute;
            bottom: 20px;
            right: 20px;
            background: rgba(16, 24, 48, 0.7);
            backdrop-filter: blur(10px);
            border: 1px solid rgba(100, 130, 255, 0.2);
            border-radius: 50%;
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

          /* Help Panel */
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

          /* Responsive Adjustments */
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
              width: calc(100% - 40px);
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
              <span class="object-title">${objectData.name || 'Object'}</span>
              <span class="object-id">ID: ${id}</span>
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
                <div class="info-value">${objectData.name || 'Unknown'}</div>
              </div>
              <div class="info-row">
                <div class="info-label">ID:</div>
                <div class="info-value">${id}</div>
              </div>
              <div class="info-row">
                <div class="info-label">Type:</div>
                <div class="info-value">${objectData.object_type || objectData.asset_type || 'Unknown'}</div>
              </div>
              <div class="info-row">
                <div class="info-label">Author:</div>
                <div class="info-value">${objectData.data?.author || 'Unknown'}</div>
              </div>
              <div class="info-row">
                <div class="info-label">Created:</div>
                <div class="info-value">${objectData.created_at ? new Date(objectData.created_at).toLocaleString() : 'Unknown'}</div>
              </div>
              <div class="info-row">
                <div class="info-label">License:</div>
                <div class="info-value">${objectData.data?.license || 'Unknown'}</div>
              </div>
            </div>
          </div>

          <div class="chain-info">
            <div class="chain-header">Blockchain Data</div>
            <div class="chain-data">
              <div class="chain-row">
                <div class="chain-label">TX Hash:</div>
                <div class="chain-value">${objectData.transaction ? objectData.transaction.substring(0, 10) + '...' : 'Unknown'}</div>
              </div>
              <div class="chain-row">
                <div class="chain-label">Block:</div>
                <div class="chain-value">#${Math.floor(Math.random() * 1000)}</div>
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
          // Use backticks for template literals in JavaScript
          const modelToLoad = \`\${modelUrl}\`; // Use the model URL determined above
          console.log("Loading model from:", modelToLoad);

          // Add error handling for the case when the model file doesn't exist
          window.addEventListener('error', function(e) {
            console.error('Error during model loading:', e.message);
            // Display error message in the scene
            const errorText = document.createElement('div');
            errorText.style.position = 'absolute';
            errorText.style.top = '50%';
            errorText.style.left = '50%';
            errorText.style.transform = 'translate(-50%, -50%)';
            errorText.style.color = '#ff5555';
            errorText.style.background = 'rgba(0,0,0,0.7)';
            errorText.style.padding = '20px';
            errorText.style.borderRadius = '5px';
            errorText.style.fontFamily = 'Arial, sans-serif';
            errorText.style.fontSize = '16px';
            errorText.style.textAlign = 'center';
            errorText.style.zIndex = '1000';
            errorText.innerHTML = 'Error loading 3D model:<br>' + e.message;
            document.body.appendChild(errorText);
          });

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
              loadingElement.textContent = \`Loading model: \${percentLoaded}%\`;
              console.log((xhr.loaded / xhr.total * 100) + '% loaded');
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

              // Add error text
              const textDiv = document.createElement('div');
              textDiv.style.position = 'absolute';
              textDiv.style.top = '60%';
              textDiv.style.left = '50%';
              textDiv.style.transform = 'translate(-50%, -50%)';
              textDiv.style.color = '#ff6666';
              textDiv.style.fontFamily = 'monospace';
              textDiv.style.fontSize = '14px';
              textDiv.style.padding = '10px';
              textDiv.style.background = 'rgba(0,0,0,0.7)';
              textDiv.style.borderRadius = '5px';
              textDiv.textContent = 'Failed to load model: ' + error.message;
              document.body.appendChild(textDiv);
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

          // Add some visual effects - particles in the background
          function addParticles() {
            const particlesGeometry = new THREE.BufferGeometry();
            const particlesCount = 1000;

            const posArray = new Float32Array(particlesCount * 3);

            for(let i = 0; i < particlesCount * 3; i++) {
              // Random positions in a sphere
              posArray[i] = (Math.random() - 0.5) * 50;
            }

            particlesGeometry.setAttribute('position', new THREE.BufferAttribute(posArray, 3));

            const particlesMaterial = new THREE.PointsMaterial({
              size: 0.05,
              color: 0x4080ff,
              transparent: true,
              opacity: 0.5,
              sizeAttenuation: true
            });

            const particlesMesh = new THREE.Points(particlesGeometry, particlesMaterial);
            scene.add(particlesMesh);

            // Animate particles
            function animateParticles() {
              particlesMesh.rotation.x += 0.0001;
              particlesMesh.rotation.y += 0.0001;
            }

            return animateParticles;
          }

          const animateParticles = addParticles();

          // Update animation loop to include particles
          function animate() {
            requestAnimationFrame(animate);
            controls.update();
            animateParticles();
            renderer.render(scene, camera);
          }

          animate();
        </script>
      </body>
      </html>
    `;

    res.setHeader('Content-Type', 'text/html');
    return res.status(200).send(viewerHtml);

  } catch (error: unknown) {
    console.error(`Error fetching object ${id} via Next.js API route:`, error);
    // Return a 500 Internal Server Error if the fetch itself fails or HTML generation fails
    const message = error instanceof Error ? error.message : 'Unknown error occurred';
    return res.status(500).json({ message: `Failed to process request for object ${id}: ${message}` });
  }
}