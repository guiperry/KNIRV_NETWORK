'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';

export default function MenuPage() {
  const router = useRouter();
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // Verify auth
    const token = localStorage.getItem('knirv_auth_token');
    if (!token) {
      router.push('/login');
      return;
    }

    // Listen for navigation events from menu iframe
    const handleMessage = (event: MessageEvent) => {
      const { type, section } = event.data || {};
      
      if (type === 'navigate') {
        if (section) {
          router.push(`/?nav=${encodeURIComponent(section)}`);
        } else {
          router.push('/');
        }
      }
    };

    window.addEventListener('message', handleMessage);
    setLoading(false);

    return () => window.removeEventListener('message', handleMessage);
  }, [router]);

  if (loading) return <div className="min-h-screen bg-black flex items-center justify-center text-white">Loading menu...</div>;

  return (
    <div className="min-h-screen bg-black">
      <iframe
        id="menu-iframe"
        src="/menu/"
        frameBorder={0}
        loading="lazy"
        className="w-full h-screen border-none bg-transparent"
        allow="autoplay"
      />
    </div>
  );
}