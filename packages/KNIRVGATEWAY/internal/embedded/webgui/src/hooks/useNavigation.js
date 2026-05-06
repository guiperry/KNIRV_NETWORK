import { useRouter } from 'next/router';
import { useState, useEffect } from 'react';
import { useRole } from '../contexts/RoleContext';

/**
 * Returns the gateway's base URL for navigation, falling back to relative.
 * window.__GATEWAY_BASE__ is injected by the gateway server (injectGatewayBase)
 * and points to http://localhost:8080.  When set, navigation links constructed
 * by getPageUrl() use this as the base so that pages served at the gateway
 * (port 8080) are reachable even when the SPA is embedded in another context
 * like the KNIRVSERVER dashboard (port 8090).
 */
function getGatewayBase() {
  if (typeof window !== 'undefined' && window.__GATEWAY_BASE__) {
    return window.__GATEWAY_BASE__;
  }
  return '';
}

/**
 * Builds a full URL for a page, using the gateway base when available so
 * navigation escapes to the gateway server instead of the current host.
 * Example: getPageUrl('network-monitor') → 'http://localhost:8080/network-monitor'
 * or just '/network-monitor' when no gateway base is configured.
 */
function getPageUrl(page) {
  const base = getGatewayBase();
  if (base) {
    // Strip trailing slash and append the page path
    const normalized = base.replace(/\/+$/, '');
    return `${normalized}/${page}`;
  }
  return `/${page}`;
}

/**
 * Custom hook for handling navigation between pages with role-based access control
 * and gateway-aware URL resolution.
 * @param {string} initialPage - The initial active page
 * @returns {Object} - Object containing activePage state and handleNavigation function
 */
export function useNavigation(initialPage) {
  const router = useRouter();
  const { canAccess } = useRole();
  
  // Initialize with the provided initialPage
  const [activePage, setActivePage] = useState(initialPage);

  // Update activePage when the component mounts and when the route changes
  useEffect(() => {
    if (!router.isReady) return;

    // Function to get the current page from the URL
    const getCurrentPage = () => {
      // Get the path without the leading slash
      const path = router.pathname.substring(1);

      // If we're on the root path ("/"), use the initialPage
      if (!path) {
        return initialPage;
      }

      // Otherwise, use the current path
      return path;
    };

    // Get the current page from the URL
    const currentPage = getCurrentPage();

    // Update the activePage state
    setActivePage(currentPage);
  }, [router.isReady, router.pathname, initialPage]);

  // Function to handle navigation between pages
  const handleNavigation = (page) => {
    // Check if the user can access this page
    if (canAccess(page)) {
      // Set the active page immediately for a responsive UI
      setActivePage(page);

      // Build the full URL using the gateway base when available.
      // This ensures that pages served at the gateway (e.g. network-monitor,
      // chain-explorer, oracle-explorer) open correctly even when the SPA is
      // embedded in the KNIRVSERVER dashboard context.
      const url = getPageUrl(page);
      router.push(url);
    } else {
      console.warn(`Access to ${page} is not allowed for your role.`);
      // Optionally show a notification to the user
    }
  };

  return { activePage, handleNavigation };
}
