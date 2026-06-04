import React from 'react';
import Image from 'next/image';

const KnirvNetworkLogo = () => {
  return (
    <div className="flex h-full w-full items-center justify-center rounded-[inherit] bg-[radial-gradient(circle_at_top_left,rgba(0,212,255,0.08),transparent_40%),linear-gradient(180deg,#0d1628_0%,#080f1e_100%)] p-4">
      <div className="relative flex h-full w-full items-center justify-center">
        <Image 
          src="/logo/knirv-logo.png" 
          alt="KNIRV Logo" 
          fill
          style={{ objectFit: 'contain' }}
          className="drop-shadow-[0_0_28px_rgba(0,212,255,0.18)]"
        />
      </div>
    </div>
  );
};

export default KnirvNetworkLogo;
