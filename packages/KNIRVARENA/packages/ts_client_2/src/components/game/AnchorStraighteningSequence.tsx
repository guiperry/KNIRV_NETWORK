/* eslint-disable react/no-unknown-property */

import { useRef, useEffect, useMemo, useState } from "react";
import { useFrame } from "@react-three/fiber";
import { useGLTF, useAnimations } from "@react-three/drei";
import * as THREE from "three";
import { useKnirvana, getStagingPosition } from "./stores/useKnirvana";

// ── Proxy agent ───────────────────────────────────────────────────────────

interface ProxyAgentProps {
  startPos: THREE.Vector3;
  targetPos: THREE.Vector3;
  index: number;
  timeRef: React.MutableRefObject<number>;
}

function ProxyAgent({ startPos, targetPos, index, timeRef }: ProxyAgentProps) {
  const groupRef = useRef<THREE.Group>(null);
  const { scene, animations } = useGLTF('/assets/avatar/Green_Bot_Explorer.glb');
  const { actions, names } = useAnimations(animations, groupRef);
  const clonedScene = useMemo(() => scene.clone(), [scene]);

  const MOVE_TO_DUR  = 2.5;
  const WORK_DUR     = 3.0;
  const MOVE_BACK_DUR = 2.5;

  // Per-agent stagger so they don't all start simultaneously
  const staggerOffset = index * 0.3;

  // Start with the walk/run animation
  useEffect(() => {
    if (names.length === 0) return;
    const walkAnim = names.find(n => /walk|run/i.test(n)) ?? names[0];
    actions[walkAnim]?.reset().play();
    return () => names.forEach(n => actions[n]?.stop());
  }, [actions, names]);

  useFrame((state, delta) => {
    if (!groupRef.current) return;

    const t = Math.max(0, timeRef.current - staggerOffset);
    const clock = state.clock.elapsedTime;

    if (t < MOVE_TO_DUR) {
      // ── Running toward anchor ─────────────────────────────────────────
      const eased = 1 - Math.pow(1 - Math.min(t / MOVE_TO_DUR, 1), 2);
      const pos = startPos.clone().lerp(targetPos, eased);
      groupRef.current.position.set(pos.x, 1, pos.z);

      const dir = targetPos.clone().sub(startPos);
      if (dir.length() > 0.01) {
        const yaw = Math.atan2(dir.x, dir.z);
        groupRef.current.rotation.y = THREE.MathUtils.lerp(groupRef.current.rotation.y, yaw, 0.2);
      }

      const walkAnim = names.find(n => /walk|run/i.test(n)) ?? names[0];
      if (walkAnim && !actions[walkAnim]?.isRunning()) {
        names.forEach(n => actions[n]?.stop());
        actions[walkAnim]?.reset().play();
      }

    } else if (t < MOVE_TO_DUR + WORK_DUR) {
      // ── Working at anchor ─────────────────────────────────────────────
      const bounce = Math.abs(Math.sin(clock * 8)) * 0.12;
      groupRef.current.position.set(targetPos.x, 1 + bounce, targetPos.z);
      groupRef.current.rotation.y += delta * 3;

      const workAnim = names.find(n => /work|dance|idle|stand/i.test(n));
      const fallback = names.find(n => /walk|run/i.test(n)) ?? names[0];
      const chosen = workAnim ?? fallback;
      if (chosen && !actions[chosen]?.isRunning()) {
        names.forEach(n => actions[n]?.stop());
        actions[chosen]?.reset().play();
      }

    } else {
      // ── Running back to staging edge ──────────────────────────────────
      const progress = Math.min((t - MOVE_TO_DUR - WORK_DUR) / MOVE_BACK_DUR, 1);
      const eased = 1 - Math.pow(1 - progress, 2);
      const pos = targetPos.clone().lerp(startPos, eased);
      groupRef.current.position.set(pos.x, 1, pos.z);

      const dir = startPos.clone().sub(targetPos);
      if (dir.length() > 0.01) {
        const yaw = Math.atan2(dir.x, dir.z);
        groupRef.current.rotation.y = THREE.MathUtils.lerp(groupRef.current.rotation.y, yaw, 0.2);
      }

      const walkAnim = names.find(n => /walk|run/i.test(n)) ?? names[0];
      if (walkAnim && !actions[walkAnim]?.isRunning()) {
        names.forEach(n => actions[n]?.stop());
        actions[walkAnim]?.reset().play();
      }
    }
  });

  return (
    <group ref={groupRef} position={[startPos.x, 1, startPos.z]}>
      {scene.children.length > 0 ? (
        // Lift primitive by 5 to keep feet on the grid at scale 3×
        // (same offset as staged AIAgent components)
        <primitive object={clonedScene} scale={3.0} position={[0, 5.0, 0]} castShadow />
      ) : (
        <mesh>
          <capsuleGeometry args={[0.4, 1.2, 4, 8]} />
          <meshStandardMaterial color="#00ff88" emissive="#003311" emissiveIntensity={0.8} />
        </mesh>
      )}
      <pointLight color="#00ff88" intensity={1.5} distance={5} decay={2} />
    </group>
  );
}

// ── Sequence controller ───────────────────────────────────────────────────

export default function AnchorStraighteningSequence() {
  const isStraighteningAnchors = useKnirvana(s => s.isStraighteningAnchors);
  const rewardAnchors           = useKnirvana(s => s.rewardAnchors);
  const agents                  = useKnirvana(s => s.agents);
  const completeStraightening   = useKnirvana(s => s.completeStraightening);

  const timeRef     = useRef(0);
  const doneRef     = useRef(false);

  // ── KEY FIX: use state, not ref, so the render updates when proxies load ──
  const [proxies, setProxies] = useState<{ start: THREE.Vector3; target: THREE.Vector3 }[]>([]);

  const TOTAL_DURATION = 8.5; // stagger + move_to + work + move_back

  useEffect(() => {
    if (!isStraighteningAnchors) {
      setProxies([]);
      return;
    }

    timeRef.current = 0;
    doneRef.current = false;

    // Any horizontal anchor drives the animation — no need for isCommitted
    const horizontalAnchors = rewardAnchors.filter(a => a.isHorizontal);
    const stagedAgents       = agents.filter(a => a.staged);

    const built = horizontalAnchors.map((anchor, i) => {
      const stagingPos =
        stagedAgents[i % Math.max(stagedAgents.length, 1)]?.position ??
        getStagingPosition(i % 3);
      return {
        start:  new THREE.Vector3(stagingPos.x, 1, stagingPos.z),
        target: new THREE.Vector3(anchor.position.x, 1, anchor.position.z),
      };
    });

    setProxies(built);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isStraighteningAnchors]);

  useFrame((_, delta) => {
    if (!isStraighteningAnchors || doneRef.current) return;
    timeRef.current += delta;
    if (timeRef.current >= TOTAL_DURATION) {
      doneRef.current = true;
      completeStraightening();
    }
  });

  if (!isStraighteningAnchors || proxies.length === 0) return null;

  return (
    <group>
      {proxies.map((proxy, i) => (
        <ProxyAgent
          key={i}
          startPos={proxy.start}
          targetPos={proxy.target}
          index={i}
          timeRef={timeRef}
        />
      ))}
    </group>
  );
}
