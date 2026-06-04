import React from 'react'
import Link from 'next/link'
import Image from 'next/image'

export default function KnirvLogo() {
  return (
    <Link href="/" data-config-nav="home" className="flex items-center gap-3 group" aria-label="Go to homepage">
      <div className="relative flex h-11 w-11 items-center justify-center rounded-2xl border border-white/10 bg-[#0d1628] shadow-[0_0_0_1px_rgba(0,212,255,0.08),0_18px_40px_rgba(0,0,0,0.25)]">
        <Image
          src="/logo/knirv-logo.png"
          alt="KNIRV Logo"
          fill
          style={{ objectFit: 'contain' }}
          className="drop-shadow-[0_0_20px_rgba(0,212,255,0.18)]"
        />
      </div>
      <div className="leading-none">
        <span className="block text-2xl font-extrabold tracking-tight knirv-gradient-text">KNIRV</span>
        <div className="mt-1 text-[10px] leading-tight tracking-[0.22em] text-white/60 uppercase">
          Key Neural Intelligence
          <br />
          Reasoning Validation
        </div>
      </div>
    </Link>
  )
}
