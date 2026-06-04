import React from 'react'
import Link from 'next/link'
import { Network } from 'lucide-react'

export default function KnirvLogo() {
  return (
    <Link href="/" data-config-nav="home" className="flex items-center gap-3 group" aria-label="Go to homepage">
      <div className="relative">
        <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-blue-500 to-cyan-400 flex items-center justify-center">
          <Network className="w-4 h-4 text-white" />
        </div>
        <div className="absolute inset-0 rounded-lg bg-blue-500/40 blur-sm -z-10" />
      </div>
      <span className="text-xl font-black tracking-tight text-white">KNIRV</span>
    </Link>
  )
}
