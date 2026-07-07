'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import dynamic from 'next/dynamic';

const ConstellationMenu = dynamic(
  () => import('@/components/menu/constellation-menu'),
  { loading: () => <div className="min-h-screen bg-[#030a18]" /> }
);

export default function MenuPage() {
  const router = useRouter();
  const [authenticated, setAuthenticated] = useState(false);

  useEffect(() => {
    const token = localStorage.getItem('knirv_nexus_token') || localStorage.getItem('knirv_auth_token');
    if (!token) {
      router.push('/login');
    } else {
      setAuthenticated(true);
    }
  }, [router]);

  if (!authenticated) return <div className="min-h-screen bg-[#030a18]" />;
  return <ConstellationMenu />;
}
