import { useRef, useMemo } from "react";
import { useFrame, ThreeEvent } from "@react-three/fiber";
import * as THREE from "three";
import { RewardAnchor, useKnirvana } from "./stores/useKnirvana";

interface RewardAnchor3DProps {
  anchor: RewardAnchor;
}

const COMMIT_DURATION = 1.5;
const SINK_DEPTH = 2.5;

export default function RewardAnchor3D({ anchor }: RewardAnchor3DProps) {
  const meshRef = useRef<THREE.Mesh>(null);
  const ringRef = useRef<THREE.Mesh>(null);
  const orientGroupRef = useRef<THREE.Group>(null);
  const outerGroupRef = useRef<THREE.Group>(null);
  const sunkOffsetRef = useRef(0);
  const outerYRef = useRef(anchor.position.y);
  const straightenProgressRef = useRef(0);
  const commitProgressRef = useRef(0);

  const selectedRewardAnchor = useKnirvana(state => state.selectedRewardAnchor);
  const selectRewardAnchor = useKnirvana(state => state.selectRewardAnchor);
  const updateRewardAnchor = useKnirvana(state => state.updateRewardAnchor);
  const isStraighteningAnchors = useKnirvana(state => state.isStraighteningAnchors);
  const linkedNode = useKnirvana(state =>
    anchor.linkedErrorNode ? state.errorNodes.find(n => n.id === anchor.linkedErrorNode) : null
  );

  const isSelected = selectedRewardAnchor === anchor.id;
  const isHorizontal = anchor.isHorizontal !== false;
  const isNoise = anchor.anchorType === 'noise';

  const crystalColor = isNoise ? "#9900ff" : anchor.isCommitted ? "#00dd66" : anchor.isSet ? "#2266ff" : "#ffcc00";
  const emissiveColor = isNoise ? "#6600aa" : anchor.isCommitted ? "#00aa44" : anchor.isSet ? "#1144cc" : "#ffaa00";
  const ringColor    = isNoise ? "#bb44ff" : anchor.isCommitted ? "#00ff88" : anchor.isSet ? "#4488ff" : "#ffcc00";
  const glowColor    = isNoise ? "#bb44ff" : anchor.isCommitted ? "#00ff88" : anchor.isSet ? "#4488ff" : "#ffcc00";

  const horizontalQuat = useMemo(() => {
    if (!linkedNode) return new THREE.Quaternion();
    const dx = anchor.position.x - linkedNode.position.x;
    const dz = anchor.position.z - linkedNode.position.z;
    const len = Math.sqrt(dx * dx + dz * dz);
    if (len < 0.001) return new THREE.Quaternion();
    const outward = new THREE.Vector3(dx / len, 0, dz / len);
    return new THREE.Quaternion().setFromUnitVectors(new THREE.Vector3(0, 1, 0), outward);
  }, [linkedNode, anchor.position.x, anchor.position.z]);

  useFrame((state, delta) => {
    const time = state.clock.elapsedTime;

    // ── Commit animation: sink below grid + rotate to horizontal ─────────
    if (anchor.isCommitting && outerGroupRef.current && orientGroupRef.current) {
      commitProgressRef.current = Math.min(commitProgressRef.current + delta / COMMIT_DURATION, 1);
      const p = commitProgressRef.current;

      // Ease-in-out
      const eased = p < 0.5 ? 2 * p * p : 1 - Math.pow(-2 * p + 2, 2) / 2;

      // Sink the outer group below the grid
      outerGroupRef.current.position.y = anchor.position.y - eased * SINK_DEPTH;
      outerYRef.current = outerGroupRef.current.position.y;

      // Rotate from vertical → horizontal
      orientGroupRef.current.quaternion.slerpQuaternions(
        new THREE.Quaternion(), // identity = vertical
        horizontalQuat,
        eased
      );

      if (commitProgressRef.current >= 1) {
        updateRewardAnchor(anchor.id, { isCommitting: false, isHorizontal: true });
        commitProgressRef.current = 0;
      }
      return; // skip normal animation while committing
    }

    // Reset commit progress when not committing
    if (!anchor.isCommitting) commitProgressRef.current = 0;

    // ── Orientation group ─────────────────────────────────────────────────
    if (orientGroupRef.current) {
      if (isHorizontal) {
        if (isStraighteningAnchors && anchor.isCommitted) {
          straightenProgressRef.current = Math.min(straightenProgressRef.current + delta * 0.4, 1);
          orientGroupRef.current.quaternion.slerpQuaternions(
            horizontalQuat,
            new THREE.Quaternion(),
            straightenProgressRef.current
          );
        } else {
          straightenProgressRef.current = 0;
          orientGroupRef.current.quaternion.copy(horizontalQuat);
        }
      } else {
        orientGroupRef.current.quaternion.identity();
        straightenProgressRef.current = 0;
      }
    }

    // ── Outer group Y ─────────────────────────────────────────────────────
    // Committed anchors STAY sunk beneath the grid — they must never pop
    // back up, even after straightening flips them vertical again.
    if (outerGroupRef.current) {
      const targetY = anchor.isCommitted
        ? anchor.position.y - SINK_DEPTH
        : anchor.position.y;
      outerYRef.current += (targetY - outerYRef.current) * Math.min(delta * 3, 1);
      outerGroupRef.current.position.y = outerYRef.current;
    }

    // ── Float / sink (only when vertical) ────────────────────────────────
    if (meshRef.current) {
      if (!isHorizontal) {
        const targetSunk = anchor.isCommitted ? -0.45 : 0;
        sunkOffsetRef.current += (targetSunk - sunkOffsetRef.current) * Math.min(delta * 2, 1);
        const amplitude = anchor.isCommitted ? 0.07 : 0.15;
        meshRef.current.position.y = sunkOffsetRef.current + Math.sin(time * 2) * amplitude;
      } else {
        meshRef.current.position.y = 0;
        sunkOffsetRef.current = 0;
      }
      meshRef.current.rotation.y = time * 0.5;
    }

    // ── Ring spin ─────────────────────────────────────────────────────────
    if (ringRef.current) {
      ringRef.current.rotation.z = time * 2;
      if (isSelected) {
        ringRef.current.scale.setScalar(1 + Math.sin(time * 4) * 0.1);
      }
    }
  });

  const handleClick = (e: ThreeEvent<MouseEvent>) => {
    e.stopPropagation();
    selectRewardAnchor(isSelected ? null : anchor.id);
  };

  return (
    <group position={[anchor.position.x, anchor.position.y, anchor.position.z]}>
      {/* outerGroupRef handles Y during commit animation */}
      <group ref={outerGroupRef}>
        {/* Orientation group — rotates between vertical and horizontal */}
        <group ref={orientGroupRef}>
          <mesh ref={meshRef} onClick={handleClick} castShadow>
            <octahedronGeometry args={[0.4, 0]} />
            <meshStandardMaterial
              color={crystalColor}
              emissive={emissiveColor}
              emissiveIntensity={isSelected ? 0.9 : 0.4}
              roughness={0.2}
              metalness={0.9}
              transparent
              opacity={0.9}
            />
          </mesh>

          <mesh ref={ringRef} rotation={[Math.PI / 2, 0, 0]}>
            <ringGeometry args={[0.6, 0.75, 6]} />
            <meshBasicMaterial
              color={isSelected ? "#ffffff" : ringColor}
              transparent
              opacity={isSelected ? 0.9 : 0.4}
              side={THREE.DoubleSide}
            />
          </mesh>

          {isSelected && (
            <mesh rotation={[Math.PI / 2, 0, 0]} position={[0, -0.1, 0]}>
              <ringGeometry args={[0.9, 1.1, 6]} />
              <meshBasicMaterial
                color={ringColor}
                transparent
                opacity={0.3}
                side={THREE.DoubleSide}
              />
            </mesh>
          )}

          <mesh position={[0, 1.5, 0]}>
            <cylinderGeometry args={[0.02, 0.02, 3, 8]} />
            <meshBasicMaterial
              color={isSelected ? "#ffffff" : ringColor}
              transparent
              opacity={isSelected ? 0.7 : 0.3}
            />
          </mesh>

          <mesh position={[0, 3.1, 0]}>
            <coneGeometry args={[0.08, 0.25, 6]} />
            <meshBasicMaterial
              color={isSelected ? "#ffffff" : ringColor}
              transparent
              opacity={isSelected ? 0.9 : 0.5}
            />
          </mesh>
        </group>
      </group>

      <pointLight
        color={glowColor}
        intensity={isSelected ? 1.5 : 0.5}
        distance={5}
        decay={2}
      />
    </group>
  );
}
