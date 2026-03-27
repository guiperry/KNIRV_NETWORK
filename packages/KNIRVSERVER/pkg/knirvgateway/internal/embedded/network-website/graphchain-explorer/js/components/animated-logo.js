/**
 * Animated KNIRV Logo Component
 * Vanilla JavaScript version of the animated oracle logo
 */

class AnimatedKNIRVLogo {
  constructor(container, options = {}) {
    this.container = container;
    this.options = {
      width: options.width || 60,
      height: options.height || 60,
      showText: options.showText !== false,
      textSize: options.textSize || 'small',
      ...options
    };
    this.glowIntensity = 0.5;
    this.animationId = null;
    this.element = null;
  }

  /**
   * Create and render the animated logo
   */
  render() {
    this.element = document.createElement('div');
    this.element.className = 'animated-knirv-logo';
    this.element.innerHTML = this.getTemplate();
    
    // Add styles
    this.addStyles();
    
    // Start animation
    this.startAnimation();
    
    return this.element;
  }

  /**
   * Get the HTML template for the logo
   */
  getTemplate() {
    const { width, height, showText, textSize } = this.options;
    
    return `
      <div class="knirv-logo-container" style="display: flex; align-items: center; gap: ${showText ? '12px' : '0'};">
        <div class="knirv-logo-svg-container" style="position: relative;">
          <svg width="${width}" height="${height}" viewBox="0 0 120 120" class="knirv-logo-svg">
            <!-- Central soma -->
            <circle cx="60" cy="60" r="8" fill="url(#somaGradient)" stroke="#00f5ff" stroke-width="1.5"/>
            
            <!-- Glow ring -->
            <circle cx="60" cy="60" r="12" fill="none" stroke="url(#glowRing)" stroke-width="1" class="glow-ring"/>
            
            <!-- Dendrites -->
            <path d="M55 52 C50 45, 45 40, 40 35 C38 32, 35 30, 32 28 C30 26, 27 25, 25 22" 
                  stroke="url(#dendriteGradient)" stroke-width="2" fill="none" opacity="0.9"/>
            <path d="M40 35 C38 32, 35 31, 33 28" 
                  stroke="url(#dendriteGradient)" stroke-width="1.5" fill="none" opacity="0.8"/>
            <path d="M35 30 C32 28, 30 25, 28 23" 
                  stroke="url(#dendriteGradient)" stroke-width="1" fill="none" opacity="0.7"/>
            
            <path d="M65 52 C70 45, 75 40, 80 35 C82 32, 85 30, 88 28 C90 26, 93 25, 95 22" 
                  stroke="url(#dendriteGradient)" stroke-width="2" fill="none" opacity="0.9"/>
            <path d="M80 35 C82 32, 85 31, 87 28" 
                  stroke="url(#dendriteGradient)" stroke-width="1.5" fill="none" opacity="0.8"/>
            <path d="M85 30 C88 28, 90 25, 92 23" 
                  stroke="url(#dendriteGradient)" stroke-width="1" fill="none" opacity="0.7"/>
            
            <path d="M55 68 C50 75, 45 80, 40 85 C38 88, 35 90, 32 92 C30 94, 27 95, 25 98" 
                  stroke="url(#dendriteGradient)" stroke-width="2" fill="none" opacity="0.9"/>
            <path d="M40 85 C38 88, 35 89, 33 92" 
                  stroke="url(#dendriteGradient)" stroke-width="1.5" fill="none" opacity="0.8"/>
            <path d="M35 90 C32 92, 30 95, 28 97" 
                  stroke="url(#dendriteGradient)" stroke-width="1" fill="none" opacity="0.7"/>
            
            <path d="M65 68 C70 75, 75 80, 80 85 C82 88, 85 90, 88 92 C90 94, 93 95, 95 98" 
                  stroke="url(#dendriteGradient)" stroke-width="2" fill="none" opacity="0.9"/>
            <path d="M80 85 C82 88, 85 89, 87 92" 
                  stroke="url(#dendriteGradient)" stroke-width="1.5" fill="none" opacity="0.8"/>
            <path d="M85 90 C88 92, 90 95, 92 97" 
                  stroke="url(#dendriteGradient)" stroke-width="1" fill="none" opacity="0.7"/>
            
            <path d="M52 60 C45 58, 38 56, 30 55 C27 54, 24 53, 20 52" 
                  stroke="url(#dendriteGradient)" stroke-width="2" fill="none" opacity="0.9"/>
            <path d="M30 55 C27 53, 24 52, 22 50" 
                  stroke="url(#dendriteGradient)" stroke-width="1.5" fill="none" opacity="0.8"/>
            
            <path d="M68 60 C75 58, 82 56, 90 55 C93 54, 96 53, 100 52" 
                  stroke="url(#dendriteGradient)" stroke-width="2" fill="none" opacity="0.9"/>
            <path d="M90 55 C93 53, 96 52, 98 50" 
                  stroke="url(#dendriteGradient)" stroke-width="1.5" fill="none" opacity="0.8"/>
            
            <!-- Terminal nodes -->
            <g class="terminal-nodes">
              <circle cx="25" cy="22" r="2" fill="#00f5ff"/>
              <circle cx="95" cy="22" r="2" fill="#00f5ff"/>
              <circle cx="25" cy="98" r="2" fill="#00f5ff"/>
              <circle cx="95" cy="98" r="2" fill="#00f5ff"/>
              <circle cx="20" cy="52" r="2" fill="#00f5ff"/>
              <circle cx="100" cy="52" r="2" fill="#00f5ff"/>
              <circle cx="33" cy="28" r="1.5" fill="#8b5cf6"/>
              <circle cx="87" cy="28" r="1.5" fill="#8b5cf6"/>
              <circle cx="33" cy="92" r="1.5" fill="#8b5cf6"/>
              <circle cx="87" cy="92" r="1.5" fill="#8b5cf6"/>
              <circle cx="22" cy="50" r="1.5" fill="#8b5cf6"/>
              <circle cx="98" cy="50" r="1.5" fill="#8b5cf6"/>
            </g>
            
            <!-- Floating particles -->
            <g class="floating-particles">
              <circle class="particle-1" r="1.5" fill="#fff"/>
              <circle class="particle-2" r="1.5" fill="#fff"/>
              <circle class="particle-3" r="1.5" fill="#fff"/>
              <circle class="particle-4" r="1.5" fill="#fff"/>
              <circle class="particle-5" r="1.5" fill="#fff"/>
              <circle class="particle-6" r="1.5" fill="#fff"/>
            </g>
            
            <!-- Central pulse -->
            <circle cx="60" cy="60" r="4" fill="#fff" class="central-pulse"/>
            
            <!-- Gradients -->
            <defs>
              <radialGradient id="somaGradient" cx="30%" cy="30%">
                <stop offset="0%" stop-color="#fff"/>
                <stop offset="30%" stop-color="#00f5ff"/>
                <stop offset="70%" stop-color="#0080ff"/>
                <stop offset="100%" stop-color="#1a1a2e"/>
              </radialGradient>
              <linearGradient id="dendriteGradient" x1="0%" y1="0%" x2="100%" y2="100%">
                <stop offset="0%" stop-color="#00f5ff"/>
                <stop offset="30%" stop-color="#0080ff"/>
                <stop offset="70%" stop-color="#8b5cf6"/>
                <stop offset="100%" stop-color="#1a1a2e"/>
              </linearGradient>
              <radialGradient id="glowRing" cx="50%" cy="50%">
                <stop offset="0%" stop-color="transparent"/>
                <stop offset="80%" stop-color="transparent"/>
                <stop offset="100%" stop-color="#00f5ff"/>
              </radialGradient>
            </defs>
          </svg>
        </div>
        ${showText ? this.getTextTemplate() : ''}
      </div>
    `;
  }

