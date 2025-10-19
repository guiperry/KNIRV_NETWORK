## Integrate Skyrim NIF Support with Three.js in React Application

This document outlines the steps to integrate Skyrim NIF support directly into your React application using Three.js, replacing the iframe approach.  This involves modifying the backend API, refactoring the `Viewport` component, and ensuring the `ViewportPage` uses the refactored component.

**Challenges & Considerations:**

*   **NIF Parsing on Server:** This is the most challenging aspect. You'll need a robust NIF parsing library on your Node.js backend. Options include:
    *   Finding Node.js bindings for C++ libraries like NifLib (requires build tools, potentially complex).
    *   Searching for pure JavaScript/TypeScript NIF parsers on npm (might be less mature or complete, especially for Skyrim specifics).
    *   Using an external tool/service to pre-convert NIF files to glTF if possible.
*   **NIF to glTF Conversion:** Once parsed, the NIF data (geometry, materials, textures, hierarchy) needs to be mapped to the glTF structure. This requires understanding both formats.
*   **Texture Handling:** NIF files reference textures (often `.dds` format) via relative paths. Your API needs to resolve these paths and include them correctly in the glTF. The client will need to load these textures. Browsers don't natively support DDS, so you might need the DDSLoader for Three.js or convert textures to PNG/JPG on the server during the conversion step.

**Step 1: Create a Server-Side API Endpoint (Conceptual)**

Let's assume you create `/pages/api/nif-model/[id].ts`. This endpoint needs to perform the NIF parsing and glTF conversion. *This requires a server-side NIF library which is not shown here but is crucial.*

```typescript
// /pages/api/nif-model/[id].ts (Conceptual - Requires actual NIF parsing/conversion logic)
import type { NextApiRequest, NextApiResponse } from 'next';
import path from 'path';
import fs from 'fs/promises'; // Or sync methods

// --- PLACEHOLDER: Import your actual NIF parser and glTF converter ---
// Example: import { parseNif, convertNifToGltf } from 'your-nif-library';

export default async function handler(req: NextApiRequest, res: NextApiResponse) {
    const { id } = req.query;

    if (typeof id !== 'string') {
        return res.status(400).json({ error: 'Invalid model ID' });
    }

    try {
        // --- 1. Locate NIF file (Adjust path as needed) ---
        const nifFilePath = path.resolve('./public/models/nif', `${id}.nif`); // Example path
        await fs.access(nifFilePath); // Check existence

        // --- 2. Parse NIF (Replace with actual library call) ---
        console.log(`Parsing NIF: ${nifFilePath}`);
        // const nifData = await parseNif(nifFilePath);
        // if (!nifData) throw new Error('Failed to parse NIF');
        console.warn("NIF parsing logic is a placeholder!"); // Placeholder warning

        // --- 3. Convert to glTF (Replace with actual conversion) ---
        console.log(`Converting NIF ${id} to glTF...`);
        // const gltfJson = await convertNifToGltf(nifData, {
        //    textureBasePath: '/models/textures/' // Base URL for client to find textures
        // });
        console.warn("NIF-to-glTF conversion logic is a placeholder!"); // Placeholder warning

        // --- 4. Send Placeholder glTF Response (FOR TESTING) ---
        // Replace this with your actual gltfJson once conversion works
        const placeholderGltf = {
            asset: { version: "2.0" },
            scenes: [{ nodes: [0] }],
            nodes: [{ mesh: 0 }],
            meshes: [{
                primitives: [{
                    attributes: { POSITION: 0 },
                    indices: 1
                }]
            }],
            buffers: [{ byteLength: 48, uri: "data:application/octet-stream;base64,AAAAAAAAAAAAAAAAAACAPwAAAAAAAAAAAAAAAAAAgD8AAAAAAAAAAAAAgD8AAAAAAACAPwAAAAA=" }],
            bufferViews: [
                { buffer: 0, byteOffset: 0, byteLength: 36, target: 34962 },
                { buffer: 0, byteOffset: 36, byteLength: 12, target: 34963 }
            ],
            accessors: [
                { bufferView: 0, byteOffset: 0, componentType: 5126, count: 3, type: "VEC3", max: [1, 1, 0], min: [0, 0, 0] },
                { bufferView: 1, byteOffset: 0, componentType: 5125, count: 3, type: "SCALAR" }
            ]
        };
        res.status(200).json(placeholderGltf); // Send placeholder glTF

    } catch (error: any) {
        console.error(`Error processing NIF model ${id}:`, error);
        if (error.code === 'ENOENT') {
            return res.status(404).json({ error: 'Model not found' });
        }
        res.status(500).json({ error: 'Failed to load or convert model', details: error.message });
    }
}
```

