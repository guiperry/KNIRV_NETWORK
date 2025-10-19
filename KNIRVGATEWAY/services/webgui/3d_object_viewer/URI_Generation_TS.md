```typescript jsx
// components/3DObjectViewer.tsx
'use client'; // Client Component - Necessary for react-three-fiber and browser interactions

import React, { Suspense, useEffect, useRef, useState } from 'react';
import * as THREE from 'three';
import { Canvas, useFrame, useLoader } from '@react-three/fiber';
import { OrbitControls } from '@react-three/drei';

interface AssetMetadata {
    assetID: string;
    author?: string;
    version?: string;
    license?: string;
    contentLocation: string; // Location to fetch the 3D model (e.g., URL, IPFS CID)
}

interface ModelProps {
    url: string;
}

function Model({ url }: ModelProps) {
    const gltf = useLoader(THREE.GLTFLoader, url);
    return <primitive object={gltf.scene} />;
}

const LoadingFallback = () => (
    <mesh>
        <sphereGeometry args={[1, 32, 32]} />
        <meshBasicMaterial color="orange" wireframe />
    </mesh>
);

interface ThreeDObjectViewerProps {
    assetMetadata: AssetMetadata;
}

const ThreeDObjectViewer: React.FC<ThreeDObjectViewerProps> = ({ assetMetadata }) => {
    const [modelUrl, setModelUrl] = useState<string | null>(null);

    useEffect(() => {
        // In a real app, you might need to resolve the contentLocation
        // If it's an IPFS CID, you might use a gateway, etc.
        // For this example, we assume contentLocation is a direct URL to a GLTF/GLB model.
        setModelUrl(assetMetadata.contentLocation);
    }, [assetMetadata]);

    if (!modelUrl) {
        return <div>Loading 3D Object Viewer...</div>; // Initial loading state
    }

    return (
        <div style={{ width: '100%', height: '500px' }}> {/* Adjust height as needed */}
            <Canvas camera={{ position: [3, 3, 3] }}>
                <ambientLight intensity={0.5} />
                <directionalLight position={[-1, 1, 1]} intensity={1} />
                <Suspense fallback={<LoadingFallback />}>
                    <Model url={modelUrl} />
                    <OrbitControls />
                </Suspense>
            </Canvas>
            <div>
                <p>Asset ID: {assetMetadata.assetID}</p>
                {assetMetadata.author && <p>Author: {assetMetadata.author}</p>}
                {assetMetadata.version && <p>Version: {assetMetadata.version}</p>}
                {assetMetadata.license && <p>License: {assetMetadata.license}</p>}
            </div>
        </div>
    );
};

export default ThreeDObjectViewer;
```

```typescript jsx
// pages/index.tsx
import React, { useEffect, useState } from 'react';
import ThreeDObjectViewer from '../components/3DObjectViewer';

interface AssetMetadata {
    assetID: string;
    author?: string;
    version?: string;
    license?: string;
    contentLocation: string;
}

const parseNRNURI = (uriString: string): AssetMetadata | null => {
    try {
        const url = new URL(uriString);
        if (url.protocol !== 'nrn:') {
            return null;
        }

        const assetID = url.pathname.substring(1); // Remove leading slash
        const params = url.searchParams;

        const assetMetadata: AssetMetadata = {
            assetID: assetID,
            author: params.get('author') || undefined,
            version: params.get('version') || undefined,
            license: params.get('license') || undefined,
            contentLocation: params.get('contentLocation') || '', // You'd likely need to define contentLocation in URI or fetch via RPC
        };

        // In this example, we expect contentLocation as a query parameter for simplicity
        if (!assetMetadata.contentLocation) {
             console.error("contentLocation is missing in URI");
             return null;
        }


        return assetMetadata;
    } catch (error) {
        console.error("Error parsing NRN URI:", error);
        return null;
    }
};

const generateNRNURI = (metadata: AssetMetadata): string => {
    const baseURI = `nrn://usercontent.nrn/${metadata.assetID}`; // Authority could be dynamic
    const params = new URLSearchParams();
    if (metadata.author) params.append('author', metadata.author);
    if (metadata.version) params.append('version', metadata.version);
    if (metadata.license) params.append('license', metadata.license);
    if (metadata.contentLocation) params.append('contentLocation', metadata.contentLocation); // Include contentLocation in URI for this example

    return `${baseURI}?${params.toString()}`;
};


