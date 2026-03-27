// KNIRV Network Bundle JS
// This file contains bundled JavaScript for the KNIRV network website

console.log('KNIRV Network Bundle loaded');

// Add any global utilities or initializations here

// Initialize any global variables
window.KNIRV = window.KNIRV || {};
window.KNIRV.version = '1.0.0';
window.KNIRV.network = 'testnet';

// Basic error handling
window.addEventListener('error', function(e) {
    console.error('KNIRV Network Error:', e.error);
});

// Basic unhandled promise rejection handling
window.addEventListener('unhandledrejection', function(e) {
    console.error('KNIRV Network Unhandled Promise Rejection:', e.reason);
});