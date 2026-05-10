"use client"

import { useEffect, useState } from "react"
import { useRouter } from "next/navigation"
import { motion } from "framer-motion"
import { Eye, Box, Globe, Cpu, Layers, Settings, Triangle, User } from "lucide-react"
import dynamic from "next/dynamic"

const StarSupernova = dynamic(() => import("./star-supernova"), {
  loading: () => <div className="fixed inset-0 bg-[#030a18] z-50" />,
})
const SettingsModal = dynamic(() => import("./settings-modal"), {
  loading: () => null,
})

const innerLabels = [
  { label: "KNIRVCLI", angle: 210 },
  { label: "KNIRVCTCLI", angle: 250 },
  { label: "KNIRVCHAIN", angle: 290 },
  { label: "KNIRVCCHAIN", angle: 330 },
  { label: "KNIRVCINEXUS", angle: 10 },
  { label: "KNIRVROUTER", angle: 50 },
  { label: "KNIRVCONTROLLER", angle: 90 },
  { label: "KNIRVCLLS", angle: 130 },
  { label: "KNIRVCWORKER", angle: 170 },
]

const middleLabels = [
  { label: "KNIRVCORPAY", angle: 230 },
  { label: "KNIRVASDK", angle: 270 },
  { label: "KNIRVCOGINS", angle: 310 },
  { label: "KNIRVCORTEX", angle: 350 },
  { label: "KNIRVGATEWAY", angle: 30 },
  { label: "KNIRVCONTROLLER", angle: 70 },
  { label: "KNIRVROUTET", angle: 110 },
  { label: "KNIRVBALI", angle: 150 },
  { label: "KNIRVTESTNET", angle: 190 },
]

// section: null opens local settings modal
const outerIcons = [
  { icon: Triangle, angle: 0,   section: "setup",      label: "Setup" },
  { icon: Globe,    angle: 45,  section: "p2p-webgui", label: "WebGUI" },
  { icon: User,     angle: 90,  section: "admin",      label: "Admin" },
  { icon: Cpu,      angle: 135, section: "cognitive",  label: "Cognitive" },
  { icon: Layers,   angle: 180, section: "nodes",      label: "DVE Nodes" },
  { icon: Settings, angle: 225, section: null,         label: "Settings" },
  { icon: Eye,      angle: 270, section: "profile",    label: "Reports" },
  { icon: Box,      angle: 315, section: "badgelab",   label: "Badge Lab" },
]

function polarToCartesian(angle: number, radius: number) {
  const radian = (angle - 90) * (Math.PI / 180)
  return {
    x: Math.cos(radian) * radius,
    y: Math.sin(radian) * radius,
  }
}