**Step 2: Refactor `components/Viewport.tsx`**

This component will now handle the direct Three.js rendering.

```typescript
import React, { useRef, useEffect, useState, useCallback } from 'react';
import * as THREE from 'three';
import { OrbitControls } from 'three/examples/jsm/controls/OrbitControls.js'; // Ensure correct import path
import { GLTFLoader } from 'three/examples/jsm/loaders/GLTFLoader.js'; // Ensure correct import path
// Optional: If handling DDS textures directly from NIF->glTF conversion
// import { DDSLoader } from 'three/examples/jsm/loaders/DDSLoader.js';

interface ViewportProps {
  modelId?: string; // Accept modelId directly
}

const Viewport: React.FC<ViewportProps> = ({ modelId }) => {
  const mountRef = useRef<HTMLDivElement>(null);
  const rendererRef = useRef<THREE.WebGLRenderer | null>(null);
  const sceneRef = useRef<THREE.Scene | null>(null);
  const cameraRef = useRef<THREE.PerspectiveCamera | null>(null);
  const controlsRef = useRef<OrbitControls | null>(null);
  const requestRef = useRef<number>(); // For animation frame
  const modelGroupRef = useRef<THREE.Group | null>(null); // To hold the loaded model

  const [isLoading, setIsLoading] = useState(false);
  const [hasError, setHasError] = useState<string | null>(null);

  // Cleanup function for Three.js resources
  const cleanupScene = useCallback(() => {
    cancelAnimationFrame(requestRef.current!);

    // Dispose controls
    if (controlsRef.current) {
        controlsRef.current.dispose();
        controlsRef.current = null;
    }

    // Dispose scene contents (geometry, material, textures)
    if (sceneRef.current) {
        sceneRef.current.traverse((object) => {
            if (object instanceof THREE.Mesh) {
                object.geometry?.dispose();
                if (Array.isArray(object.material)) {
                    object.material.forEach(material => material.dispose());
                } else if (object.material) {
                    object.material.dispose();
                    // Check and dispose textures
                    for (const key of Object.keys(object.material)) {
                        const value = object.material[key as keyof THREE.Material];
                        if (value instanceof THREE.Texture) {
                            value.dispose();
                        }
                    }
                }
            }
        });
        // Remove the main model group
        if (modelGroupRef.current) {
            sceneRef.current.remove(modelGroupRef.current);
            modelGroupRef.current = null;
        }
        sceneRef.current = null; // Clear scene reference
    }


    // Dispose renderer
    if (rendererRef.current) {
        rendererRef.current.dispose();
        // Check if the canvas is still a child and remove it
        if (mountRef.current && rendererRef.current.domElement.parentNode === mountRef.current) {
            mountRef.current.removeChild(rendererRef.current.domElement);
        }
        rendererRef.current = null;
    }

    cameraRef.current = null; // Clear camera reference
    console.log("Three.js scene cleaned up");
  }, []); // No dependencies needed for cleanup logic itself

  // Effect for initializing and loading
  useEffect(() => {
    if (!mountRef.current || !modelId) {
        cleanupScene(); // Clean up if no model or mount point
        setIsLoading(false);
        setHasError(null);
        return;
    }

    // --- Start Setup ---
    setIsLoading(true);
    setHasError(null);
    cleanupScene(); // Clean up previous instance before setting up new one
    const currentMount = mountRef.current;

    // Scene
    const scene = new THREE.Scene();
    scene.background = new THREE.Color(0x0a0e17);
    sceneRef.current = scene;

    // Camera
    const camera = new THREE.PerspectiveCamera(75, currentMount.clientWidth / currentMount.clientHeight, 0.1, 1000);
    camera.position.z = 5; // Initial position
    cameraRef.current = camera;

    // Renderer
    const renderer = new THREE.WebGLRenderer({ antialias: true });
    renderer.setSize(currentMount.clientWidth, currentMount.clientHeight);
    renderer.setPixelRatio(window.devicePixelRatio);
    // Make sure not to add the canvas multiple times on re-renders
    if (currentMount.children.length === 0) {
        currentMount.appendChild(renderer.domElement);
    }
    rendererRef.current = renderer;

    // Controls
    const controls = new OrbitControls(camera, renderer.domElement);
    controls.enableDamping = true;
    controlsRef.current = controls;

    // Lighting
    const ambientLight = new THREE.AmbientLight(0xffffff, 0.7);
    scene.add(ambientLight);
    const directionalLight = new THREE.DirectionalLight(0xffffff, 0.8);
    directionalLight.position.set(5, 10, 7.5);
    scene.add(directionalLight);

    // --- Load Model ---
    const loader = new GLTFLoader();

    // Optional: DDS Loader setup if textures are DDS and not pre-converted
    // const ddsLoader = new DDSLoader();
    // THREE.DefaultLoadingManager.addHandler(/\.dds$/i, ddsLoader);

    const modelUrl = `/api/nif-model/${modelId}`;
    console.log(`Loading model from: ${modelUrl}`);

    loader.load(
        modelUrl,
        (gltf) => {
            console.log(`Model ${modelId} loaded successfully.`);
            modelGroupRef.current = gltf.scene; // Store reference to the loaded model group
            scene.add(gltf.scene);

            // Auto-center and scale model
            try {
                const box = new THREE.Box3().setFromObject(gltf.scene);
                const center = box.getCenter(new THREE.Vector3());
                const size = box.getSize(new THREE.Vector3());
                const maxDim = Math.max(size.x, size.y, size.z);

                // Adjust camera distance based on model size and FOV
                const fov = camera.fov * (Math.PI / 180);
                let cameraZ = Math.abs(maxDim / 1.5 / Math.tan(fov / 2)); // Adjust divisor for distance
                cameraZ = Math.max(cameraZ, 1); // Ensure minimum distance

                camera.position.copy(center);
                // Offset camera slightly for a better initial view
                camera.position.x += size.x * 0.1;
                camera.position.y += size.y * 0.3;
                camera.position.z += cameraZ;

                camera.lookAt(center);
                controls.target.copy(center);
                controls.update();
                console.log("Model centered and scaled.");
            } catch(e) {
                console.error("Could not auto-center model:", e);
                // Fallback position if bounding box calculation fails
                camera.position.set(0, 1, 5);
                controls.target.set(0,0,0);
                controls.update();
            }

            setIsLoading(false);
        },
        undefined, // Progress callback (optional)
        (error) => {
            console.error(`Error loading model ${modelId}:`, error);
            // Try to provide more specific error info if available
            let errorMsg = `Failed to load model ${modelId}.`;
            if (error instanceof ErrorEvent) {
                 errorMsg += ` Network error or invalid response. Check API endpoint and NIF conversion.`;
            } else if (typeof error === 'string') {
                 errorMsg += ` Details: ${error}`;
            } else {
                 errorMsg += ` Check console for details.`;
            }
            setHasError(errorMsg);
            setIsLoading(false);
        }
    );

    // --- Render Loop ---
    const animate = () => {
        requestRef.current = requestAnimationFrame(animate);
        controls.update(); // Update controls for damping
        renderer.render(scene, camera);
    };
    animate();

    // --- Resize Handling ---
    const handleResize = () => {
        if (currentMount && rendererRef.current && cameraRef.current) {
            const width = currentMount.clientWidth;
            const height = currentMount.clientHeight;
            cameraRef.current.aspect = width / height;
            cameraRef.current.updateProjectionMatrix();
            rendererRef.current.setSize(width, height);
        }
    };
    window.addEventListener('resize', handleResize);

    // --- Cleanup on unmount or modelId change ---
    return () => {
        console.log(`Cleaning up Three.js for modelId: ${modelId}`);
        window.removeEventListener('resize', handleResize);
        cleanupScene();
    };

  }, [modelId, cleanupScene]); // Rerun effect if modelId changes

  // --- Render Component ---
  return (
    <div style={{ width: '100%', height: '100%', minHeight: '400px', position: 'relative', background: '#0a0e17' }}>
      {/* Mount point for Three.js canvas */}
      <div ref={mountRef} style={{ width: '100%', height: '100%' }} />

      {/* Loading Indicator */}
      {isLoading && (
        <div style={{ position: 'absolute', top: '50%', left: '50%', transform: 'translate(-50%, -50%)', color: '#fff', background: 'rgba(0,0,0,0.7)', padding: '15px 25px', borderRadius: '8px', zIndex: 10, textAlign: 'center' }}>
          Loading 3D Model...
        </div>
      )}

      {/* Error Message */}
      {hasError && (
        <div style={{ position: 'absolute', top: '50%', left: '50%', transform: 'translate(-50%, -50%)', color: '#fff', background: 'rgba(200,0,0,0.8)', padding: '15px 25px', borderRadius: '8px', zIndex: 10, textAlign: 'center', maxWidth: '80%' }}>
          <strong>Error:</strong> {hasError}
        </div>
      )}

      {/* Placeholder when no model is selected */}
      {!modelId && !isLoading && !hasError && (
         <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%', color: '#888', fontSize: '1.2em' }}>
            Select an object to view
         </div>
      )}
    </div>
  );
};

export default Viewport;
```

