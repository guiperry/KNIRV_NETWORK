// src/app/error.tsx
"use client"; 

import { useEffect } from 'react';
import { Button } from '@/components/ui/button';
import { AlertTriangle } from 'lucide-react';

export default function Error({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    console.error(error);
  }, [error]);

  return (
    <div className="flex flex-col items-center justify-center text-center py-20 px-4">
      <AlertTriangle className="w-20 h-20 text-destructive mb-8" />
      <h2 className="text-4xl font-bold text-foreground mb-4">Something went wrong!</h2>
      <p className="text-lg text-muted-foreground mb-6 max-w-lg">
        We encountered an unexpected issue. Please try again. If the problem persists, contact support.
      </p>
      {error.message && <p className="text-sm text-muted-foreground mb-6">Error: {error.message}</p>}
      <Button
        onClick={() => reset()}
        size="lg"
      >
        Try again
      </Button>
    </div>
  );
}
