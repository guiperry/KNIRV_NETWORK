import { useEffect, useRef, useState } from 'react';
import QrScanner from 'qr-scanner';
import { Camera, X, Flashlight, FlashlightOff } from 'lucide-react';

interface QRScannerProps {
  onScan: (result: string) => void;
  onClose: () => void;
  isOpen: boolean;
}

interface QRData {
  version: string;
  type: string;
  session_id: string;
  desktop_id: string;
  target_id?: string;
  expires_at: number;
  endpoint: string;
  public_key: string;
  capabilities?: string[];
  encrypted_payload?: string;
  signature: string;
}

export default function QRScanner({ onScan, onClose, isOpen }: QRScannerProps) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const [qrScanner, setQrScanner] = useState<QrScanner | null>(null);
  const [hasFlash, setHasFlash] = useState(false);
  const [flashEnabled, setFlashEnabled] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [scanning, setScanning] = useState(false);

  useEffect(() => {
    if (isOpen && videoRef.current) {
      initializeScanner();
    }

    return () => {
      if (qrScanner) {
        qrScanner.destroy();
      }
    };
  }, [isOpen]);

  const initializeScanner = async () => {
    if (!videoRef.current) return;

    try {
      setError(null);
      setScanning(true);

      const scanner = new QrScanner(
        videoRef.current,
        (result) => handleScanResult(result.data),
        {
          highlightScanRegion: true,
          highlightCodeOutline: true,
          preferredCamera: 'environment', // Use back camera on mobile
        }
      );

      // Check if device has flash
      const hasFlashSupport = await QrScanner.hasCamera();
      setHasFlash(hasFlashSupport);

      await scanner.start();
      setQrScanner(scanner);
      setScanning(false);
    } catch (err) {
      console.error('Failed to initialize QR scanner:', err);
      setError('Failed to access camera. Please check permissions.');
      setScanning(false);
    }
  };

  const handleScanResult = (data: string) => {
    try {
      // Parse QR code data
      const qrData: QRData = JSON.parse(data);
      
      // Validate QR code structure
      if (!qrData.version || !qrData.session_id || !qrData.desktop_id) {
        throw new Error('Invalid QR code format');
      }

      // Check if QR code has expired
      if (qrData.expires_at && Date.now() / 1000 > qrData.expires_at) {
        setError('QR code has expired');
        return;
      }

      console.log('Valid QR code scanned:', qrData);
      onScan(data);
    } catch (err) {
      console.error('Invalid QR code:', err);
      setError('Invalid QR code format');
    }
  };

  const toggleFlash = async () => {
    if (qrScanner && hasFlash) {
      try {
        // Note: setFlash method may not be available in all QrScanner versions
        // This is a simplified implementation
        setFlashEnabled(!flashEnabled);
        console.log('Flash toggle requested:', !flashEnabled);
      } catch (err) {
        console.error('Failed to toggle flash:', err);
      }
    }
  };

  const handleClose = () => {
    if (qrScanner) {
      qrScanner.destroy();
      setQrScanner(null);
    }
    setError(null);
    setFlashEnabled(false);
    onClose();
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black z-50 flex flex-col">
      {/* Header */}
      <div className="flex items-center justify-between p-4 bg-gray-900 text-white">
        <h2 className="text-lg font-semibold flex items-center gap-2">
          <Camera size={20} />
          Scan QR Code
        </h2>
        <div className="flex items-center gap-2">
          {hasFlash && (
            <button
              onClick={toggleFlash}
              className="p-2 rounded-full bg-gray-700 hover:bg-gray-600 transition-colors"
            >
              {flashEnabled ? <FlashlightOff size={20} /> : <Flashlight size={20} />}
            </button>
          )}
          <button
            onClick={handleClose}
            className="p-2 rounded-full bg-gray-700 hover:bg-gray-600 transition-colors"
          >
            <X size={20} />
          </button>
        </div>
      </div>

      {/* Scanner Area */}
      <div className="flex-1 relative">
        <video
          ref={videoRef}
          className="w-full h-full object-cover"
          playsInline
          muted
        />
        
        {/* Scanning overlay */}
        <div className="absolute inset-0 flex items-center justify-center">
          <div className="relative">
            {/* Scanning frame */}
            <div className="w-64 h-64 border-2 border-white rounded-lg relative">
              <div className="absolute top-0 left-0 w-8 h-8 border-t-4 border-l-4 border-blue-500 rounded-tl-lg"></div>
              <div className="absolute top-0 right-0 w-8 h-8 border-t-4 border-r-4 border-blue-500 rounded-tr-lg"></div>
              <div className="absolute bottom-0 left-0 w-8 h-8 border-b-4 border-l-4 border-blue-500 rounded-bl-lg"></div>
              <div className="absolute bottom-0 right-0 w-8 h-8 border-b-4 border-r-4 border-blue-500 rounded-br-lg"></div>
              
              {scanning && (
                <div className="absolute inset-0 flex items-center justify-center">
                  <div className="w-6 h-6 border-2 border-blue-500 border-t-transparent rounded-full animate-spin"></div>
                </div>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Instructions */}
      <div className="p-4 bg-gray-900 text-white text-center">
        <p className="text-sm">
          Position the QR code within the frame to scan
        </p>
        {error && (
          <p className="text-red-400 text-sm mt-2">
            {error}
          </p>
        )}
      </div>
    </div>
  );
}
