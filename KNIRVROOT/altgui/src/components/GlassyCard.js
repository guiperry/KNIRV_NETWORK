import React from 'react';
import styles from './GlassyCard.module.css';

const GlassyCard = ({ 
  children, 
  title, 
  darker = false, 
  className = '' 
}) => {
  return (
    <div className={`${styles.glassyCard} ${darker ? styles.darkerContainer : ''} ${className}`}>
      {title && <h3 className={styles.cardTitle}>{title}</h3>}
      {children}
    </div>
  );
};

export default GlassyCard;