export default function ConstellationMenu() {
  const router = useRouter()
  const [mounted, setMounted] = useState(false)
  const [hoveredIcon, setHoveredIcon] = useState<number | null>(null)
  const [loadingComplete, setLoadingComplete] = useState(false)
  const [settingsOpen, setSettingsOpen] = useState(false)

  useEffect(() => {
    setMounted(true)
  }, [])

  const handleLoadingComplete = () => {
    setLoadingComplete(true)
  }

  const handleIconClick = (section: string | null) => {
    if (section === null) {
      setSettingsOpen(true)
      return
    }
    if (window.parent && window.parent !== window) {
      window.parent.postMessage({ type: 'navigate', section }, '*')
    } else {
      router.push(`/?nav=${section}`)
    }
  }

  const centerSize = 160
  const ring1Radius = 120
  const ring2Radius = 180
  const ring3Radius = 250
  const ring4Radius = 320
  const iconRadius = 360

  return (
    <>
      {!loadingComplete && <StarSupernova onComplete={handleLoadingComplete} />}

      <div
        className="relative flex min-h-screen w-full items-center justify-center overflow-hidden"
        style={{ backgroundColor: "#030a18" }}
      >
        <div className="relative">
          {/* Starfield */}
          <div className="absolute inset-0">
            {[...Array(200)].map((_, i) => (
              <motion.div
                key={i}
                className="absolute rounded-full bg-white"
                style={{
                  left: `${Math.random() * 100}%`,
                  top: `${Math.random() * 100}%`,
                  width: Math.random() > 0.9 ? 2 : 1,
                  height: Math.random() > 0.9 ? 2 : 1,
                  opacity: Math.random() * 0.6 + 0.2,
                }}
                animate={{ opacity: [0.2, 0.6, 0.2] }}
                transition={{
                  duration: Math.random() * 4 + 2,
                  repeat: Infinity,
                  ease: "easeInOut",
                  delay: Math.random() * 2,
                }}
              />
            ))}
          </div>

          {/* Bottom Horizon Glow */}
          <div className="pointer-events-none absolute bottom-0 left-0 right-0 h-48">
            <div className="absolute inset-0 bg-gradient-to-t from-blue-500/10 via-blue-500/5 to-transparent" />
            <div className="absolute bottom-12 left-1/2 h-px w-[80%] -translate-x-1/2 bg-gradient-to-r from-transparent via-cyan-400/60 to-transparent" />
            <div className="absolute bottom-0 left-[10%] h-16 w-4 bg-gradient-to-t from-blue-900/30 to-transparent" style={{ clipPath: 'polygon(50% 0%, 100% 100%, 0% 100%)' }} />
            <div className="absolute bottom-0 left-[15%] h-24 w-6 bg-gradient-to-t from-blue-900/20 to-transparent" style={{ clipPath: 'polygon(50% 0%, 100% 100%, 0% 100%)' }} />
            <div className="absolute bottom-0 right-[10%] h-20 w-5 bg-gradient-to-t from-blue-900/30 to-transparent" style={{ clipPath: 'polygon(50% 0%, 100% 100%, 0% 100%)' }} />
            <div className="absolute bottom-0 right-[15%] h-28 w-7 bg-gradient-to-t from-blue-900/20 to-transparent" style={{ clipPath: 'polygon(50% 0%, 100% 100%, 0% 100%)' }} />
          </div>

          {/* Main Container */}
          <div className="relative" style={{ width: 900, height: 900 }}>
            {/* SVG Rings */}
            <svg
              className="absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2"
              width="900"
              height="900"
              viewBox="-450 -450 900 900"
            >
              <motion.circle cx={0} cy={0} r={ring4Radius} fill="none" stroke="rgba(72, 136, 255,0.3)" strokeWidth="1.5"
                initial={{ pathLength: 0, opacity: 0 }} animate={loadingComplete ? { pathLength: 1, opacity: 1 } : {}} transition={{ duration: 2, delay: 0.2 }} />
              <motion.circle cx={0} cy={0} r={355} fill="none" stroke="rgba(72, 136, 255,0.2)" strokeWidth="1"
                initial={{ pathLength: 0, opacity: 0 }} animate={loadingComplete ? { pathLength: 1, opacity: 1 } : {}} transition={{ duration: 2, delay: 0.1 }} />
              <motion.circle cx={0} cy={0} r={365} fill="none" stroke="rgba(72, 136, 255,0.2)" strokeWidth="1"
                initial={{ pathLength: 0, opacity: 0 }} animate={loadingComplete ? { pathLength: 1, opacity: 1 } : {}} transition={{ duration: 2, delay: 0.1 }} />
              <motion.circle cx={0} cy={0} r={ring3Radius} fill="none" stroke="rgba(72, 136, 255,0.25)" strokeWidth="1.5"
                initial={{ pathLength: 0, opacity: 0 }} animate={loadingComplete ? { pathLength: 1, opacity: 1 } : {}} transition={{ duration: 2, delay: 0.4 }} />
              <motion.circle cx={0} cy={0} r={ring2Radius} fill="none" stroke="rgba(72, 136, 255,0.35)" strokeWidth="1.5"
                initial={{ pathLength: 0, opacity: 0 }} animate={loadingComplete ? { pathLength: 1, opacity: 1 } : {}} transition={{ duration: 2, delay: 0.6 }} />
              <motion.circle cx={0} cy={0} r={ring1Radius} fill="none" stroke="rgba(72, 136, 255,0.4)" strokeWidth="2"
                initial={{ pathLength: 0, opacity: 0 }} animate={loadingComplete ? { pathLength: 1, opacity: 1 } : {}} transition={{ duration: 2, delay: 0.8 }} />

              {/* Gear Ring */}
              <motion.g initial={{ opacity: 0, scale: 0.8 }} animate={mounted ? { opacity: 1, scale: 1 } : {}} transition={{ duration: 1, delay: 1 }}>
                {[...Array(24)].map((_, i) => {
                  const gearRadius = 145
                  const toothHeight = 8
                  const angle = (i * 360) / 24
                  const angleRad = (angle * Math.PI) / 180
                  const innerR = gearRadius - toothHeight / 2
                  const outerR = gearRadius + toothHeight / 2
                  const x1 = Math.cos(angleRad - 0.08) * innerR; const y1 = Math.sin(angleRad - 0.08) * innerR
                  const x2 = Math.cos(angleRad + 0.08) * innerR; const y2 = Math.sin(angleRad + 0.08) * innerR
                  const x3 = Math.cos(angleRad + 0.06) * outerR; const y3 = Math.sin(angleRad + 0.06) * outerR
                  const x4 = Math.cos(angleRad - 0.06) * outerR; const y4 = Math.sin(angleRad - 0.06) * outerR
                  return (
                    <polygon key={`tooth-${i}`} points={`${x1},${y1} ${x2},${y2} ${x3},${y3} ${x4},${y4}`}
                      fill="rgba(72, 136, 255,0.25)" stroke="rgba(72, 136, 255,0.4)" strokeWidth="0.5" />
                  )
                })}
                <circle cx={0} cy={0} r={137} fill="none" stroke="rgba(72, 136, 255,0.3)" strokeWidth="1" />
                <circle cx={0} cy={0} r={153} fill="none" stroke="rgba(72, 136, 255,0.3)" strokeWidth="1" />
              </motion.g>

              <defs>
                <filter id="glow" x="-50%" y="-50%" width="200%" height="200%">
                  <feGaussianBlur stdDeviation="3" result="coloredBlur"/>
                  <feMerge><feMergeNode in="coloredBlur"/><feMergeNode in="SourceGraphic"/></feMerge>
                </filter>
                <filter id="strongGlow" x="-50%" y="-50%" width="200%" height="200%">
                  <feGaussianBlur stdDeviation="6" result="coloredBlur"/>
                  <feMerge><feMergeNode in="coloredBlur"/><feMergeNode in="SourceGraphic"/></feMerge>
                </filter>
              </defs>

              {/* Connection nodes */}
              {[...Array(16)].map((_, i) => {
                const angle = (i * 360) / 16
                const pos1 = polarToCartesian(angle, ring1Radius)
                const pos2 = polarToCartesian(angle, ring2Radius)
                const pos3 = polarToCartesian(angle, ring3Radius)
                const pos4 = polarToCartesian(angle, ring4Radius)
                const isCardinal = i % 2 === 0
                return (
                  <g key={`nodes-${i}`}>
                    <motion.circle cx={pos1.x} cy={pos1.y} r={isCardinal ? 4.5 : 3} fill="#4888ff" filter="url(#glow)"
                      initial={{ scale: 0, opacity: 0 }} animate={mounted ? { scale: 1, opacity: isCardinal ? 1 : 0.8 } : {}} transition={{ duration: 0.3, delay: 1 + i * 0.05 }} />
                    <motion.circle cx={pos2.x} cy={pos2.y} r={2.5} fill="#4888ff" filter="url(#glow)"
                      initial={{ scale: 0, opacity: 0 }} animate={mounted ? { scale: 1, opacity: 0.6 } : {}} transition={{ duration: 0.3, delay: 1.2 + i * 0.05 }} />
                    <motion.circle cx={pos3.x} cy={pos3.y} r={2} fill="#4888ff" filter="url(#glow)"
                      initial={{ scale: 0, opacity: 0 }} animate={mounted ? { scale: 1, opacity: 0.5 } : {}} transition={{ duration: 0.3, delay: 1.4 + i * 0.05 }} />
                    <motion.circle cx={pos4.x} cy={pos4.y} r={1.5} fill="#4888ff"
                      initial={{ scale: 0, opacity: 0 }} animate={mounted ? { scale: 1, opacity: 0.4 } : {}} transition={{ duration: 0.3, delay: 1.6 + i * 0.05 }} />
                  </g>
                )
              })}

              {/* Radial lines */}
              {[...Array(12)].map((_, i) => {
                const angle = (i * 360) / 12
                const inner = polarToCartesian(angle, ring1Radius - 20)
                const outer = polarToCartesian(angle, ring4Radius + 20)
                return (
                  <motion.line key={`radial-${i}`} x1={inner.x} y1={inner.y} x2={outer.x} y2={outer.y}
                    stroke="rgba(72, 136, 255,0.1)" strokeWidth="1"
                    initial={{ pathLength: 0, opacity: 0 }} animate={loadingComplete ? { pathLength: 1, opacity: 1 } : {}} transition={{ duration: 1, delay: 1.5 + i * 0.05 }} />
                )
              })}

              {/* Rotating arc segments */}
              <motion.g animate={loadingComplete ? { rotate: 360 } : {}} transition={{ duration: 120, repeat: Number.POSITIVE_INFINITY, ease: "linear" }}>
                {[45, 135, 225, 315].map((startAngle, i) => {
                  const r = ring3Radius + 15
                  const start = polarToCartesian(startAngle, r)
                  const end = polarToCartesian(startAngle + 30, r)
                  return (
                    <motion.path key={`arc-${i}`} d={`M ${start.x} ${start.y} A ${r} ${r} 0 0 1 ${end.x} ${end.y}`}
                      fill="none" stroke="rgba(72, 136, 255,0.4)" strokeWidth="3" filter="url(#glow)"
                      initial={{ pathLength: 0, opacity: 0 }} animate={loadingComplete ? { pathLength: 1, opacity: 1 } : {}} transition={{ duration: 0.8, delay: 2 + i * 0.1 }} />
                  )
                })}
              </motion.g>

              {/* Pulse effects */}
              {mounted && (
                <>
                  <motion.circle cx={0} cy={0} r={ring2Radius} fill="none" stroke="rgba(100, 160, 255, 0.6)"
                    strokeWidth="2" strokeDasharray="10 20" filter="url(#strongGlow)"
                    animate={{ rotate: [0, 360] }} transition={{ duration: 20, repeat: Infinity, ease: "linear" }}
                    style={{ transformOrigin: "center" }} />
                  <motion.circle cx={0} cy={0} r={ring3Radius} fill="none" stroke="rgba(100, 160, 255, 0.4)"
                    strokeWidth="1.5" strokeDasharray="5 30" filter="url(#glow)"
                    animate={{ rotate: [360, 0] }} transition={{ duration: 30, repeat: Infinity, ease: "linear" }}
                    style={{ transformOrigin: "center" }} />
                </>
              )}
            </svg>

            {/* Central Orb */}
            <div
              className="absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2"
              style={{ opacity: loadingComplete ? 1 : 0, transition: 'opacity 1.5s ease-in-out' }}
            >
              <div className="absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 rounded-full"
                style={{
                  width: centerSize + 16, height: centerSize + 16,
                  background: "radial-gradient(circle, transparent 65%, rgba(72, 136, 255, 0.5) 85%, rgba(72, 136, 255, 0.8) 95%, rgba(72, 136, 255,0.4) 100%)",
                  filter: "blur(4px)",
                }} />
              <div className="relative flex items-center justify-center rounded-full"
                style={{
                  width: centerSize, height: centerSize,
                  background: "#030a18",
                  border: "2px solid rgba(72, 136, 255, 0.8)",
                  boxShadow: "0 0 15px rgba(72, 136, 255, 0.6), 0 0 30px rgba(72, 136, 255, 0.3)",
                }}>
                <svg className="absolute overflow-visible" width={centerSize + 20} height={centerSize + 20} viewBox="-10 -10 180 180" style={{ left: -10, top: -10 }}>
                  {[...Array(90)].map((_, i) => {
                    const seed = i * 7919 + 1234
                    const x = 15 + ((seed * 13 + i * 23) % 130)
                    const y = 15 + ((seed * 17 + i * 31) % 130)
                    const distFromCenter = Math.sqrt(Math.pow(x - 80, 2) + Math.pow(y - 80, 2))
                    const size = distFromCenter > 60 ? 0.8 + (i % 3) * 0.3 : 0.5 + (i % 3) * 0.2
                    return <circle key={`node-${i}`} cx={x} cy={y} r={size} fill={distFromCenter > 65 ? "rgba(72, 136, 255, 0.8)" : "rgba(72, 136, 255, 0.5)"} />
                  })}
                  {[...Array(60)].map((_, i) => {
                    const angle = (i * 6 + ((i * 7919) % 4)) * (Math.PI / 180)
                    const radius = 78 + ((i * 7919) % 12)
                    const x = 80 + Math.cos(angle) * radius
                    const y = 80 + Math.sin(angle) * radius
                    return <circle key={`edge-${i}`} cx={x} cy={y} r={0.6 + (i % 2) * 0.3} fill="rgba(72, 136, 255, 0.7)" />
                  })}
                  {[...Array(100)].map((_, i) => {
                    const seed1 = i * 7919; const seed2 = ((i + 3) % 100) * 7919
                    const x1 = 15 + ((seed1 * 13 + i * 17) % 130); const y1 = 15 + ((seed1 * 17 + i * 23) % 130)
                    const x2 = 15 + ((seed2 * 13 + i * 29) % 130); const y2 = 15 + ((seed2 * 17 + i * 31) % 130)
                    return <line key={`mesh-${i}`} x1={x1} y1={y1} x2={x2} y2={y2} stroke="rgba(72, 136, 255,0.3)" strokeWidth="0.4" />
                  })}
                  {[...Array(50)].map((_, i) => {
                    const s1 = (i * 2) * 7919; const s2 = ((i * 2 + 5) % 90) * 7919; const s3 = ((i * 2 + 11) % 90) * 7919
                    const x1 = 20 + ((s1 * 11) % 120); const y1 = 20 + ((s1 * 19) % 120)
                    const x2 = 20 + ((s2 * 11) % 120); const y2 = 20 + ((s2 * 19) % 120)
                    const x3 = 20 + ((s3 * 11) % 120); const y3 = 20 + ((s3 * 19) % 120)
                    return <g key={`tri-${i}`}><line x1={x1} y1={y1} x2={x2} y2={y2} stroke="rgba(72, 136, 255,0.25)" strokeWidth="0.35" /><line x1={x2} y1={y2} x2={x3} y2={y3} stroke="rgba(72, 136, 255,0.25)" strokeWidth="0.35" /><line x1={x3} y1={y3} x2={x1} y2={y1} stroke="rgba(72, 136, 255,0.25)" strokeWidth="0.35" /></g>
                  })}
                  {[...Array(6)].map((_, i) => {
                    const s1 = (i * 7) * 7919; const s2 = (i * 7 + 4) * 7919
                    const x1 = 25 + ((s1 * 11) % 110); const y1 = 25 + ((s1 * 19) % 110)
                    const x2 = 25 + ((s2 * 11) % 110); const y2 = 25 + ((s2 * 19) % 110)
                    return <line key={`purple-${i}`} x1={x1} y1={y1} x2={x2} y2={y2} stroke="rgba(180, 100, 220, 0.45)" strokeWidth="0.5" />
                  })}
                </svg>
                <span className="relative z-10 text-xl font-bold tracking-[0.15em] text-white"
                  style={{ textShadow: "0 0 10px rgba(255, 255, 255, 0.8), 0 0 20px rgba(72, 136, 255,0.6)" }}>
                  KNIRVANA
                </span>
              </div>
              <div className="absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 rounded-full"
                style={{
                  width: centerSize + 24, height: centerSize + 24,
                  border: "2px solid rgba(72, 136, 255, 0.6)",
                  boxShadow: "0 0 30px rgba(72, 136, 255, 0.6), 0 0 60px rgba(72, 136, 255,0.4), inset 0 0 20px rgba(72, 136, 255, 0.3)",
                }} />
            </div>

            {/* Inner Labels */}
            {innerLabels.map((item, i) => {
              const pos = polarToCartesian(item.angle, ring1Radius + 35)
              return (
                <motion.div key={`inner-${i}`} className="absolute left-1/2 top-1/2 whitespace-nowrap text-[9px] font-medium tracking-wider"
                  style={{ transform: `translate(calc(-50% + ${pos.x}px), calc(-50% + ${pos.y}px))`, color: "rgba(72, 136, 255, 0.9)", textShadow: "0 0 8px rgba(72, 136, 255,0.6)" }}
                  initial={{ opacity: 0 }} animate={loadingComplete ? { opacity: 1 } : {}} transition={{ duration: 0.5, delay: 2 + i * 0.05 }}>
                  {item.label}
                </motion.div>
              )
            })}

            {/* Middle Labels */}
            {middleLabels.map((item, i) => {
              const pos = polarToCartesian(item.angle, ring2Radius + 40)
              return (
                <motion.div key={`middle-${i}`} className="absolute left-1/2 top-1/2 whitespace-nowrap text-[10px] font-semibold tracking-wider"
                  style={{ transform: `translate(calc(-50% + ${pos.x}px), calc(-50% + ${pos.y}px))`, color: "rgba(0, 230, 255, 1)", textShadow: "0 0 10px rgba(72, 136, 255,0.8)" }}
                  initial={{ opacity: 0 }} animate={loadingComplete ? { opacity: 1 } : {}} transition={{ duration: 0.5, delay: 2.3 + i * 0.05 }}>
                  {item.label}
                </motion.div>
              )
            })}

            {/* Outer Icons */}
            {outerIcons.map((item, i) => {
              const pos = polarToCartesian(item.angle, iconRadius)
              const IconComponent = item.icon
              const isHovered = hoveredIcon === i
              const absoluteX = 450 + pos.x - 28
              const absoluteY = 450 + pos.y - 28
              return (
                <motion.div key={`icon-${i}`} className="absolute cursor-pointer" style={{ left: absoluteX, top: absoluteY }}
                  initial={{ opacity: 0, scale: 0 }} animate={loadingComplete ? { opacity: 1, scale: 1 } : {}}
                  transition={{ duration: 0.5, delay: 2.6 + i * 0.1, type: "spring" }}
                  onMouseEnter={() => setHoveredIcon(i)} onMouseLeave={() => setHoveredIcon(null)}
                  onClick={() => handleIconClick(item.section)}>
                  <motion.div className="flex items-center justify-center rounded-full"
                    style={{
                      width: 56, height: 56,
                      background: isHovered ? "rgba(20, 60, 180, 0.6)" : "rgba(4, 20, 80, 0.4)",
                      backdropFilter: "blur(8px)",
                      border: `2px solid ${isHovered ? "rgba(72, 136, 255, 0.8)" : "rgba(72, 136, 255, 0.4)"}`,
                      boxShadow: isHovered
                        ? "0 0 30px rgba(72, 136, 255, 0.7), 0 0 60px rgba(72, 136, 255, 0.4), inset 0 0 20px rgba(72, 136, 255, 0.3)"
                        : "0 0 15px rgba(72, 136, 255, 0.3), inset 0 0 10px rgba(72, 136, 255, 0.1)",
                    }}
                    animate={{ scale: isHovered ? 1.15 : 1 }} transition={{ duration: 0.2 }}>
                    <IconComponent className="h-6 w-6" style={{
                      color: isHovered ? "#4888ff" : "rgba(72, 136, 255, 0.8)",
                      filter: isHovered ? "drop-shadow(0 0 10px rgba(100, 160, 255, 1))" : "drop-shadow(0 0 4px rgba(72, 136, 255, 0.5))",
                    }} />
                  </motion.div>
                  <div className="absolute whitespace-nowrap text-center pointer-events-none"
                    style={{
                      top: 62, left: "50%", transform: "translateX(-50%)",
                      fontSize: 9, letterSpacing: "0.15em",
                      color: isHovered ? "rgba(72, 136, 255, 1)" : "rgba(72, 136, 255, 0.45)",
                      textShadow: isHovered ? "0 0 8px rgba(72, 136, 255, 0.9)" : "none",
                      transition: "color 0.2s, text-shadow 0.2s",
                    }}>
                    {item.label}
                  </div>
                </motion.div>
              )
            })}

            {/* KNIRV.COM label */}
            <motion.div
              className="absolute bottom-4 left-1/2 -translate-x-1/2 text-2xl font-bold tracking-[0.4em]"
              style={{ color: "rgba(255, 255, 255, 0.95)", textShadow: "0 0 20px rgba(72, 136, 255,0.6), 0 0 40px rgba(72, 136, 255,0.3)" }}
              initial={{ opacity: 0, y: 20 }} animate={loadingComplete ? { opacity: 1, y: 0 } : {}} transition={{ duration: 0.8, delay: 3 }}>
              KNIRV.COM
            </motion.div>
          </div>
        </div>
      </div>

      <SettingsModal isOpen={settingsOpen} onClose={() => setSettingsOpen(false)} />
    </>
  )
}
