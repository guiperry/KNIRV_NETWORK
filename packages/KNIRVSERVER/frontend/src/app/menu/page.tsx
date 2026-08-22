'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';

// Retain the old URL for bookmarked links, but the constellation menu now
// belongs to the gateway WebGUI rather than the KNIRVSERVER console.
export default function MenuPage() {
  const router = useRouter();

  useEffect(() => {
    router.replace('/');
  }, [router]);

  return null;
}
