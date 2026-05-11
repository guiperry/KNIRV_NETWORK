import React, { useState, useEffect } from 'react';
import styles from './Footer.module.css';

const Footer = () => {
  const [config, setConfig] = useState(null);
  const currentYear = new Date().getFullYear();

  useEffect(() => {
    // Load configuration - in a real implementation, this would come from a config service
    const mockConfig = {
      navigation: {
        main_site: 'https://knirv.com',
        products: 'https://knirv.network/products',
        documentation: 'https://knirv.network/docs',
        graphchain_explorer: 'https://knirv.network/graphchain-explorer',
        nanda_ans: 'https://knirv.network/nanda-ans',
        developer_portal: 'https://knirv.network/developer-portal',
        agentify: 'https://knirv.network/agentify'
      },
      footer: {
        resources: {
          support: 'https://knirv.network/support',
          blog: 'https://blog.knirv.com',
          forum: 'https://knirv.network/forum'
        },
        social: {
          discord: 'https://discord.gg/knirv',
          telegram: 'https://t.me/knirv',
          twitter: 'https://twitter.com/knirvnetwork',
          github: 'https://github.com/knirv-network'
        },
        legal: {
          terms: 'https://knirv.network/terms',
          privacy: 'https://knirv.network/privacy',
          contribution: 'https://knirv.network/contributing'
        }
      }
    };
    setConfig(mockConfig);
  }, []);

  const getLink = (path, fallback = '#') => {
    try {
      if (!config) return fallback;

      const keys = path.split('.');
      let value = config;

      for (const key of keys) {
        if (value && typeof value === 'object' && key in value) {
          value = value[key];
        } else {
          return fallback;
        }
      }

      return typeof value === 'string' ? value : fallback;
    } catch (error) {
      console.warn('Error getting link for path:', path, error);
      return fallback;
    }
  };

  if (!config) {
    return null; // Don't render footer until config is loaded
  }

  return (
    <footer className={styles.knirvFooter}>
      <div className={styles.footerContainer}>
        <div className={styles.footerContent}>
          <div className={styles.footerSection}>
            <h4>KNIRV Network</h4>
            <ul>
              <li><a href={getLink('navigation.main_site')} target="_blank" rel="noopener noreferrer">Main Site</a></li>
              <li><a href={getLink('navigation.products')}>Products</a></li>
              <li><a href={getLink('navigation.documentation')}>Documentation</a></li>
              <li><a href={getLink('footer.resources.support')}>Support</a></li>
            </ul>
          </div>

          <div className={styles.footerSection}>
            <h4>Developer Tools</h4>
            <ul>
              <li><a href={getLink('navigation.developer_portal')}>Developer Portal</a></li>
              <li><a href={getLink('navigation.agentify')}>Agentify</a></li>
            </ul>
          </div>

          <div className={styles.footerSection}>
            <h4>Community</h4>
            <ul>
              <li><a href={getLink('footer.social.discord')} target="_blank" rel="noopener noreferrer">Discord</a></li>
              <li><a href={getLink('footer.social.telegram')} target="_blank" rel="noopener noreferrer">Telegram</a></li>
              <li><a href={getLink('footer.social.twitter')} target="_blank" rel="noopener noreferrer">Twitter</a></li>
              <li><a href={getLink('footer.resources.forum')}>Forum</a></li>
            </ul>
          </div>

          <div className={styles.footerSection}>
            <h4>Resources</h4>
            <ul>
              <li><a href={getLink('footer.resources.blog')} target="_blank" rel="noopener noreferrer">Blog</a></li>
              <li><a href={getLink('footer.social.github')} target="_blank" rel="noopener noreferrer">GitHub</a></li>
              <li><a href={getLink('footer.legal.terms')}>Terms of Service</a></li>
              <li><a href={getLink('footer.legal.privacy')}>Privacy Policy</a></li>
            </ul>
          </div>
        </div>

        <div className={styles.footerBottom}>
          <div className={styles.footerBottomContent}>
            <div className={styles.footerLogo}>
              <span className={styles.footerBrand}>KNIRV</span>
              <span className={styles.footerTagline}>Decentralized Trusted Execution Network</span>
            </div>
            <div className={styles.footerLegal}>
              <p>&copy; {currentYear} KNIRV Network. All rights reserved.</p>
              <div className={styles.footerLegalLinks}>
                <a href={getLink('footer.legal.terms')}>Terms</a>
                <a href={getLink('footer.legal.privacy')}>Privacy</a>
                <a href={getLink('footer.legal.contribution')}>Contributing</a>
              </div>
            </div>
          </div>
        </div>
      </div>
    </footer>
  );
};

export default Footer;
