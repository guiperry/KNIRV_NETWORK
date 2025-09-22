import React from 'react'
import { Network } from 'lucide-react'
import Link from 'next/link'

export default function KnirvLogo() {
  return (
    <Link href="/" data-config-nav="home" className="flex items-center space-x-2" aria-label="Go to homepage">
      <Network className="h-8 w-8 knirv-text-primary" />
      <div className="leading-none">
        <span className="text-2xl font-bold knirv-gradient-text">KNIRV</span>
        <div className="text-xs text-white/60">CORTEX</div>
      </div>
    </Link>
  )
}