**Step 3: Verify `pages/viewport.tsx`**

This page already uses the `Viewport` component and passes the `id` from the router query. With the refactored `Viewport`, it should now work correctly, rendering the specified model directly with Three.js.

```typescript
import React from 'react';
import { useRouter } from 'next/router';
import Viewport from '../components/Viewport'; // Ensure this path is correct

const ViewportPage: React.FC = () => {
  const router = useRouter();
  const { id } = router.query; // Get the ID from the URL like /viewport?id=mymodel

  return (
    <div style={{
      width: '100%',
      height: '100vh', // Use viewport height
      background: '#0a0e17', // Match viewport background potentially
      display: 'flex',
      flexDirection: 'column',
      overflow: 'hidden' // Prevent potential scrollbars from layout shifts
    }}>
      <header style={{
        padding: '15px 20px',
        background: 'rgba(16, 24, 48, 0.7)', // Example header style
        borderBottom: '1px solid rgba(100, 130, 255, 0.2)',
        color: '#fff',
        flexShrink: 0 // Prevent header from shrinking
      }}>
        {/* You might want to make the title dynamic based on the loaded model later */}
        <h1>KNIRVCHAIN 3D Viewer {id ? `- ${id}` : ''}</h1>
      </header>

      {/* The Viewport component will take the remaining space */}
      <div style={{ flex: 1, position: 'relative' /* Needed for absolute positioned overlays */ }}>
        {/* Pass the modelId extracted from the router query */}
        {/* Ensure id is passed only when it's a valid string */}
        <Viewport modelId={typeof id === 'string' ? id : undefined} />
      </div>
    </div>
  );
};

export default ViewportPage;
```

