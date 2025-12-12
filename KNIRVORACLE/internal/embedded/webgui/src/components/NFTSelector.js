import React from 'react';
import Image from 'next/image';
import styles from './NFTSelector.module.css';

const NFTSelector = ({ nfts, selectedNft, onSelect, searchQuery }) => {
  const filteredNfts = nfts.filter(nft => 
    nft.name.toLowerCase().includes(searchQuery.toLowerCase())
  );

  return (
    <div className={styles.nftSelector}>
      {filteredNfts.length === 0 ? (
        <div className={styles.emptyState}>No NFTs found</div>
      ) : (
        <div className={styles.nftList}>
          {filteredNfts.map(nft => (
            <div 
              key={nft.id} 
              className={`${styles.nftItem} ${selectedNft?.id === nft.id ? styles.selected : ''}`}
              onClick={() => onSelect(nft)}
            >
              <div className={styles.nftThumbnail}>
                <Image
                  src={nft.image_url}
                  alt={nft.name}
                  width={60}
                  height={60}
                  style={{ objectFit: 'cover' }}
                />
              </div>
              <div className={styles.nftInfo}>
                <h4>{nft.name}</h4>
                <div className={styles.capabilityCount}>
                  {nft.capabilities?.length || 0} capabilities
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

export default NFTSelector;