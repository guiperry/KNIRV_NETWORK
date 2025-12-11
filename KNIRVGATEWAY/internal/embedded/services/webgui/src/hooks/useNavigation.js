import { useRouter } from 'next/router';
import { useState, useEffect } from 'react';
import { useRole } from '../contexts/RoleContext';

/**
 * Custom hook for handling navigation between pages with role-based access control
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

      // Navigate to the new page
      router.push(`/${page}`);
    } else {
      console.warn(`Access to ${page} is not allowed for your role.`);
      // Optionally show a notification to the user
    }
  };

  return { activePage, handleNavigation };
}