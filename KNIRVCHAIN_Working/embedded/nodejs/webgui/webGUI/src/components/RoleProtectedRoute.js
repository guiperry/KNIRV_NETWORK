import { useEffect } from 'react';
import { useRouter } from 'next/router';
import { useRole } from '../contexts/RoleContext';

export default function RoleProtectedRoute({ children }) {
  const { role, canAccess } = useRole();
  const router = useRouter();
  const currentPage = router.pathname.substring(1) || 'dashboard';
  
  useEffect(() => {
    // If the user can't access this page, redirect to a page they can access
    if (!canAccess(currentPage)) {
      // Find the first accessible page for their role
      const accessiblePages = {
        Root: 'dashboard',
        Bootnode: 'dashboard',
        Peer: 'dashboard',
        Client: 'inventory'
      };
      
      const redirectPage = accessiblePages[role] || 'inventory';
      router.push(`/${redirectPage}`);
    }
  }, [role, currentPage, canAccess, router]);
  
  // If they can access the page, render the children
  return canAccess(currentPage) ? children : null;
}