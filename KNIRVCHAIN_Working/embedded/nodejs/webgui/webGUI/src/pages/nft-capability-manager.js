import { useState, useEffect } from 'react';
import { useNavigation } from '../hooks/useNavigation';
import { useBackend } from '../contexts/BackendContext';
import api from '../utils/api';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import NFTSelector from '../components/NFTSelector';
import CapabilitySelector from '../components/CapabilitySelector';
import CapabilityAttachmentForm from '../components/CapabilityAttachmentForm';
import CapabilityAttachmentHistory from '../components/CapabilityAttachmentHistory';
import Image from 'next/image';
import styles from './nft-capability-manager.module.css';

export default function NFTCapabilityManager() {
  const { activePage } = useNavigation('nft-capability-manager');
  const { isRunning } = useBackend();
  const [nfts, setNfts] = useState([]);
  const [selectedNft, setSelectedNft] = useState(null);
  const [capabilities, setCapabilities] = useState([]);
  const [selectedCapability, setSelectedCapability] = useState(null);
  const [attachmentHistory, setAttachmentHistory] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [searchQuery, setSearchQuery] = useState('');

  useEffect(() => {
    if (isRunning) {
      fetchNFTs();
      fetchCapabilities();
    }
  }, [isRunning]);

  useEffect(() => {
    if (selectedNft && isRunning) {
      fetchAttachmentHistory(selectedNft.id);
    }
  }, [selectedNft, isRunning]);

  const fetchNFTs = async () => {
    setIsLoading(true);
    try {
      const response = await api.get('/nft/list');
      setNfts(response.data.nfts || []);
      setError('');
    } catch (error) {
      setError('Failed to fetch NFTs');
      console.error('Error fetching NFTs:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const fetchCapabilities = async () => {
    try {
      const response = await api.get('/mcp/capability/list');
      setCapabilities(response.data.capabilities || []);
    } catch (error) {
      console.error('Error fetching capabilities:', error);
    }
  };

  const fetchAttachmentHistory = async (nftId) => {
    try {
      // This endpoint would need to be implemented on the backend
      const response = await api.get(`/nft/capability-history/${nftId}`);
      setAttachmentHistory(response.data.history || []);
    } catch (error) {
      console.error('Error fetching attachment history:', error);
      // If the endpoint doesn't exist yet, we can use an empty array
      setAttachmentHistory([]);
    }
  };

  const handleNftSelect = (nft) => {
    setSelectedNft(nft);
    setSelectedCapability(null);
  };

  const handleCapabilitySelect = (capability) => {
    setSelectedCapability(capability);
  };

  const handleAttachCapability = async (params) => {
    if (!selectedNft || !selectedCapability) return;

    try {
      await api.post('/nft/attach-capability', {
        nftId: selectedNft.id,
        capabilityId: selectedCapability.id,
        params: params
      });
      
      setSuccess('Capability attached successfully!');
      fetchNFTs();
      fetchAttachmentHistory(selectedNft.id);
      setSelectedCapability(null);
      
      // Clear success message after 3 seconds
      setTimeout(() => {
        setSuccess('');
      }, 3000);
    } catch (error) {
      setError('Failed to attach capability: ' + (error.response?.data?.message || error.message));
      
      // Clear error message after 3 seconds
      setTimeout(() => {
        setError('');
      }, 3000);
    }
  };

  const handleSearch = (query) => {
    setSearchQuery(query);
  };

  if (!isRunning) {
    return (
      <PageLayout activePage={activePage} pageTitle="NFT Capability Manager" onSearch={handleSearch}>
        <div className={styles.notRunning}>Backend is not running. Please start the KNIRVCHAIN node.</div>
      </PageLayout>
    );
  }

  return (
    <PageLayout activePage={activePage} pageTitle="NFT Capability Manager" onSearch={handleSearch}>
      <PageHeader 
        title="NFT Capability Manager" 
        subtitle="Enhance your NFTs with powerful capabilities"
      />

      {error && <GlassyCard darker className={styles.error}>{error}</GlassyCard>}
      {success && <GlassyCard darker className={styles.success}>{success}</GlassyCard>}

      <div className={styles.container}>
        <div className={styles.leftPanel}>
          <GlassyCard darker className={styles.nftSelectorCard}>
            <h3 className={styles.cardTitle}>Your NFTs</h3>
            <NFTSelector 
              nfts={nfts} 
              selectedNft={selectedNft} 
              onSelect={handleNftSelect} 
              searchQuery={searchQuery}
            />
          </GlassyCard>
        </div>

        <div className={styles.rightPanel}>
          {selectedNft ? (
            <>
              <GlassyCard darker className={styles.nftDetailsCard}>
                <h3 className={styles.cardTitle}>Selected NFT</h3>
                <div className={styles.nftDetails}>
                  <div className={styles.nftImage}>
                    <Image
                      src={selectedNft.image_url}
                      alt={selectedNft.name}
                      width={300}
                      height={300}
                      layout="responsive"
                    />
                  </div>
                  <div className={styles.nftInfo}>
                    <h2>{selectedNft.name}</h2>
                    <p>{selectedNft.description}</p>
                    <div className={styles.nftMetadata}>
                      <div><strong>ID:</strong> {selectedNft.id}</div>
                      <div><strong>Created:</strong> {new Date(selectedNft.created_at).toLocaleString()}</div>
                    </div>
                  </div>
                </div>
              </GlassyCard>

              <GlassyCard darker className={styles.capabilitiesCard}>
                <h3 className={styles.cardTitle}>Available Capabilities</h3>
                <CapabilitySelector 
                  capabilities={capabilities} 
                  selectedCapability={selectedCapability} 
                  onSelect={handleCapabilitySelect}
                  attachedCapabilities={selectedNft.capabilities || []}
                />
              </GlassyCard>

              {selectedCapability && (
                <GlassyCard darker className={styles.attachmentFormCard}>
                  <h3 className={styles.cardTitle}>Attach Capability</h3>
                  <CapabilityAttachmentForm 
                    nft={selectedNft}
                    capability={selectedCapability}
                    onAttach={handleAttachCapability}
                  />
                </GlassyCard>
              )}

              <GlassyCard darker className={styles.historyCard}>
                <h3 className={styles.cardTitle}>Capability History</h3>
                <CapabilityAttachmentHistory history={attachmentHistory} />
              </GlassyCard>
            </>
          ) : (
            <GlassyCard darker className={styles.selectPrompt}>
              <div className={styles.promptContent}>
                <div className={styles.promptIcon}>🖼️</div>
                <h3>Select an NFT</h3>
                <p>Choose an NFT from your collection to view and manage its capabilities.</p>
              </div>
            </GlassyCard>
          )}
        </div>
      </div>
    </PageLayout>
  );
}