**Next Steps:**

1.  **Install Dependencies:** `npm install three @types/three`
2.  **Implement Backend NIF Handling:** This is crucial. Research and integrate a suitable Node.js NIF parser and glTF converter. Replace the placeholder logic in `/pages/api/nif-model/[id].ts`.
3.  **Texture Path Handling:** Ensure your NIF-to-glTF converter correctly resolves texture paths and that those textures are served statically by your Next.js app (e.g., place them in the `/public` directory and adjust `textureBasePath` accordingly). Consider DDS conversion or using DDSLoader.
4.  **Testing:** Place a sample `.nif` file (and its textures) where the API endpoint expects it (e.g., `/public/models/nif/mymodel.nif`). Test loading it via the main page (`/`) or directly via `/viewport?id=mymodel`.
5.  **Error Handling:** Improve error reporting from the API and the Viewport component.
6.  **Refine Styling:** Adjust the CSS/styles as needed for the loading/error states and overall layout.




**I. Overview of Niftools Repository (https://github.com/niftools/)**

The Niftools repository is a collection of tools and libraries for working with NIF files. It's primarily written in C++ and Python, but we can leverage its components to some extent in a JavaScript context.

**II. Key Tools and Libraries in Niftools:**

*   **1. libniflib (C++ Library):**
    *   **Description:** The core NIF parsing and manipulation library. Provides C++ classes and functions for reading, writing, and modifying NIF files.
    *   **JavaScript Integration:**
        *   **Not Directly Usable:** C++ code cannot be directly executed in a JavaScript application (typically).
        *   **Potential Workarounds (Complex):**
            *   **WebAssembly (WASM):** Compile `libniflib` to WebAssembly using tools like Emscripten. This allows you to run C++ code in the browser (or Node.js). *This is a significant undertaking.* You would need to:
                *   Set up Emscripten.
                *   Configure `libniflib` to compile with Emscripten (likely requiring modifications to the build system).
                *   Write JavaScript code to interface with the WASM module.
                *   *This is a very complex solution and is only recommended if you have advanced knowledge of C++, Emscripten, and WebAssembly.*
            *   **Node.js Native Addon:** Create a Node.js native addon using C++. This involves:
                *   Writing a C++ wrapper around `libniflib` functions.
                *   Compiling the C++ code into a native addon (a `.node` file).
                *   Using Node.js's `require()` function to load the addon in your JavaScript code.
                *   *This is also complex, requiring C++ and Node.js native addon development expertise.* You would still need C++ experience and would need to know how to write C++ modules that the Javascript app can use.
        *   **Recommendation:** *Unless you have strong C++ and WASM/Node.js native addon skills, directly using `libniflib` is not practical for a JavaScript app.* Explore the Python options first!

*   **2. pyffi (Python Library):**
    *   **Description:** A Python library for reading, writing, and manipulating NIF files.  `pyffi` is built on top of `libniflib`.
    *   **JavaScript Integration:**
        *   **Python Bridge:** Use a library like `brython` (compiles Python to JavaScript) or a server-side Python execution environment (e.g., Flask, Django) with an API that your JavaScript app can call. The steps involved would be:
            1.  **Python Backend:** Create a Python backend that uses `pyffi` to perform NIF file manipulations.
            2.  **API Endpoint:** Expose an API endpoint (e.g., using Flask) that accepts NIF file data as input and returns the processed NIF file data as output.
            3.  **JavaScript Frontend:** In your JavaScript app, send HTTP requests to the Python backend API with the NIF file data.
            4.  **Data Handling:**  The Python backend processes the NIF file using `pyffi` and returns the result to the JavaScript app.
        *   **Pros:** Simpler than WASM or Node.js native addons. Leverage the existing `pyffi` library.
        *   **Cons:** Requires a Python server. Adds latency due to network requests. Not a pure JavaScript solution. You must keep the python deployment up to date with security patches.
        *   **Example (Conceptual):**

            **Python (Flask):**

```python
from flask import Flask, request, jsonify
from pyffi.formats.nif import NifFormat

app = Flask(__name__)

@app.route('/process_nif', methods=['POST'])
def process_nif():
    try:
        nif_data = request.files['nif_file'].read()  # Read from file upload

        # Load NIF using pyffi (example: reading only)
        nif_struct = NifFormat.Data()
        nif_struct.read(nif_data)

        #Modify the nif structure and re-export
        # example NifFormat.Data().write(modified_nif_structure)

        # Convert NIF back to bytes and send back
        return jsonify({'success': True, 'nif_data': "b64encoded NIF FILE DATA"})
    except Exception as e:
        return jsonify({'success': False, 'error': str(e)})

if __name__ == '__main__':
    app.run(debug=True)

```

            **JavaScript:**

```javascript
async function processNIF(nifFile) {
    const formData = new FormData();
    formData.append('nif_file', nifFile);

    try {
        const response = await fetch('http://localhost:5000/process_nif', {
            method: 'POST',
            body: formData
        });

        const data = await response.json();
        if (data.success) {
            console.log('NIF file processed successfully:', data.nif_data);

            return data.nif_data;
        } else {
            console.error('Error processing NIF file:', data.error);
            return null;
        }
    } catch (error) {
        console.error('Network error:', error);
        return null;
    }
}

// Example: Call the function with a File object
const fileInput = document.getElementById('nifFileInput');
fileInput.addEventListener('change', async () => {
    const nifFile = fileInput.files[0];
    if (nifFile) {
        const processedNIFData = await processNIF(nifFile);
       //Do somthing with the exported file data
    }
});
```

            **Notes:**

            *   Replace `"http://localhost:5000/process_nif"` with the actual URL of your Python API.
            *   Error handling and data conversion (e.g., base64 encoding) are simplified for clarity.

*   **3. Blender NIF Plugin:**
    *   **Description:** A Blender plugin for importing and exporting NIF files.
    *   **JavaScript Integration:**
        *   **Not Directly Usable:** Blender is a separate application, so you can't directly access it from JavaScript.
        *   **Potential Workaround:**
            *   **Blender Automation (Complex):** Use Blender's Python API to automate NIF import/export and other operations. You could then:
                *   Run Blender in the background using a command-line interface.
                *   Communicate with Blender using a socket or file-based IPC (Inter-Process Communication).
                *   *This is very complex and would require extensive knowledge of Blender's Python API.* It would be similar to running a python webserver for `pyffi` use, as the python program is external to your JavaScript application
        *   **Recommendation:** This is generally not practical for a web-based JavaScript app due to the complexity of automating Blender and the overhead of IPC.

*   **4. Command Line Tools:**
    *The niftools collection contains various command line tools, like niflyscope.
    *   **JavaScript Integration:**
        *   **Process execution in JavaScript (Complex):** Tools like niflyscope could, potentially, be run from a terminal using a `child_process` in Node.js or from the command line from web applications. This, like `pyffi` would require a Python back-end.
    *   **Recommendation:** This is generally not practical for a web-based JavaScript app due to the complexity of running and parsing from external tools.

**III. Summary and Recommendations:**

The most practical approach for manipulating NIF files in a JavaScript application is to:

1.  **Use `pyffi`:** Create a Python backend with a Flask API that handles NIF file processing.
2.  **Communicate via HTTP:** Send HTTP requests from your JavaScript frontend to the Python API.

**Example Workflow:**

1.  **User Uploads NIF File (JavaScript):**
    *   The user uploads a NIF file to your JavaScript app.
2.  **Send NIF to Python API (JavaScript):**
    *   The JavaScript app sends the NIF file data (as a `FormData` object or base64 encoded string) to the Python API.
3.  **Process NIF (Python):**
    *   The Python API receives the NIF data.
    *   It uses `pyffi` to read, modify, or write the NIF file.
4.  **Return Result (Python):**
    *   The Python API returns the processed NIF file data (or a success/error message) to the JavaScript app.
5.  **Display Result (JavaScript):**
    *   The JavaScript app receives the result and displays it to the user (e.g., shows the modified NIF file, displays metadata, etc.).

**Challenges:**

*   **Python Server Requirement:** You need to deploy and maintain a Python server.
*   **Network Latency:** There will be network latency when communicating between the JavaScript app and the Python server.
*   **Complexity:** Implementing the Python API and handling data conversion can be complex.
*   **Security:** Ensure that your Python API is secure to prevent unauthorized access or malicious code execution.

By following this approach, you can leverage the power of the Niftools ecosystem to manipulate NIF files in a JavaScript application, even though you can't directly run C++ or Python code in the browser. Remember to focus on creating a well-designed API, handling errors gracefully, and securing your Python backend.