const HomePage: React.FC = () => {
    const [currentAssetMetadata, setCurrentAssetMetadata] = useState<AssetMetadata | null>(null);
    const [generatedUri, setGeneratedUri] = useState<string>('');

    useEffect(() => {
        // Check if the page was loaded with a custom URI (deep linking)
        if (typeof window !== 'undefined') {
            const currentUrl = window.location.href;
            const nrnUriPrefix = 'nrn://';

            if (currentUrl.startsWith(nrnUriPrefix)) {
                const parsedMetadata = parseNRNURI(currentUrl);
                if (parsedMetadata) {
                    setCurrentAssetMetadata(parsedMetadata);
                } else {
                    console.error("Invalid NRN URI in URL:", currentUrl);
                    // Handle invalid URI case (e.g., display an error message)
                }
                // To prevent browser from trying to "navigate" again if URI is in URL hash, clear hash
                window.location.hash = ''; //optional, depends on your routing strategy
            }
        }
    }, []);

    const handleGenerateUriExample = () => {
        const exampleMetadata: AssetMetadata = {
            assetID: 'example-asset-123',
            author: 'TestUser',
            version: 'v1',
            license: 'CC-BY-NC',
            contentLocation: '/models/Duck.glb', // Example relative path in public directory - adjust as needed
        };
        setGeneratedUri(generateNRNURI(exampleMetadata));
    };


    return (
        <div>
            <h1>3D Asset Viewer</h1>

            <div>
                <h2>Generate NRN URI Example</h2>
                <button onClick={handleGenerateUriExample}>Generate Example URI</button>
                {generatedUri && (
                    <p>Generated URI: <a href={generatedUri}>{generatedUri}</a></p>
                )}
            </div>


            {currentAssetMetadata && (
                <div>
                    <h2>Viewing 3D Asset from NRN URI</h2>
                    <ThreeDObjectViewer assetMetadata={currentAssetMetadata} />
                </div>
            )}

            {!currentAssetMetadata && (
                <p>No NRN URI detected in URL. Generate an example URI or open the application with a valid NRN URI.</p>
            )}
        </div>
    );
};

export default HomePage;
```

**To Run this Example:**

1.  **Create a Next.js project:** If you don't have one already, create a new Next.js project using `npx create-next-app my-3d-viewer`.
2.  **Install dependencies:**
    ```bash
    npm install three @react-three/fiber @react-three/drei
    ```
3.  **Create components and pages:**
    *   Create a `components` folder and add `3DObjectViewer.tsx`.
    *   Replace the content of `pages/index.tsx` with the code above.
4.  **Add a 3D Model:**
    *   Download a GLTF or GLB model (e.g., search for "free GLTF models" or use the example Duck.glb from three.js examples).
    *   Place the model file (e.g., `Duck.glb`) in the `public/models` directory in your Next.js project. If using Duck.glb, you can find it in the three.js examples repository or use any other simple gltf model.  Adjust `contentLocation` accordingly.
5.  **Run the development server:** `npm run dev`

**Testing the URI Scheme and Deep Linking:**

1.  **Run the Next.js dev server** (e.g., `npm run dev`).  It will typically be at `http://localhost:3000`.
2.  **Generate an Example URI:** Click the "Generate Example URI" button on the page. This will display a `nrn://` URI.
3.  **Manually Enter URI in Browser:**
    *   Copy the generated `nrn://` URI.
    *   Open a new browser tab or window.
    *   Paste the `nrn://` URI into the address bar and press Enter.

    *Since browsers don't natively handle `nrn://` schemes to launch applications directly (OS registration needed for that), you'll likely need to *replace* `nrn://` with `http://localhost:3000/?uri=` or similar prefix to get the browser to navigate to your Next.js app and pass the `nrn://` URI as a parameter in the URL.*  However, for this example we are directly using the `nrn://` as the base url for the Next.js app itself.

    *A more direct way to test (without OS registration) within the browser is to construct a URL like:*
    ```
    http://localhost:3000/#nrn://usercontent.nrn/example-asset-123?author=TestUser&version=v1&license=CC-BY-NC&contentLocation=/models/Duck.glb
    ```
    And then modify the `useEffect` in `pages/index.tsx` to check `window.location.hash` instead of `window.location.href` and parse the URI from the hash (removing the `#`).  This is a simpler way to simulate deep linking for browser testing without OS registration. The current code expects the full `nrn://` URI to be the *entire* URL in the address bar in order to be directly parsed by `URL` constructor.

**Important Considerations and Next Steps:**

*   **OS-Level URI Registration:**  For a true desktop application or video game, you would need to implement the OS-specific URI scheme registration as discussed previously (using Go or platform-native methods during installation).  This TypeScript/Next.js code *cannot* do OS-level registration directly.
*   **Backend RPC Integration:** In a real application, you would replace the `contentLocation` query parameter with a mechanism to fetch the `contentLocation` (and other asset details) from your backend RPC node, using the `assetID` from the parsed `nrn://` URI.
*   **Error Handling:** Add more robust error handling in `parseNRNURI` and when fetching 3D models.
*   **Security:** Consider security implications, especially if you are handling user-generated content or access control.
*   **3D Model Loading and Rendering:**  This example uses a simple GLTF loader. For more complex scenes or formats, you might need to use more advanced techniques and libraries.
*   **Video Game Integration:**  If you want to integrate this with a separate native video game application, the inter-application communication and URI handling would be significantly more complex and platform-dependent. You might need to use OS-specific APIs or middleware for inter-process communication. For a web-based game, you could potentially embed the Next.js 3D viewer component within the game's UI if both are in a web context.

This comprehensive example provides a foundation for URI generation, parsing, and 3D rendering within a Next.js browser application. Remember to adapt and extend it based on the specific requirements of your project, especially regarding backend integration, OS-level URI registration (if needed), and the nature of your video game application.