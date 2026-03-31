/** @type {import('tailwindcss').Config} */
export default {
  content: [
    './index.html',
    './src/**/*.{js,ts,jsx,tsx}',
    './src/**/*.{html,js,ts,jsx,tsx}',
    './src/components/game/**/*.{js,ts,jsx,tsx}',
  ],
  theme: {
    extend: {
      colors: {
        'knirv-blue': '#3B82F6',
        'knirv-teal': '#14B8A6',
        'knirv-purple': '#8B5CF6',
        // Gaming arena colors
        'game-primary': '#000510',
        'game-accent': '#00ffff',
        'game-secondary': '#4400ff',
        'game-success': '#00ff00',
        'game-error': '#ff0000',
        'game-warning': '#ff6600',
        'cyberpunk': '#00ffff',
        'neon-purple': '#8800ff',
      },
      animation: {
        'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        'spin-slow': 'spin 3s linear infinite',
        // Gaming animations
        'float': 'float 4s ease-in-out infinite',
        'glow': 'glow 2s ease-in-out infinite alternate',
        'neon-flicker': 'neon-flicker 1.5s infinite alternate',
      },
    },
  },
  plugins: [],
  // Custom animations for game effects
  keyframes: {
    float: {
      '0%, 100%': { transform: 'translateY(0px)' },
      '50%': { transform: 'translateY(-10px)' },
    },
    glow: {
      '0%, 100%': { boxShadow: '0 0 5px rgba(0, 255, 255, 0.5)' },
      '50%': { boxShadow: '0 0 20px rgba(0, 255, 255, 0.8), 0 0 30px rgba(0, 255, 255, 0.6)' },
    },
    'neon-flicker': {
      '0%, 100%': { opacity: '1' },
      '50%': { opacity: '0.8' },
    },
  },
};
