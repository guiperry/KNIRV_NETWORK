import React from 'react';
import Image from 'next/image';

const KnirvNetworkLogo = () => {
  return (
    <div className="flex items-center justify-center w-full h-full bg-gradient-to-br from-gray-900 via-black to-slate-900 p-4">
      <div className="relative w-full h-full flex items-center justify-center">
        <Image 
          src="/logo/knirv-logo.png" 
          alt="KNIRV Logo" 
          fill
          style={{ objectFit: 'contain' }}
          className="drop-shadow-2xl"
        />
      </div>
    </div>
  );
};

export default KnirvNetworkLogo;
