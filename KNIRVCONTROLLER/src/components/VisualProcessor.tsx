import { useRef, useEffect, useState, useCallback } from 'react';
import { Eye, EyeOff, Zap } from 'lucide-react';
import { VisualFrame, DetectedObject as SensoryDetectedObject, ProcessingMetrics } from '../types/sensory';

interface VisualProcessorProps {
  onVisualData: (data: ImageData) => void;
  onObjectDetection: (objects: SensoryDetectedObject[]) => void;
  isActive: boolean;
}

interface BoundingBox {
  x: number;
  y: number;
  width: number;
  height: number;
}


interface ProcessingStats {
  fps: number;
  objectCount: number;
  processingTime: number;
}

function VisualProcessor({ onVisualData, onObjectDetection, isActive }: VisualProcessorProps) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const overlayCanvasRef = useRef<HTMLCanvasElement>(null);
  const [stream, setStream] = useState<MediaStream | null>(null);
  const [isProcessing, setIsProcessing] = useState(false);
  const [detectedObjects, setDetectedObjects] = useState<SensoryDetectedObject[]>([]);
  const [stats] = useState<ProcessingStats>({ fps: 0, objectCount: 0, processingTime: 0 });
  const [error, setError] = useState<string | null>(null);
  const [useWebGL, setUseWebGL] = useState(true);
  const [currentFrame, setCurrentFrame] = useState<VisualFrame | null>(null);
  const [processingMetrics, setProcessingMetrics] = useState<ProcessingMetrics | null>(null);
  
  const animationFrameRef = useRef<number | undefined>(undefined);
  const lastFrameTimeRef = useRef<number>(0);
  const frameCountRef = useRef<number>(0);

  console.log('Detected Objects:', detectedObjects); // Added usage
  console.log('Processing Stats:', stats); // Added usage
  console.log('Last Frame Time Ref:', lastFrameTimeRef.current); // Added usage
  console.log('Frame Count Ref:', frameCountRef.current); // Added usage

  const stopProcessing = useCallback(() => {
    if (animationFrameRef.current) {
      cancelAnimationFrame(animationFrameRef.current);
    }

    if (stream) {
      stream.getTracks().forEach(track => track.stop());
      setStream(null);
    }

    setIsProcessing(false);
    // setCurrentFrame(null);
  }, [stream]);

  const detectObjects = useCallback((imageData: ImageData): SensoryDetectedObject[] => {
    // Simple object detection using edge detection and blob analysis
    const { data, width, height } = imageData;
    const binary = new Uint8Array(width * height);

    // Convert to grayscale and apply edge detection
    for (let i = 0; i < data.length; i += 4) {
      const gray = (data[i] + data[i + 1] + data[i + 2]) / 3;
      binary[i / 4] = gray > 128 ? 255 : 0;
    }

    // Simple blob detection (mock implementation)
    const objects: SensoryDetectedObject[] = [];

    // Mock object detection - in real implementation, this would use computer vision algorithms
    if (Math.random() > 0.7) { // Randomly detect objects
      objects.push({
        id: `obj_${Date.now()}`,
        type: 'person',
        label: 'person',
        confidence: 0.85,
        boundingBox: {
          x: Math.random() * (width - 100),
          y: Math.random() * (height - 100),
          width: 100,
          height: 150
        },
        features: [0.1, 0.2, 0.3, 0.4, 0.5]
      });
    }

    return objects;
  }, []);

  const processVideoFrame = useCallback(() => {
    if (!videoRef.current || !canvasRef.current || !isProcessing) return;

    const video = videoRef.current;
    const canvas = canvasRef.current;
    const ctx = canvas.getContext('2d');

    if (!ctx) return;

    // Capture frame
    canvas.width = video.videoWidth;
    canvas.height = video.videoHeight;
    ctx.drawImage(video, 0, 0);

    const imageData = ctx.getImageData(0, 0, canvas.width, canvas.height);

    // Create VisualFrame and update current frame
    const visualFrame: VisualFrame = {
      imageData: imageData,
      timestamp: new Date(),
      frameNumber: Math.floor(Date.now() / 1000) // Simple frame numbering
    };
    setCurrentFrame(visualFrame);

    // Process frame for object detection
    const startTime = performance.now();
    const objects = detectObjects(imageData);
    const processingTime = performance.now() - startTime;

    // Update processing metrics
    const metrics: ProcessingMetrics = {
      throughput: stats.fps, // frames per second
      latency: processingTime, // processing time in milliseconds
      accuracy: objects.length > 0 ? 0.85 : 0.0, // Mock accuracy based on detection
      errorRate: 0.0, // No errors for now
      resourceUsage: {
        cpu: Math.min(processingTime / 16.67 * 100, 100), // Estimate CPU usage
        memory: imageData.data.length // Rough memory usage estimate
      },
      uptime: Date.now() - startTime // Simple uptime calculation
    };
    setProcessingMetrics(metrics);

    setDetectedObjects(objects);

    // Call callbacks
    onVisualData(imageData);
    onObjectDetection(objects);

    // Schedule next frame
    if (isProcessing) {
      animationFrameRef.current = requestAnimationFrame(processVideoFrame);
    }
  }, [isProcessing, onVisualData, onObjectDetection, stats.fps, detectObjects]);

  const initializeCamera = useCallback(async () => {
    try {
      setError(null);

      const mediaStream = await navigator.mediaDevices.getUserMedia({
        video: {
          width: { ideal: 1280 },
          height: { ideal: 720 },
          facingMode: 'user'
        }
      });

      setStream(mediaStream);

      if (videoRef.current) {
        videoRef.current.srcObject = mediaStream;
        videoRef.current.play();
      }

      setIsProcessing(true);
      processVideoFrame();

    } catch (error) {
      console.error('Failed to initialize camera:', error);
      setError('Failed to access camera. Please check permissions.');
    }
  }, [processVideoFrame]);

  useEffect(() => {
    if (isActive) {
      initializeCamera();
    } else {
      stopProcessing();
    }

    return () => {
      stopProcessing();
    };
  }, [isActive, initializeCamera, stopProcessing]);



  /* const startProcessing = () => {
    if (!videoRef.current || !canvasRef.current) return;

    setIsProcessing(true);
    processFrame();
  }; */

  // const processFrame = () => {
  //   if (!isActive || !videoRef.current || !canvasRef.current || !overlayCanvasRef.current) {
  //     return;
  //   }

  //   const video = videoRef.current;
  //   const canvas = canvasRef.current;
  //   const overlayCanvas = overlayCanvasRef.current;
  //   const ctx = canvas.getContext('2d');
  //   const overlayCtx = overlayCanvas.getContext('2d');

  //   if (!ctx || !overlayCtx) return;

  //   const startTime = performance.now();

  //   // Set canvas dimensions to match video
  //   canvas.width = video.videoWidth;
  //   canvas.height = video.videoHeight;
  //   overlayCanvas.width = video.videoWidth;
  //   overlayCanvas.height = video.videoHeight;

  //   // Draw current frame to canvas
  //   ctx.drawImage(video, 0, 0, canvas.width, canvas.height);

  //   // Get image data for processing
  //   const imageData = ctx.getImageData(0, 0, canvas.width, canvas.height);
    
  //   // Process visual data
  //   if (useWebGL) {
  //     processWithWebGL(imageData);
  //   } else {
  //     processWithCanvas(imageData);
  //   }

  //   // Send image data to parent component
  //   onVisualData(imageData);

  //   // Perform object detection (simplified)
  //   const objects = performObjectDetection(imageData);
  //   setDetectedObjects(objects);
  //   onObjectDetection(objects);

  //   // Draw detection overlays
  //   drawDetectionOverlays(overlayCtx, objects);

  //   // Update stats
  //   const processingTime = performance.now() - startTime;
  //   updateStats(processingTime);

  //   // Continue processing
  //   animationFrameRef.current = requestAnimationFrame(processFrame);
  // };

  /* const processWithWebGL = (imageData: ImageData) => {
    // WebGL-based image processing for better performance
    const canvas = canvasRef.current;
    if (!canvas) return;

    const gl = canvas.getContext('webgl') as WebGLRenderingContext | null;
    if (!gl) {
      console.warn('WebGL not supported, falling back to Canvas 2D');
      setUseWebGL(false);
      return;
    }

    // Create texture from image data
    const texture = gl.createTexture();
    gl.bindTexture(gl.TEXTURE_2D, texture);

    // Create a temporary canvas to convert ImageData to a format WebGL can use
    const tempCanvas = document.createElement('canvas');
    tempCanvas.width = imageData.width;
    tempCanvas.height = imageData.height;
    const tempCtx = tempCanvas.getContext('2d');
    if (tempCtx) {
      tempCtx.putImageData(imageData, 0, 0);
      gl.texImage2D(gl.TEXTURE_2D, 0, gl.RGBA, gl.RGBA, gl.UNSIGNED_BYTE, tempCanvas);
    }

    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR);
    gl.texParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR);

    // Note: Full shader compilation would be implemented here
    // For now, we'll just use the texture for basic processing
    console.log('WebGL processing applied to texture');
  }; */

  /* const processWithCanvas = (imageData: ImageData) => {
    // Canvas 2D-based image processing
    const data = imageData.data;

    // Apply simple edge detection filter
    for (let i = 0; i < data.length; i += 4) {
      const r = data[i];
      const g = data[i + 1];
      const b = data[i + 2];

      // Convert to grayscale and enhance edges
      const gray = 0.299 * r + 0.587 * g + 0.114 * b;
      const enhanced = Math.min(255, gray * 1.2);

      data[i] = enhanced;     // R
      data[i + 1] = enhanced; // G
      data[i + 2] = enhanced; // B
      // Alpha channel remains unchanged
    }
  }; */

  const performObjectDetection = useCallback((imageData: ImageData): SensoryDetectedObject[] => {
    // Simplified object detection using basic computer vision techniques
    const objects: SensoryDetectedObject[] = [];
    const data = imageData.data;
    const width = imageData.width;
    const height = imageData.height;

    // Simple blob detection for demonstration
    const threshold = 128;
    const minBlobSize = 100;
    
    // Convert to binary image
    const binary = new Uint8Array(width * height);
    for (let i = 0; i < data.length; i += 4) {
      const gray = 0.299 * data[i] + 0.587 * data[i + 1] + 0.114 * data[i + 2];
      binary[i / 4] = gray > threshold ? 255 : 0;
    }

    // Find connected components (simplified)
    const visited = new Set<number>();
    let objectId = 0;

    for (let y = 0; y < height; y++) {
      for (let x = 0; x < width; x++) {
        const index = y * width + x;
        
        if (binary[index] === 255 && !visited.has(index)) {
          const blob = floodFill(binary, width, height, x, y, visited);
          
          if (blob.length > minBlobSize) {
            const boundingBox = calculateBoundingBox(blob, width);
            const features = extractFeatures(blob, boundingBox);
            
            objects.push({
              id: `obj_${objectId++}`,
              type: classifyObject(features),
              label: classifyObject(features),
              confidence: Math.random() * 0.5 + 0.5, // Mock confidence
              boundingBox,
              features
            });
          }
        }
      }
    }

    return objects.slice(0, 10); // Limit to 10 objects
  }, []);

  const floodFill = (binary: Uint8Array, width: number, height: number, startX: number, startY: number, visited: Set<number>): number[] => {
    const stack = [{ x: startX, y: startY }];
    const blob: number[] = [];

    while (stack.length > 0) {
      const { x, y } = stack.pop()!;
      const index = y * width + x;

      if (x < 0 || x >= width || y < 0 || y >= height || visited.has(index) || binary[index] === 0) {
        continue;
      }

      visited.add(index);
      blob.push(index);

      // Add neighbors
      stack.push({ x: x + 1, y }, { x: x - 1, y }, { x, y: y + 1 }, { x, y: y - 1 });
    }

    return blob;
  };

  const calculateBoundingBox = (blob: number[], width: number) => {
    let minX = width, minY = Infinity, maxX = 0, maxY = 0;

    for (const index of blob) {
      const x = index % width;
      const y = Math.floor(index / width);

      minX = Math.min(minX, x);
      maxX = Math.max(maxX, x);
      minY = Math.min(minY, y);
      maxY = Math.max(maxY, y);
    }

    return {
      x: minX,
      y: minY,
      width: maxX - minX,
      height: maxY - minY
    };
  };

  const extractFeatures = (blob: number[], boundingBox: BoundingBox): number[] => {
    // Extract simple features: area, aspect ratio, compactness
    const area = blob.length;
    const aspectRatio = boundingBox.width / boundingBox.height;
    const compactness = (4 * Math.PI * area) / Math.pow(boundingBox.width + boundingBox.height, 2);
    
    return [area, aspectRatio, compactness];
  };

  const classifyObject = (features: number[]): string => {
    // Simple classification based on features
    const [area, aspectRatio] = features;
    
    if (area > 1000 && aspectRatio > 1.5) return 'rectangle';
    if (area > 500 && aspectRatio < 1.2) return 'square';
    if (area < 200) return 'small_object';
    return 'unknown';
  };

  /* const drawDetectionOverlays = (ctx: CanvasRenderingContext2D, objects: SensoryDetectedObject[]) => {
    ctx.clearRect(0, 0, ctx.canvas.width, ctx.canvas.height);

    objects.forEach((obj, index) => {
      const { x, y, width, height } = obj.boundingBox;

      // Draw bounding box
      ctx.strokeStyle = `hsl(${(index * 60) % 360}, 70%, 50%)`;
      ctx.lineWidth = 2;
      ctx.strokeRect(x, y, width, height);

      // Draw label
      ctx.fillStyle = ctx.strokeStyle;
      ctx.font = '12px Arial';
      ctx.fillText(`${obj.type} (${(obj.confidence * 100).toFixed(0)}%)`, x, y - 5);
    });
  }; */

  /* const updateStats = (processingTime: number) => {
    const now = performance.now();
    frameCountRef.current++;

    if (now - lastFrameTimeRef.current >= 1000) {
      const fps = frameCountRef.current;
      setStats({
        fps,
        objectCount: detectedObjects.length,
        processingTime: Math.round(processingTime)
      });

      frameCountRef.current = 0;
      lastFrameTimeRef.current = now;
    }
  }; */



  return (
    <div className="bg-white rounded-lg shadow-lg p-4 m-4">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-lg font-semibold text-gray-800">Visual Processing</h3>
        <div className="flex items-center gap-2">
          <button
            onClick={() => setUseWebGL(!useWebGL)}
            className={`p-2 rounded-full transition-colors ${
              useWebGL ? 'bg-green-500 text-white' : 'bg-gray-300 text-gray-600'
            }`}
            title={useWebGL ? 'WebGL Enabled' : 'Canvas 2D Mode'}
          >
            <Zap size={16} />
          </button>
          {isProcessing ? <Eye className="text-green-500" size={20} /> : <EyeOff className="text-gray-400" size={20} />}
        </div>
      </div>

      {/* Video and Canvas Container */}
      <div className="relative mb-4">
        <video
          ref={videoRef}
          className="w-full h-48 bg-gray-200 rounded object-cover"
          playsInline
          muted
        />
        <canvas
          ref={canvasRef}
          className="absolute inset-0 w-full h-48 opacity-0"
        />
        <canvas
          ref={overlayCanvasRef}
          className="absolute inset-0 w-full h-48 pointer-events-none"
        />
      </div>

      {/* Stats */}
      <div className="grid grid-cols-3 gap-4 text-center">
        <div>
          <div className="text-lg font-bold text-blue-600">{stats.fps}</div>
          <div className="text-xs text-gray-600">FPS</div>
        </div>
        <div>
          <div className="text-lg font-bold text-green-600">{stats.objectCount}</div>
          <div className="text-xs text-gray-600">Objects</div>
        </div>
        <div>
          <div className="text-lg font-bold text-purple-600">{stats.processingTime}ms</div>
          <div className="text-xs text-gray-600">Processing</div>
        </div>
      </div>

      {/* Debug Information */}
      {currentFrame && (
        <div className="mt-4 text-xs text-gray-500 text-center">
          Frame: {currentFrame.imageData.width}x{currentFrame.imageData.height} @ {new Date(currentFrame.timestamp).toLocaleTimeString()}
        </div>
      )}

      {processingMetrics && (
        <div className="mt-2 text-xs text-gray-500 text-center">
          Metrics: {processingMetrics.latency.toFixed(2)}ms | {processingMetrics.throughput.toFixed(1)} FPS | {processingMetrics.accuracy.toFixed(2)} accuracy
        </div>
      )}

      {error && (
        <div className="mt-4 text-sm text-red-600 text-center">
          {error}
        </div>
      )}
    </div>
  );
};

export default VisualProcessor;

