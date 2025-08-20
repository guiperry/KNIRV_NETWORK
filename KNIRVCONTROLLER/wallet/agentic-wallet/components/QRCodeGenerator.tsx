import React, { useState, useEffect } from 'react';
import {
  View,
  Text,
  StyleSheet,
  TouchableOpacity,
  Share,
  Alert,
  Clipboard,
  Dimensions,
} from 'react-native';
import QRCode from 'react-native-qrcode-svg';
import { Ionicons } from '@expo/vector-icons';
import { LinearGradient } from 'expo-linear-gradient';
import { QRCodeService } from '../services/QRCodeService';

interface QRCodeGeneratorProps {
  type: 'wallet_connect' | 'transaction_request' | 'sync_request';
  data: {
    walletAddress?: string;
    publicKey?: string;
    chainId?: string;
    sessionId?: string;
    transactionData?: any;
    syncData?: any;
  };
  onClose: () => void;
  title?: string;
  subtitle?: string;
}

const { width } = Dimensions.get('window');
const qrSize = width * 0.7;

export const QRCodeGenerator: React.FC<QRCodeGeneratorProps> = ({
  type,
  data,
  onClose,
  title,
  subtitle,
}) => {
  const [qrData, setQrData] = useState<string>('');
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string>('');
  const qrService = QRCodeService.getInstance();

  useEffect(() => {
    generateQRCode();
  }, [type, data]);

  const generateQRCode = async () => {
    setIsLoading(true);
    setError('');

    try {
      let qrString = '';

      switch (type) {
        case 'wallet_connect':
          if (!data.walletAddress || !data.publicKey) {
            throw new Error('Wallet address and public key are required');
          }
          qrString = await qrService.generateWalletConnectionQR(
            data.walletAddress,
            data.publicKey,
            data.chainId || 'knirv-1'
          );
          break;

        case 'transaction_request':
          if (!data.sessionId || !data.transactionData) {
            throw new Error('Session ID and transaction data are required');
          }
          qrString = await qrService.generateTransactionRequestQR(
            data.sessionId,
            data.transactionData.from,
            data.transactionData.to,
            data.transactionData.amount,
            {
              token: data.transactionData.token,
              memo: data.transactionData.memo,
              gasLimit: data.transactionData.gasLimit,
            }
          );
          break;

        case 'sync_request':
          if (!data.sessionId || !data.syncData) {
            throw new Error('Session ID and sync data are required');
          }
          qrString = await qrService.generateSyncRequestQR(
            data.sessionId,
            data.syncData
          );
          break;

        default:
          throw new Error('Invalid QR code type');
      }

      setQrData(qrString);
    } catch (err) {
      setError(err.message || 'Failed to generate QR code');
    } finally {
      setIsLoading(false);
    }
  };

  const handleCopyToClipboard = async () => {
    try {
      await Clipboard.setString(qrData);
      Alert.alert('Copied', 'QR code data copied to clipboard');
    } catch (error) {
      Alert.alert('Error', 'Failed to copy to clipboard');
    }
  };

  const handleShare = async () => {
    try {
      await Share.share({
        message: qrData,
        title: 'KNIRV Wallet Connection',
      });
    } catch (error) {
      Alert.alert('Error', 'Failed to share QR code');
    }
  };

  const getDisplayInfo = () => {
    switch (type) {
      case 'wallet_connect':
        return {
          title: title || 'Connect Wallet',
          subtitle: subtitle || 'Scan this QR code with another KNIRV wallet to connect',
          icon: 'wallet-outline',
          color: '#4F46E5',
        };
      case 'transaction_request':
        return {
          title: title || 'Transaction Request',
          subtitle: subtitle || 'Scan to approve this transaction',
          icon: 'send-outline',
          color: '#059669',
        };
      case 'sync_request':
        return {
          title: title || 'Sync Wallets',
          subtitle: subtitle || 'Scan to synchronize wallet data',
          icon: 'sync-outline',
          color: '#DC2626',
        };
      default:
        return {
          title: 'QR Code',
          subtitle: 'Scan with KNIRV wallet',
          icon: 'qr-code-outline',
          color: '#6B7280',
        };
    }
  };

  const displayInfo = getDisplayInfo();

  if (isLoading) {
    return (
      <View style={styles.container}>
        <LinearGradient
          colors={['#FF6B35', '#FF8E53']}
          style={styles.header}
        >
          <TouchableOpacity style={styles.closeButton} onPress={onClose}>
            <Ionicons name="close" size={24} color="#fff" />
          </TouchableOpacity>
          <Text style={styles.headerTitle}>Generating QR Code...</Text>
          <View style={styles.placeholder} />
        </LinearGradient>
        <View style={styles.loadingContainer}>
          <Ionicons name="qr-code-outline" size={64} color="#FF6B35" />
          <Text style={styles.loadingText}>Generating QR code...</Text>
        </View>
      </View>
    );
  }

  if (error) {
    return (
      <View style={styles.container}>
        <LinearGradient
          colors={['#FF6B35', '#FF8E53']}
          style={styles.header}
        >
          <TouchableOpacity style={styles.closeButton} onPress={onClose}>
            <Ionicons name="close" size={24} color="#fff" />
          </TouchableOpacity>
          <Text style={styles.headerTitle}>Error</Text>
          <View style={styles.placeholder} />
        </LinearGradient>
        <View style={styles.errorContainer}>
          <Ionicons name="alert-circle-outline" size={64} color="#DC2626" />
          <Text style={styles.errorTitle}>Failed to Generate QR Code</Text>
          <Text style={styles.errorText}>{error}</Text>
          <TouchableOpacity style={styles.retryButton} onPress={generateQRCode}>
            <Text style={styles.retryButtonText}>Try Again</Text>
          </TouchableOpacity>
        </View>
      </View>
    );
  }

  return (
    <View style={styles.container}>
      <LinearGradient
        colors={['#FF6B35', '#FF8E53']}
        style={styles.header}
      >
        <TouchableOpacity style={styles.closeButton} onPress={onClose}>
          <Ionicons name="close" size={24} color="#fff" />
        </TouchableOpacity>
        <Text style={styles.headerTitle}>{displayInfo.title}</Text>
        <View style={styles.placeholder} />
      </LinearGradient>

      <View style={styles.content}>
        <View style={styles.infoSection}>
          <View style={[styles.iconContainer, { backgroundColor: displayInfo.color }]}>
            <Ionicons name={displayInfo.icon as any} size={32} color="#fff" />
          </View>
          <Text style={styles.title}>{displayInfo.title}</Text>
          <Text style={styles.subtitle}>{displayInfo.subtitle}</Text>
        </View>

        <View style={styles.qrContainer}>
          <View style={styles.qrWrapper}>
            <QRCode
              value={qrData}
              size={qrSize}
              backgroundColor="#fff"
              color="#000"
              logoSize={qrSize * 0.15}
              logoBackgroundColor="#fff"
              logoMargin={4}
              logoBorderRadius={8}
            />
          </View>
        </View>

        <View style={styles.actionsContainer}>
          <TouchableOpacity style={styles.actionButton} onPress={handleCopyToClipboard}>
            <Ionicons name="copy-outline" size={20} color="#FF6B35" />
            <Text style={styles.actionButtonText}>Copy</Text>
          </TouchableOpacity>

          <TouchableOpacity style={styles.actionButton} onPress={handleShare}>
            <Ionicons name="share-outline" size={20} color="#FF6B35" />
            <Text style={styles.actionButtonText}>Share</Text>
          </TouchableOpacity>

          <TouchableOpacity style={styles.actionButton} onPress={generateQRCode}>
            <Ionicons name="refresh-outline" size={20} color="#FF6B35" />
            <Text style={styles.actionButtonText}>Refresh</Text>
          </TouchableOpacity>
        </View>

        <View style={styles.warningContainer}>
          <Ionicons name="warning-outline" size={16} color="#F59E0B" />
          <Text style={styles.warningText}>
            This QR code expires in 5 minutes for security
          </Text>
        </View>
      </View>
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#fff',
  },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingTop: 60,
    paddingBottom: 20,
    paddingHorizontal: 20,
  },
  closeButton: {
    width: 40,
    height: 40,
    borderRadius: 20,
    backgroundColor: 'rgba(255, 255, 255, 0.2)',
    justifyContent: 'center',
    alignItems: 'center',
  },
  headerTitle: {
    fontSize: 18,
    fontWeight: 'bold',
    color: '#fff',
  },
  placeholder: {
    width: 40,
  },
  content: {
    flex: 1,
    padding: 20,
  },
  infoSection: {
    alignItems: 'center',
    marginBottom: 32,
  },
  iconContainer: {
    width: 64,
    height: 64,
    borderRadius: 32,
    justifyContent: 'center',
    alignItems: 'center',
    marginBottom: 16,
  },
  title: {
    fontSize: 24,
    fontWeight: 'bold',
    color: '#1F2937',
    marginBottom: 8,
  },
  subtitle: {
    fontSize: 16,
    color: '#6B7280',
    textAlign: 'center',
    lineHeight: 24,
  },
  qrContainer: {
    alignItems: 'center',
    marginBottom: 32,
  },
  qrWrapper: {
    padding: 20,
    backgroundColor: '#fff',
    borderRadius: 16,
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 4 },
    shadowOpacity: 0.1,
    shadowRadius: 12,
    elevation: 8,
  },
  actionsContainer: {
    flexDirection: 'row',
    justifyContent: 'space-around',
    marginBottom: 24,
  },
  actionButton: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingVertical: 12,
    paddingHorizontal: 20,
    borderRadius: 8,
    backgroundColor: 'rgba(255, 107, 53, 0.1)',
    borderWidth: 1,
    borderColor: '#FF6B35',
  },
  actionButtonText: {
    color: '#FF6B35',
    fontSize: 14,
    fontWeight: '600',
    marginLeft: 8,
  },
  warningContainer: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    padding: 12,
    backgroundColor: 'rgba(245, 158, 11, 0.1)',
    borderRadius: 8,
    borderWidth: 1,
    borderColor: '#F59E0B',
  },
  warningText: {
    color: '#F59E0B',
    fontSize: 12,
    marginLeft: 8,
    textAlign: 'center',
  },
  loadingContainer: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
  },
  loadingText: {
    fontSize: 16,
    color: '#6B7280',
    marginTop: 16,
  },
  errorContainer: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: 20,
  },
  errorTitle: {
    fontSize: 20,
    fontWeight: 'bold',
    color: '#DC2626',
    marginTop: 16,
    marginBottom: 8,
  },
  errorText: {
    fontSize: 14,
    color: '#6B7280',
    textAlign: 'center',
    marginBottom: 24,
    lineHeight: 20,
  },
  retryButton: {
    backgroundColor: '#FF6B35',
    paddingHorizontal: 24,
    paddingVertical: 12,
    borderRadius: 8,
  },
  retryButtonText: {
    color: '#fff',
    fontSize: 16,
    fontWeight: '600',
  },
});
