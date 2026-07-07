import React from 'react';
import PropTypes from 'prop-types';

// Simple modal overlay with an iframe inside. No external CSS deps.
export default function IframeModal({ title, src, onClose, height = '80vh' }) {
  return (
    <div style={styles.backdrop} role="dialog" aria-modal="true">
      <div style={styles.modal}>
        <div style={styles.header}>
          <div style={styles.title}>{title}</div>
          <div style={styles.actions}>
            <a href={src} target="_blank" rel="noreferrer" style={styles.link}>
              Open in new tab ↗
            </a>
            <button onClick={onClose} style={styles.closeBtn} aria-label="Close">
              ✕
            </button>
          </div>
        </div>
        <iframe
          title={title}
          src={src}
          style={{ ...styles.iframe, height }}
        />
      </div>
    </div>
  );
}

IframeModal.propTypes = {
  title: PropTypes.string.isRequired,
  src: PropTypes.string.isRequired,
  onClose: PropTypes.func.isRequired,
  height: PropTypes.string,
};

const styles = {
  backdrop: {
    position: 'fixed',
    inset: 0,
    background: 'rgba(0,0,0,0.6)',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    zIndex: 1000,
  },
  modal: {
    width: '90vw',
    maxWidth: 1400,
    background: '#0b1220',
    border: '1px solid rgba(255,255,255,0.12)',
    borderRadius: 12,
    boxShadow: '0 10px 30px rgba(0,0,0,0.5)',
    overflow: 'hidden',
  },
  header: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    padding: '12px 16px',
    background: 'rgba(255,255,255,0.06)',
    borderBottom: '1px solid rgba(255,255,255,0.1)',
    color: '#e6efff',
  },
  title: { fontWeight: 600 },
  actions: { display: 'flex', gap: 12, alignItems: 'center' },
  link: {
    color: '#8ab4ff',
    textDecoration: 'none',
    fontSize: 14,
  },
  closeBtn: {
    background: 'transparent',
    color: '#e6efff',
    border: '1px solid rgba(255,255,255,0.2)',
    borderRadius: 6,
    cursor: 'pointer',
    padding: '6px 10px',
  },
  iframe: {
    width: '100%',
    border: 'none',
    background: '#0b1220',
  },
};