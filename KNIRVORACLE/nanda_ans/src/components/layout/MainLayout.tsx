// src/components/layout/MainLayout.tsx
import type { ReactNode } from 'react';
import Header from './Header';
import Footer from './Footer';
import { Toaster } from "@/components/ui/toaster";


interface MainLayoutProps {
  children: ReactNode;
}

const MainLayout = ({ children }: MainLayoutProps) => {
  return (
    <div className="flex flex-col min-h-screen">
      <Header />
      <main className="flex-grow container mx-auto px-4 py-8 md:px-6 lg:px-8">
        {children}
      </main>
      <Footer />
      <Toaster />
    </div>
  );
};

export default MainLayout;
