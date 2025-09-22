// src/app/not-found.tsx
import Link from 'next/link';
import { Button } from '@/components/ui/button';
import { FileSearch } from 'lucide-react';

export default function NotFound() {
  return (
    <div className="flex flex-col items-center justify-center text-center py-20">
      <FileSearch className="w-24 h-24 text-primary mb-8" />
      <h1 className="text-5xl font-bold text-foreground mb-4">404 - Page Not Found</h1>
      <p className="text-xl text-muted-foreground mb-8 max-w-md">
        Oops! The page you are looking for does not exist. It might have been moved or deleted.
      </p>
      <Button asChild size="lg">
        <Link href="/">Go Back to Discovery</Link>
      </Button>
    </div>
  );
}