  /**
   * Get text template based on size
   */
  getTextTemplate() {
    const { textSize } = this.options;
    
    if (textSize === 'large') {
      return `
        <div class="knirv-text-container">
          <h1 class="knirv-title-large">KNIRV</h1>
          <div class="knirv-divider"></div>
          <h2 class="knirv-subtitle-large">Network</h2>
          <div class="knirv-dots">
            <div class="dot dot-1"></div>
            <div class="dot dot-2"></div>
            <div class="dot dot-3"></div>
            <div class="dot dot-4"></div>
          </div>
        </div>
      `;
    } else {
      return `
        <div class="knirv-text-container">
          <h1 class="knirv-title-small">KNIRV</h1>
          <h2 class="knirv-subtitle-small">Network</h2>
        </div>
      `;
    }
  }

  /**
   * Add CSS styles
   */
  addStyles() {
    if (document.getElementById('animated-knirv-logo-styles')) return;
    
    const style = document.createElement('style');
    style.id = 'animated-knirv-logo-styles';
    style.textContent = `
      .animated-knirv-logo {
        display: inline-block;
      }
      
      .knirv-logo-svg {
        filter: drop-shadow(0 0 10px rgba(0, 245, 255, 0.3)) drop-shadow(0 0 20px rgba(139, 92, 246, 0.2));
        transition: filter 0.3s ease;
      }
      
      .glow-ring {
        animation: pulse-ring 2s ease-in-out infinite;
      }
      
      .terminal-nodes {
        animation: pulse-nodes 3s ease-in-out infinite;
      }
      
      .central-pulse {
        animation: pulse-center 1.5s ease-in-out infinite;
      }
      
      .floating-particles .particle-1 { animation: float-1 4s ease-in-out infinite; }
      .floating-particles .particle-2 { animation: float-2 3.5s ease-in-out infinite 0.5s; }
      .floating-particles .particle-3 { animation: float-3 4.5s ease-in-out infinite 1s; }
      .floating-particles .particle-4 { animation: float-4 3.8s ease-in-out infinite 1.5s; }
      .floating-particles .particle-5 { animation: float-5 4.2s ease-in-out infinite 2s; }
      .floating-particles .particle-6 { animation: float-6 3.9s ease-in-out infinite 2.5s; }
      
      .knirv-title-large {
        font-size: 2.5rem;
        font-weight: 700;
        background: linear-gradient(135deg, #a855f7, #ec4899, #8b5cf6);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        background-clip: text;
        font-family: 'Orbitron', 'Arial Black', sans-serif;
        letter-spacing: 0.1em;
        margin: 0;
        text-shadow: 0 0 20px rgba(168, 85, 247, 0.3);
      }
      
      .knirv-subtitle-large {
        font-size: 1rem;
        font-weight: 300;
        color: #d1d5db;
        font-family: 'Rajdhani', 'Arial', sans-serif;
        letter-spacing: 0.3em;
        text-transform: uppercase;
        margin: 0.25rem 0 0 0;
      }
      
      .knirv-title-small {
        font-size: 1.5rem;
        font-weight: 700;
        background: linear-gradient(135deg, #a855f7, #ec4899, #8b5cf6);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        background-clip: text;
        font-family: 'Orbitron', 'Arial Black', sans-serif;
        letter-spacing: 0.1em;
        margin: 0;
        line-height: 1;
      }
      
      .knirv-subtitle-small {
        font-size: 0.75rem;
        font-weight: 300;
        color: #9ca3af;
        font-family: 'Rajdhani', 'Arial', sans-serif;
        letter-spacing: 0.2em;
        text-transform: uppercase;
        margin: 0;
        line-height: 1;
      }
      
      .knirv-divider {
        height: 1px;
        background: linear-gradient(90deg, transparent, #8b5cf6, transparent);
        margin: 0.25rem 0;
        animation: pulse-divider 2s ease-in-out infinite;
      }
      
      .knirv-dots {
        display: flex;
        gap: 0.25rem;
        margin-top: 0.5rem;
      }
      
      .dot {
        width: 6px;
        height: 6px;
        border-radius: 50%;
        animation: pulse-dots 2s ease-in-out infinite;
      }
      
      .dot-1 { background: #a855f7; animation-delay: 0s; }
      .dot-2 { background: #ec4899; animation-delay: 0.2s; }
      .dot-3 { background: #8b5cf6; animation-delay: 0.4s; }
      .dot-4 { background: #ec4899; animation-delay: 0.6s; }
      
      @keyframes pulse-ring {
        0%, 100% { opacity: 0.3; }
        50% { opacity: 0.8; }
      }
      
      @keyframes pulse-nodes {
        0%, 100% { opacity: 0.6; }
        50% { opacity: 1; }
      }
      
      @keyframes pulse-center {
        0%, 100% { opacity: 0.6; transform: scale(1); }
        50% { opacity: 1; transform: scale(1.2); }
      }
      
      @keyframes pulse-divider {
        0%, 100% { opacity: 0.5; }
        50% { opacity: 1; }
      }
      
      @keyframes pulse-dots {
        0%, 100% { opacity: 0.4; transform: scale(1); }
        50% { opacity: 1; transform: scale(1.2); }
      }
      
      @keyframes float-1 {
        0%, 100% { cx: 45; cy: 45; opacity: 0.3; }
        50% { cx: 75; cy: 75; opacity: 0.8; }
      }
      
      @keyframes float-2 {
        0%, 100% { cx: 75; cy: 45; opacity: 0.3; }
        50% { cx: 45; cy: 75; opacity: 0.8; }
      }
      
      @keyframes float-3 {
        0%, 100% { cx: 45; cy: 75; opacity: 0.3; }
        50% { cx: 75; cy: 45; opacity: 0.8; }
      }
      
      @keyframes float-4 {
        0%, 100% { cx: 75; cy: 75; opacity: 0.3; }
        50% { cx: 45; cy: 45; opacity: 0.8; }
      }
      
      @keyframes float-5 {
        0%, 100% { cx: 35; cy: 60; opacity: 0.3; }
        50% { cx: 85; cy: 60; opacity: 0.8; }
      }
      
      @keyframes float-6 {
        0%, 100% { cx: 60; cy: 35; opacity: 0.3; }
        50% { cx: 60; cy: 85; opacity: 0.8; }
      }
    `;
    
    document.head.appendChild(style);
  }

