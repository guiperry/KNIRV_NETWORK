import { useEffect, useState } from 'react';
import { useNavigation } from '../hooks/useNavigation';
import { useBackend } from '../contexts/BackendContext';
import api from '../utils/api';
import Image from 'next/image';
import PageLayout from '../components/PageLayout';
import PageHeader from '../components/PageHeader';
import GlassyCard from '../components/GlassyCard';
import styles from './nft-vault.module.css';

export default function NFTVault() {
  const { activePage } = useNavigation('nft-vault');
  const [searchQuery, setSearchQuery] = useState('');

  const handleSearch = (query) => {
    setSearchQuery(query);
  };
  const { isRunning } = useBackend();
  const [nfts, setNfts] = useState([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');
  const [selectedNft, setSelectedNft] = useState(null);
  const [uploadForm, setUploadForm] = useState({
    name: '',
    description: '',
    file: null,
    previewUrl: ''
  });
  const [capabilities, setCapabilities] = useState([]);
  const [selectedCapability, setSelectedCapability] = useState('');

  useEffect(() => {
    if (isRunning) {
      fetchNFTs();
      fetchCapabilities();
    }
  }, [isRunning]);

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

  const handleFileChange = (e) => {
    const file = e.target.files[0];
    if (!file) return;

    setUploadForm({
      ...uploadForm,
      file,
      previewUrl: URL.createObjectURL(file)
    });
  };

  const handleInputChange = (e) => {
    const { name, value } = e.target;
    setUploadForm({
      ...uploadForm,
      [name]: value
    });
  };

  const handleUpload = async (e) => {
    e.preventDefault();
    if (!uploadForm.file || !uploadForm.name) return;

    const formData = new FormData();
    formData.append('file', uploadForm.file);
    formData.append('name', uploadForm.name);
    formData.append('description', uploadForm.description);

    try {
      await api.post('/nft/upload', formData, {
        headers: {
          'Content-Type': 'multipart/form-data'
        }
      });
      
      setUploadForm({
        name: '',
        description: '',
        file: null,
        previewUrl: ''
      });
      
      fetchNFTs();
    } catch (error) {
      setError('Failed to upload NFT: ' + (error.response?.data?.message || error.message));
    }
  };

  const handleAttachCapability = async () => {
    if (!selectedNft || !selectedCapability) return;

    try {
      await api.post('/nft/attach-capability', {
        nft_id: selectedNft.id,
        capability_id: selectedCapability
      });
      
      alert('Capability attached successfully!');
      fetchNFTs();
    } catch (error) {
      alert('Failed to attach capability: ' + (error.response?.data?.message || error.message));
    }
  };

  if (!isRunning) {
    return (
      <PageLayout activePage={activePage} pageTitle="NFT Vault" onSearch={handleSearch}>
        <div className={styles.notRunning}>Backend is not running. Please start the KNIRVCHAIN node.</div>
      </PageLayout>
    );
  }

  if (isLoading) {
    return (
      <PageLayout activePage={activePage} pageTitle="NFT Vault" onSearch={handleSearch}>
        <div className={styles.loading}>Loading NFT vault...</div>
      </PageLayout>
    );
  }

  return (
    <PageLayout activePage={activePage} pageTitle="NFT Vault" onSearch={handleSearch}>
      <PageHeader 
        title="NFT Vault" 
        subtitle="Manage your digital collectibles and assets"
      />

      {error && <div className={styles.error}>{error}</div>}

      <GlassyCard title="Upload New NFT" darker className={styles.uploadSection}>
        <form onSubmit={handleUpload} className={styles.uploadForm}>
          <div className={styles.formGroup}>
            <label htmlFor="name">NFT Name</label>
            <input
              id="name"
              name="name"
              type="text"
              value={uploadForm.name}
              onChange={handleInputChange}
              required
            />
          </div>
          
          <div className={styles.formGroup}>
            <label htmlFor="description">Description</label>
            <textarea
              id="description"
              name="description"
              value={uploadForm.description}
              onChange={handleInputChange}
              rows={3}
            />
          </div>
          
          <div className={styles.fileUpload}>
            <label htmlFor="file">Upload Image</label>
            <input
              id="file"
              type="file"
              accept="image/*"
              onChange={handleFileChange}
              required
            />
            
            {uploadForm.previewUrl && (
              <div className={styles.preview}>
                <Image
                  src={uploadForm.previewUrl}
                  alt="Preview"
                  width={300}
                  height={300}
                />
              </div>
            )}
          </div>
          
          <button type="submit" className={`${styles.button} ${styles.primary}`}>
            Upload NFT
          </button>
        </form>
      </GlassyCard>
      
      <GlassyCard title="Your NFTs" darker className={styles.nftGallery}>
        {nfts.length === 0 ? (
          <div className={styles.emptyState}>No NFTs found in your vault</div>
        ) : (
          <div className={styles.nftGrid}>
            {nfts.map((nft) => (
              <div 
                key={nft.id} 
                className={`${styles.nftCard} ${selectedNft?.id === nft.id ? styles.selected : ''}`}
                onClick={() => setSelectedNft(nft)}
              >
                <div className={styles.nftImage}>
                  <Image
                    src={nft.image_url}
                    alt={nft.name}
                    fill
                    style={{ objectFit: 'cover' }}
                  />
                </div>
                <div className={styles.nftInfo}>
                  <h3>{nft.name}</h3>
                  <p>{nft.description}</p>
                </div>
              </div>
            ))}
          </div>
        )}
      </GlassyCard>
      
      {selectedNft && (
        <GlassyCard title="NFT Details" darker className={styles.nftDetails}>
          <div className={styles.detailsContent}>
            <div className={styles.nftImage}>
              <Image
                src={selectedNft.image_url}
                alt={selectedNft.name}
                fill
                style={{ objectFit: 'cover' }}
              />
            </div>
            
            <div className={styles.nftInfo}>
              <h3>{selectedNft.name}</h3>
              <p>{selectedNft.description}</p>
              <div><strong>ID:</strong> {selectedNft.id}</div>
              <div><strong>Created:</strong> {new Date(selectedNft.created_at).toLocaleString()}</div>
              
              <div className={styles.capabilitiesSection}>
                <h4>Attached Capabilities</h4>
                {selectedNft.capabilities?.length > 0 ? (
                  <ul>
                    {selectedNft.capabilities.map((cap) => (
                      <li key={cap.id}>{cap.name} ({cap.type})</li>
                    ))}
                  </ul>
                ) : (
                  <p>No capabilities attached</p>
                )}
                
                <div className={styles.attachCapability}>
                  <select
                    value={selectedCapability}
                    onChange={(e) => setSelectedCapability(e.target.value)}
                  >
                    <option value="">Select a capability</option>
                    {capabilities.map((cap) => (
                      <option key={cap.id} value={cap.id}>
                        {cap.name} ({cap.type})
                      </option>
                    ))}
                  </select>
                  <button className={`${styles.button} ${styles.secondary}`}
                    onClick={handleAttachCapability}
                    disabled={!selectedCapability}
                  >
                    Attach Capability
                  </button>
                </div>
              </div>
            </div>
          </div>
        </GlassyCard>
      )}
    </PageLayout>
  );
}
