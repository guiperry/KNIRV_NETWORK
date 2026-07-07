import { useRef, useMemo } from "react";
import { useFrame, ThreeEvent } from "@react-three/fiber";
import * as THREE from "three";
import { useKnirvana, getCommittedRingNodes } from "./stores/useKnirvana";
import {
  GRID_SINK_DEPTH,
  GRID_SINK_RADIUS,
  setDepthCenters,
  type DepthCenter,
} from "./gridDepthField";

interface ArenaGridProps {
  floorColor: string;
  gridColor: string;
  floorOpacity: number;
  gridOpacity: number;
  onFloorClick: (event: ThreeEvent<MouseEvent>) => void;
}

// The grid floor sinks parabolically beneath every error node whose
// validation ring has been committed (sculpted). Plane geometry is rotated
// -PI/2 about X, so: world.x = local.x, world.y = local.z, world.z = -local.y.

const EPSILON = 0.002;

function applyDipToGeometry(geometry: THREE.PlaneGeometry, centers: DepthCenter[]) {
  const pos = geometry.attributes.position as THREE.BufferAttribute;
  const r2max = GRID_SINK_RADIUS * GRID_SINK_RADIUS;
  for (let i = 0; i < pos.count; i++) {
    const wx = pos.getX(i);
    const wz = -pos.getY(i);
    let dip = 0;
    for (const c of centers) {
      const dx = wx - c.x;
      const dz = wz - c.z;
      const r2 = (dx * dx + dz * dz) / r2max;
      if (r2 < 1) dip += c.depth * (1 - r2);
    }
    pos.setZ(i, -dip);
  }
  pos.needsUpdate = true;
}

export default function ArenaGrid({ floorColor, gridColor, floorOpacity, gridOpacity, onFloorClick }: ArenaGridProps) {
  const floorGeomRef = useRef<THREE.PlaneGeometry>(null);
  const linesGeomRef = useRef<THREE.PlaneGeometry>(null);
  // id → animated depth; survives across renders for smooth transitions
  const depthsRef = useRef<Map<string, { x: number; z: number; depth: number }>>(new Map());

  const errorNodes = useKnirvana(s => s.errorNodes);
  const rewardAnchors = useKnirvana(s => s.rewardAnchors);

  const sunkTargets = useMemo(
    () => getCommittedRingNodes(errorNodes, rewardAnchors)
      .map(n => ({ id: n.id, x: n.position.x, z: n.position.z })),
    [errorNodes, rewardAnchors]
  );

  useFrame((_, delta) => {
    const depths = depthsRef.current;
    const targetIds = new Set(sunkTargets.map(t => t.id));

    // Register new bowls
    for (const t of sunkTargets) {
      if (!depths.has(t.id)) depths.set(t.id, { x: t.x, z: t.z, depth: 0 });
    }

    // Animate every bowl toward its target depth (or back up to 0)
    let animating = false;
    const lerp = Math.min(delta * 1.8, 1);
    for (const [id, c] of depths) {
      const target = targetIds.has(id) ? GRID_SINK_DEPTH : 0;
      const next = c.depth + (target - c.depth) * lerp;
      if (Math.abs(next - c.depth) > EPSILON) animating = true;
      c.depth = next;
      if (target === 0 && next < EPSILON) {
        depths.delete(id);
        animating = true;
      }
    }

    // Publish the field so grid particles can ride the deformation
    const centers: DepthCenter[] = [];
    for (const [id, c] of depths) {
      if (c.depth > EPSILON) centers.push({ id, x: c.x, z: c.z, depth: c.depth });
    }
    setDepthCenters(centers);

    if (animating) {
      if (floorGeomRef.current) applyDipToGeometry(floorGeomRef.current, centers);
      if (linesGeomRef.current) applyDipToGeometry(linesGeomRef.current, centers);
    }
  });

  return (
    <>
      {/* TRON grid floor — deforms beneath committed validation rings */}
      <mesh rotation={[-Math.PI / 2, 0, 0]} position={[0, -0.1, 0]} receiveShadow onClick={onFloorClick}>
        <planeGeometry ref={floorGeomRef} args={[100, 100, 50, 50]} />
        <meshStandardMaterial
          color={floorColor}
          wireframe={true}
          transparent={true}
          opacity={floorOpacity}
        />
      </mesh>

      {/* Solid floor for raycasting clicks (stays flat) */}
      <mesh rotation={[-Math.PI / 2, 0, 0]} position={[0, -0.11, 0]} onClick={onFloorClick} visible={false}>
        <planeGeometry args={[100, 100]} />
        <meshBasicMaterial transparent opacity={0} />
      </mesh>

      {/* Grid lines — same deformation */}
      <mesh rotation={[-Math.PI / 2, 0, 0]} position={[0, 0, 0]}>
        <planeGeometry ref={linesGeomRef} args={[100, 100, 25, 25]} />
        <meshBasicMaterial
          color={gridColor}
          transparent
          opacity={gridOpacity}
          wireframe
        />
      </mesh>
    </>
  );
}