  /**
   * Start the animation loop
   */
  startAnimation() {
    const animate = () => {
      this.glowIntensity = 0.3 + Math.sin(Date.now() * 0.002) * 0.2;
      
      if (this.element) {
        const svg = this.element.querySelector('.knirv-logo-svg');
        if (svg) {
          svg.style.filter = `drop-shadow(0 0 ${20 * this.glowIntensity}px rgba(0, 245, 255, 0.4)) drop-shadow(0 0 ${40 * this.glowIntensity}px rgba(139, 92, 246, 0.3))`;
        }
        
        const glowRing = this.element.querySelector('.glow-ring');
        if (glowRing) {
          glowRing.style.opacity = this.glowIntensity * 0.6;
        }
        
        const centralPulse = this.element.querySelector('.central-pulse');
        if (centralPulse) {
          centralPulse.style.opacity = this.glowIntensity * 0.8;
        }
      }
      
      this.animationId = requestAnimationFrame(animate);
    };
    
    animate();
  }

  /**
   * Stop the animation
   */
  stopAnimation() {
    if (this.animationId) {
      cancelAnimationFrame(this.animationId);
      this.animationId = null;
    }
  }

  /**
   * Destroy the component
   */
  destroy() {
    this.stopAnimation();
    if (this.element && this.element.parentNode) {
      this.element.parentNode.removeChild(this.element);
    }
    this.element = null;
  }

  /**
   * Update glow intensity manually
   */
  setGlowIntensity(intensity) {
    this.glowIntensity = Math.max(0, Math.min(1, intensity));
  }
}

// Export for module usage
if (typeof module !== 'undefined' && module.exports) {
  module.exports = AnimatedKNIRVLogo;
}
