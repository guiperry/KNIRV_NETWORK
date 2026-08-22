"use client"

import { useEffect, useState } from "react"
import { motion } from "framer-motion"

interface Particle {
  id: number
  x: number
  y: number
  vx: number
  vy: number
  size: number
  opacity: number
  color: string
  targetX?: number
  targetY?: number
}

export default function StarSupernova({ onComplete }: { onComplete: () => void }) {
  const [phase, setPhase] = useState<'pulsing' | 'supernova' | 'forming' | 'complete'>('pulsing')
  const [particles, setParticles] = useState<Particle[]>([])
  const [menuScale, setMenuScale] = useState(0)

  useEffect(() => {
    const supernovaTimer = setTimeout(() => {
      setPhase('supernova')

      const newParticles: Particle[] = []
      const particleCount = 200

      for (let i = 0; i < particleCount; i++) {
        const angle = (Math.PI * 2 * i) / particleCount + (Math.random() - 0.5) * 0.5
        const velocity = 1.5 + Math.random() * 3
        const size = Math.random() * 2.5 + 0.5
        const colorVariation = Math.random()

        let color = "rgba(72, 136, 255, "
        if (colorVariation > 0.7) {
          color = "rgba(100, 150, 255, "
        } else if (colorVariation > 0.4) {
          color = "rgba(72, 136, 255, "
        }

        const formationAngle = (Math.PI * 2 * i) / particleCount
        const formationRadius = 150 + Math.random() * 200
        const targetX = Math.cos(formationAngle) * formationRadius
        const targetY = Math.sin(formationAngle) * formationRadius

        newParticles.push({
          id: i,
          x: 0,
          y: 0,
          vx: Math.cos(angle) * velocity,
          vy: Math.sin(angle) * velocity,
          size,
          opacity: 1,
          color,
          targetX,
          targetY
        })
      }

      setParticles(newParticles)

      setTimeout(() => {
        setPhase('forming')
        setMenuScale(1)
        setTimeout(onComplete, 1500)
      }, 800)

    }, 2000)

    return () => clearTimeout(supernovaTimer)
  }, [onComplete])

  if (phase === 'complete') {
    return null
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center overflow-hidden">
      <div
        className="absolute inset-0"
        style={{ backgroundColor: "#030a18" }}
      >
        {[...Array(100)].map((_, i) => (
          <motion.div
            key={i}
            className="absolute rounded-full bg-white"
            style={{
              left: `${Math.random() * 100}%`,
              top: `${Math.random() * 100}%`,
              width: Math.random() > 0.8 ? 2 : 1,
              height: Math.random() > 0.8 ? 2 : 1,
              opacity: Math.random() * 0.4 + 0.1,
            }}
            animate={{ opacity: [0.1, 0.4, 0.1] }}
            transition={{
              duration: Math.random() * 3 + 2,
              repeat: Infinity,
              ease: "easeInOut",
              delay: Math.random() * 2,
            }}
          />
        ))}
      </div>

      <motion.div
        className="relative flex items-center justify-center"
        style={{ width: 100, height: 100 }}
        animate={
          phase === 'pulsing'
            ? { scale: [1, 1.2, 1] }
            : phase === 'supernova'
            ? { scale: [1, 3, 0], opacity: [1, 0.8, 0] }
            : {}
        }
        transition={
          phase === 'pulsing'
            ? { duration: 2, repeat: Infinity, ease: "easeInOut" }
            : { duration: 1, ease: "easeOut" }
        }
      >
        {/* Glow behind brain */}
        <div
          className="absolute inset-0 rounded-full"
          style={{
            background: `radial-gradient(circle at 50% 50%,
              rgba(72, 136, 255, 0.3) 0%,
              rgba(0, 180, 255, 0.15) 40%,
              transparent 70%
            )`,
            boxShadow: `
              0 0 60px rgba(72, 136, 255, 0.6),
              0 0 100px rgba(72, 136, 255, 0.4),
              0 0 150px rgba(0, 100, 200, 0.2)
            `,
          }}
        />
        {/* Brain image */}
        <motion.img
          src="/icons/brain.png"
          alt="KNIRV Brain"
          className="relative"
          style={{
            width: 60,
            height: 60,
            objectFit: "contain",
            filter: "drop-shadow(0 0 12px rgba(72, 136, 255, 0.8)) drop-shadow(0 0 30px rgba(0, 100, 200, 0.4))",
          }}
          animate={phase === 'pulsing' ? { scale: [1, 1.15, 1], opacity: [0.85, 1, 0.85] } : {}}
          transition={{ duration: 2, repeat: Infinity, ease: "easeInOut" }}
        />
        {/* Outer glow ring */}
        <motion.div
          className="absolute inset-0 rounded-full"
          style={{
            background: `radial-gradient(circle,
              transparent 40%,
              rgba(72, 136, 255, 0.06) 55%,
              rgba(72, 136, 255, 0.03) 75%,
              transparent 100%
            )`,
            filter: 'blur(8px)',
          }}
          animate={phase === 'pulsing' ? { scale: [1, 1.4, 1], opacity: [0.4, 0.6, 0.4] } : {}}
          transition={{ duration: 2.5, repeat: Infinity, ease: "easeInOut" }}
        />
      </motion.div>

      {phase === 'pulsing' && (
        <>
          <motion.div
            className="absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 rounded-full border-2"
            style={{
              width: 140,
              height: 140,
              borderColor: "rgba(72, 136, 255, 0.3)",
              boxShadow: "0 0 20px rgba(72, 136, 255, 0.2)",
            }}
            animate={{ scale: [1, 1.6, 1], opacity: [0.3, 0.1, 0.3] }}
            transition={{ duration: 2, repeat: Infinity, ease: "easeInOut" }}
          />
          <motion.div
            className="absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 rounded-full border"
            style={{
              width: 180,
              height: 180,
              borderColor: "rgba(72, 136, 255, 0.2)",
              boxShadow: "0 0 30px rgba(72, 136, 255, 0.1)",
            }}
            animate={{ scale: [1, 1.4, 1], opacity: [0.2, 0.05, 0.2] }}
            transition={{ duration: 2.5, repeat: Infinity, ease: "easeInOut" }}
          />
        </>
      )}

      {phase !== 'pulsing' && particles.map((particle) => (
        <motion.div
          key={particle.id}
          className="absolute rounded-full"
          style={{
            width: particle.size,
            height: particle.size,
            backgroundColor: particle.color + "1)",
            boxShadow: `0 0 ${particle.size * 3}px ${particle.color + "0.8)"}`,
          }}
          initial={{ x: 0, y: 0, opacity: 1 }}
          animate={
            phase === 'supernova'
              ? { x: particle.vx * 150, y: particle.vy * 150, opacity: 0.8, scale: [1, 1.2, 1] }
              : { x: particle.targetX || 0, y: particle.targetY || 0, opacity: 0.3, scale: [1, 0.8, 0.6] }
          }
          transition={
            phase === 'supernova'
              ? { duration: 0.8, ease: "easeOut" }
              : { duration: 1.5, ease: "easeInOut" }
          }
        />
      ))}

      {phase === 'forming' && (
        <motion.div
          className="absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2"
          style={{ width: 900, height: 900 }}
          animate={{ scale: menuScale, opacity: menuScale * 0.3 }}
          transition={{ duration: 1.5, ease: "easeOut" }}
        >
          {[120, 180, 250, 320].map((radius, i) => (
            <div
              key={i}
              className="absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 rounded-full border"
              style={{
                width: radius * 2,
                height: radius * 2,
                borderColor: "rgba(72, 136, 255, 0.2)",
                boxShadow: `0 0 ${20 - i * 3}px rgba(72, 136, 255, 0.1)`,
              }}
            />
          ))}
          <div
            className="absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 rounded-full"
            style={{
              width: 160,
              height: 160,
              background: "rgba(0, 100, 150, 0.1)",
              border: "2px solid rgba(72, 136, 255, 0.3)",
              boxShadow: "0 0 30px rgba(72, 136, 255, 0.2)",
            }}
          />
        </motion.div>
      )}
    </div>
  )
}
