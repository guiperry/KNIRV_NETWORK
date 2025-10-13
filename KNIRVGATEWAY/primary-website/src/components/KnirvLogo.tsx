import React from 'react'
import Link from 'next/link'
import Image from 'next/image'

export default function KnirvLogo() {
  return (
    <Link href="/" data-config-nav="home" className="flex items-center space-x-3" aria-label="Go to homepage">
      <div className="relative w-10 h-10">
        <Image 
          src="/logo/knirv-logo.png" 
          alt="KNIRV Logo" 
          fill
          style={{ objectFit: 'contain' }}
          className="drop-shadow-lg"
        />
      </div>
      <div className="leading-none">
        <span className="text-2xl font-bold knirv-gradient-text">KNIRV</span>
        <div className="text-[10px] leading-tight text-white/60">Key Neural Intelligence<br />Reasoning Validation</div>
      </div>
    </Link>
  )
}
