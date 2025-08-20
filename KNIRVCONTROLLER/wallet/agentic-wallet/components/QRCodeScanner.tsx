import React, { useState, useEffect } from 'react';
import {
  View,
  Text,
  StyleSheet,
  TouchableOpacity,
  Alert,
  Dimensions,
  ActivityIndicator,
} from 'react-native';
import { Camera, CameraView, useCameraPermissions } from 'expo-camera';
import { Ionicons } from '@expo/vector-icons';
import { LinearGradient } from 'expo-linear-gradient';
import { QRCodeService, QRCodeData } from '../services/QRCodeService';

interface QRCodeScannerProps {
  onScanSuccess: (data: QRCodeData) => void;
  onClose: () => void;
  title?: string;
  subtitle?: string;
}

const { width, height } = Dimensions.get('window');
const scanAreaSize = width * 0.7;

export const QRCodeScanner: React.FC<QRCodeScannerProps> = ({
  onScanSuccess,
  onClose,
  title = 'Scan QR Code',
  subtitle = 'Point your camera at the QR code to connect wallets',
}) => {
  const [permission, requestPermission] = useCameraPermissions();
  const [isScanning, setIsScanning] = useState(true);
  const [isProcessing, setIsProcessing] = useState(false);
  const qrService = QRCodeService.getInstance();

  useEffect(() => {
    if (!permission?.granted) {
      requestPermission();
    }
  }, [permission, requestPermission]);

  const handleBarCodeScanned = async ({ data }: { data: string }) => {
    if (!isScanning || isProcessing) return;

    setIsScanning(false);
    setIsProcessing(true);

    try {
      // Parse the QR code data
      const qrData = await qrService.parseQRCodeData(data);
      
      // Handle different types of QR codes
      switch (qrData.type) {
        case 'WALLET_CONNECT':
          const connected = await qrService.handleWalletConnection(qrData);
          if (connected) {
            onScanSuccess(qrData);
          } else {
            setIsScanning(true);
          }
          break;
          
        case 'TRANSACTION_REQUEST':
          onScanSuccess(qrData);
          break;
          
        case 'SYNC_REQUEST':
          onScanSuccess(qrData);
          break;
          
        default:
          Alert.alert('Invalid QR Code', 'This QR code is not supported');
          setIsScanning(true);
      }
    } catch (error) {
      Alert.alert(
        'Scan Error',
        error.message || 'Failed to process QR code',
        [
          {
            text: 'Try Again',
            onPress: () => setIsScanning(true),
          },
          {
            text: 'Cancel',
            onPress: onClose,
          },
        ]
      );
    } finally {
      setIsProcessing(false);
    }
  };

  const renderPermissionRequest = () => (
    <View style={styles.permissionContainer}>
      <Ionicons name="camera-outline" size={64} color="#FF6B35" />
      <Text style={styles.permissionTitle}>Camera Permission Required</Text>
      <Text style={styles.permissionText}>
        We need access to your camera to scan QR codes
      </Text>
      <TouchableOpacity style={styles.permissionButton} onPress={requestPermission}>
        <Text style={styles.permissionButtonText}>Grant Permission</Text>
      </TouchableOpacity>
    </View>
  );

  const renderScanOverlay = () => (
    <View style={styles.overlay}>
      <LinearGradient
        colors={['rgba(0,0,0,0.8)', 'transparent']}
        style={styles.topOverlay}
      >
        <View style={styles.header}>
          <TouchableOpacity style={styles.closeButton} onPress={onClose}>
            <Ionicons name="close" size={24} color="#fff" />
          </TouchableOpacity>
          <View style={styles.headerText}>
            <Text style={styles.title}>{title}</Text>
            <Text style={styles.subtitle}>{subtitle}</Text>
          </View>
          <View style={styles.placeholder} />
        </View>
      </LinearGradient>

      <View style={styles.scanArea}>
        <View style={styles.scanFrame}>
          {/* Corner indicators */}
          <View style={[styles.corner, styles.topLeft]} />
          <View style={[styles.corner, styles.topRight]} />
          <View style={[styles.corner, styles.bottomLeft]} />
          <View style={[styles.corner, styles.bottomRight]} />
          
          {/* Scanning animation */}
          {isScanning && !isProcessing && (
            <View style={styles.scanLine} />
          )}
          
          {/* Processing indicator */}
          {isProcessing && (
            <View style={styles.processingContainer}>
              <ActivityIndicator size="large" color="#FF6B35" />
              <Text style={styles.processingText}>Processing...</Text>
            </View>
          )}
        </View>
      </View>

      <LinearGradient
        colors={['transparent', 'rgba(0,0,0,0.8)']}
        style={styles.bottomOverlay}
      >
        <View style={styles.instructions}>
          <Ionicons name="qr-code-outline" size={32} color="#fff" />
          <Text style={styles.instructionText}>
            Align the QR code within the frame
          </Text>
          <Text style={styles.instructionSubtext}>
            Make sure the code is clearly visible and well-lit
          </Text>
        </View>
      </LinearGradient>
    </View>
  );

  if (!permission) {
    return (
      <View style={styles.container}>
        <ActivityIndicator size="large" color="#FF6B35" />
      </View>
    );
  }

  if (!permission.granted) {
    return (
      <View style={styles.container}>
        {renderPermissionRequest()}
      </View>
    );
  }

  return (
    <View style={styles.container}>
      <CameraView
        style={styles.camera}
        facing="back"
        onBarcodeScanned={isScanning ? handleBarCodeScanned : undefined}
        barcodeScannerSettings={{
          barcodeTypes: ['qr'],
        }}
      />
      {renderScanOverlay()}
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#000',
  },
  camera: {
    flex: 1,
  },
  permissionContainer: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: 20,
    backgroundColor: '#fff',
  },
  permissionTitle: {
    fontSize: 24,
    fontWeight: 'bold',
    color: '#1F2937',
    marginTop: 20,
    marginBottom: 10,
  },
  permissionText: {
    fontSize: 16,
    color: '#6B7280',
    textAlign: 'center',
    marginBottom: 30,
    lineHeight: 24,
  },
  permissionButton: {
    backgroundColor: '#FF6B35',
    paddingHorizontal: 32,
    paddingVertical: 12,
    borderRadius: 8,
  },
  permissionButtonText: {
    color: '#fff',
    fontSize: 16,
    fontWeight: '600',
  },
  overlay: {
    position: 'absolute',
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
  },
  topOverlay: {
    height: (height - scanAreaSize) / 2,
    justifyContent: 'flex-end',
    paddingBottom: 20,
  },
  bottomOverlay: {
    height: (height - scanAreaSize) / 2,
    justifyContent: 'flex-start',
    paddingTop: 20,
  },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: 20,
    paddingTop: 60,
  },
  closeButton: {
    width: 40,
    height: 40,
    borderRadius: 20,
    backgroundColor: 'rgba(255, 255, 255, 0.2)',
    justifyContent: 'center',
    alignItems: 'center',
  },
  headerText: {
    flex: 1,
    alignItems: 'center',
  },
  placeholder: {
    width: 40,
  },
  title: {
    fontSize: 20,
    fontWeight: 'bold',
    color: '#fff',
    textAlign: 'center',
  },
  subtitle: {
    fontSize: 14,
    color: 'rgba(255, 255, 255, 0.8)',
    textAlign: 'center',
    marginTop: 4,
  },
  scanArea: {
    height: scanAreaSize,
    justifyContent: 'center',
    alignItems: 'center',
  },
  scanFrame: {
    width: scanAreaSize,
    height: scanAreaSize,
    position: 'relative',
  },
  corner: {
    position: 'absolute',
    width: 30,
    height: 30,
    borderColor: '#FF6B35',
    borderWidth: 3,
  },
  topLeft: {
    top: 0,
    left: 0,
    borderRightWidth: 0,
    borderBottomWidth: 0,
  },
  topRight: {
    top: 0,
    right: 0,
    borderLeftWidth: 0,
    borderBottomWidth: 0,
  },
  bottomLeft: {
    bottom: 0,
    left: 0,
    borderRightWidth: 0,
    borderTopWidth: 0,
  },
  bottomRight: {
    bottom: 0,
    right: 0,
    borderLeftWidth: 0,
    borderTopWidth: 0,
  },
  scanLine: {
    position: 'absolute',
    top: '50%',
    left: 0,
    right: 0,
    height: 2,
    backgroundColor: '#FF6B35',
    opacity: 0.8,
  },
  processingContainer: {
    position: 'absolute',
    top: '50%',
    left: '50%',
    transform: [{ translateX: -50 }, { translateY: -50 }],
    alignItems: 'center',
    backgroundColor: 'rgba(0, 0, 0, 0.8)',
    padding: 20,
    borderRadius: 12,
  },
  processingText: {
    color: '#fff',
    fontSize: 16,
    marginTop: 12,
  },
  instructions: {
    alignItems: 'center',
    paddingHorizontal: 40,
  },
  instructionText: {
    fontSize: 18,
    fontWeight: '600',
    color: '#fff',
    textAlign: 'center',
    marginTop: 12,
  },
  instructionSubtext: {
    fontSize: 14,
    color: 'rgba(255, 255, 255, 0.7)',
    textAlign: 'center',
    marginTop: 8,
    lineHeight: 20,
  },
});
