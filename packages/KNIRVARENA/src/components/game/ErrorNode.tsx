import { useRef, useState, useMemo } from "react";
import { useFrame } from "@react-three/fiber";
import { ThreeEvent } from "@react-three/fiber";
import * as THREE from "three";
import { ErrorNode as ErrorNodeType, useKnirvana } from "./stores/useKnirvana";

interface ErrorNodeProps {
  node: ErrorNodeType;
  isSelected: boolean;
  onSelect: () => void;
}

export default function ErrorNode({ node, isSelected, onSelect }: ErrorNodeProps) {
  const groupRef = useRef<THREE.Group>(null);
  const meshRef = useRef<THREE.Mesh>(null);
  const ringRef = useRef<THREE.Mesh>(null);
  const spikeMeshRefs = useRef<(THREE.Mesh | null)[]>(Array(8).fill(null));
  const hoveredSpikeRef = useRef<number | null>(null);
  const elevationRef = useRef(0);
  const [hoveredSpike, setHoveredSpike] = useState<number | null>(null);

  const isAnalyzing = useKnirvana(state => state.isAnalyzing);
  const isSculpting = useKnirvana(state => state.isSculpting);
  const isNoiseInjecting = useKnirvana(state => state.isNoiseInjecting);
  const addRewardAnchor = useKnirvana(state => state.addRewardAnchor);
  const openDVEWorkspace = useKnirvana(state => state.openDVEWorkspace);
  const setSculpting = useKnirvana(state => state.setSculpting);
  const selectedErrorNode = useKnirvana(state => state.selectedErrorNode);
  const errorNodes = useKnirvana(state => state.errorNodes);
  const rewardAnchors = useKnirvana(state => state.rewardAnchors);

  // True once all 8 spike positions around this node have a committed anchor
  const hasCommittedRing = useMemo(() => {
    const RADIUS = 1.5;
    return Array.from({ length: 8 }).every((_, i) => {
      const angle = (i / 8) * Math.PI * 2;
      const sx = node.position.x + Math.cos(angle) * RADIUS;
      const sz = node.position.z + Math.sin(angle) * RADIUS;
      return rewardAnchors.some(
        a => a.isCommitted && Math.abs(a.position.x - sx) < 0.05 && Math.abs(a.position.z - sz) < 0.05
      );
    });
  }, [rewardAnchors, node.position]);

  // Keep a ref so useFrame always sees the latest value without stale closure
  const hasCommittedRingRef = useRef(false);
  hasCommittedRingRef.current = hasCommittedRing;

  useFrame((state, delta) => {
    const time = state.clock.elapsedTime;

    // Elevate the whole node group slightly when its ring has been committed
    if (groupRef.current) {
      const targetElevation = hasCommittedRingRef.current ? 0.3 : 0;
      elevationRef.current += (targetElevation - elevationRef.current) * Math.min(delta * 2.5, 1);
      groupRef.current.position.set(
        node.position.x,
        node.position.y + 1 + elevationRef.current,
        node.position.z
      );
    }

    if (meshRef.current) {
      const baseScale = isAnalyzing ? 1.3 : 1;
      const scale = isSelected ? baseScale + 0.2 + Math.sin(time * 4) * 0.1 : baseScale;
      meshRef.current.scale.setScalar(scale);
      meshRef.current.rotation.y += 0.01;
    }

    if (ringRef.current && isAnalyzing) {
      ringRef.current.rotation.z = time * 2;
      ringRef.current.scale.setScalar(1 + Math.sin(time * 3) * 0.15);
    }

    // Pulse hovered spike in sculpt mode
    if (isSculpting && isAnalyzing) {
      spikeMeshRefs.current.forEach((mesh, i) => {
        if (!mesh) return;
        if (hoveredSpikeRef.current === i) {
          const pulse = 1 + Math.sin(time * 10) * 0.25;
          mesh.scale.setScalar(pulse);
        } else {
          mesh.scale.setScalar(1);
        }
      });
    }
  });

  const handleSpikeEnter = (index: number) => {
    hoveredSpikeRef.current = index;
    setHoveredSpike(index);
    document.body.style.cursor = "crosshair";
  };

  const handleSpikeLeave = () => {
    hoveredSpikeRef.current = null;
    setHoveredSpike(null);
    document.body.style.cursor = "auto";
  };

  const handleSpikeClick = (e: ThreeEvent<MouseEvent>, index: number) => {
    if (!isSculpting) return;
    e.stopPropagation();

    const angle = (index / 8) * Math.PI * 2;
    const radius = 1.5;
    const anchorX = node.position.x + Math.cos(angle) * radius;
    const anchorZ = node.position.z + Math.sin(angle) * radius;

    // One validation node per spike — ignore repeat clicks
    const occupied = rewardAnchors.some(
      a => Math.abs(a.position.x - anchorX) < 0.05 && Math.abs(a.position.z - anchorZ) < 0.05
    );
    if (occupied) return;

    const errorNodeData = selectedErrorNode
      ? errorNodes.find(n => n.id === selectedErrorNode)
      : null;
    const metadata = errorNodeData
      ? {
          logs: ["Error: Memory allocation failed"],
          traces: ["Component: DataProcessor"],
          severity: "high",
          description: "Memory leak detected",
        }
      : undefined;

    addRewardAnchor({
      id: `anchor-${Date.now()}`,
      position: { x: anchorX, y: 0.5, z: anchorZ },
      weights: { w_c: 0.6, w_l: 0.3, w_s: 0.1 },
      constraints: "// Define constraints here",
      linkedErrorNode: node.id, // always link to the node whose spike was clicked
      metadata,
    });

    document.body.style.cursor = "auto";
  };

  const handleNoiseArrowClick = (e: ThreeEvent<MouseEvent>, index: number) => {
    e.stopPropagation();
    const angle = (index / 8) * Math.PI * 2;
    const radius = 3.0;
    addRewardAnchor({
      id: `anchor-noise-${Date.now()}`,
      position: { x: node.position.x + Math.cos(angle) * radius, y: 0.5, z: node.position.z + Math.sin(angle) * radius },
      weights: { w_c: 0.6, w_l: 0.3, w_s: 0.1 },
      constraints: '// Noise injection constraint',
      linkedErrorNode: node.id,
      anchorType: 'noise',
    });
  };

  // Clicking the node selects it; outside anchor-placement modes it also
  // opens the node's own DVE workspace.
  const handleNodeClick = (e: ThreeEvent<MouseEvent>) => {
    e.stopPropagation();
    onSelect();
    if (!isAnalyzing && !isSculpting && !isNoiseInjecting) {
      openDVEWorkspace(node.id);
    }
  };

  const color = node.isBeingSolved ? "#ff6600" : "#ff0000";

  return (
    // Initial position set here; useFrame overrides position.y every frame for the elevation animation
    <group ref={groupRef} position={[node.position.x, node.position.y + 1, node.position.z]}>
      <mesh ref={meshRef} onClick={handleNodeClick} castShadow receiveShadow>
        <sphereGeometry args={[0.8, 16, 16]} />
        <meshStandardMaterial
          color={color}
          emissive={color}
          emissiveIntensity={isAnalyzing ? 0.8 : (isSelected ? 0.5 : 0.2)}
          roughness={0.3}
          metalness={0.7}
        />

        {node.isBeingSolved && (
          <mesh position={[0, 1.2, 0]}>
            <cylinderGeometry args={[0.3, 0.3, 2, 8]} />
            <meshBasicMaterial color="#00ff00" transparent opacity={0.6} />
          </mesh>
        )}

        <pointLight
          position={[0, 0.5, 0]}
          color={color}
          intensity={isAnalyzing ? 2 : (isSelected ? 1 : 0.5)}
          distance={isAnalyzing ? 8 : 5}
          decay={2}
        />
      </mesh>

      {/* Analyze mode - red pulsing ring */}
      {isAnalyzing && (
        <mesh ref={ringRef} rotation={[Math.PI / 2, 0, 0]}>
          <ringGeometry args={[1.5, 1.8, 32]} />
          <meshBasicMaterial
            color="#ff0000"
            transparent
            opacity={0.7}
            side={THREE.DoubleSide}
          />
        </mesh>
      )}

      {/* Analyze mode - outer glow ring */}
      {isAnalyzing && (
        <mesh rotation={[Math.PI / 2, 0, 0]} position={[0, -0.5, 0]}>
          <ringGeometry args={[2.0, 2.3, 32]} />
          <meshBasicMaterial
            color="#ff3333"
            transparent
            opacity={0.4}
            side={THREE.DoubleSide}
          />
        </mesh>
      )}

      {/* Analyze mode - white light spikes */}
      {isAnalyzing && Array.from({ length: 8 }).map((_, i) => {
        const angle = (i / 8) * Math.PI * 2;
        const radius = 1.5;
        const x = Math.cos(angle) * radius;
        const z = Math.sin(angle) * radius;
        const isHovered = hoveredSpike === i && isSculpting;
        return (
          <group key={i} position={[x, 0, z]}>
            <mesh
              ref={el => { spikeMeshRefs.current[i] = el; }}
              position={[0, 1.3, 0]}
              onPointerEnter={isSculpting ? () => handleSpikeEnter(i) : undefined}
              onPointerLeave={isSculpting ? handleSpikeLeave : undefined}
              onClick={isSculpting ? (e) => handleSpikeClick(e, i) : undefined}
            >
              <cylinderGeometry args={[0.015, 0.025, 2.6, 6]} />
              <meshBasicMaterial
                color={isHovered ? "#ffdd00" : "#ffffff"}
                transparent
                opacity={isHovered ? 1.0 : 0.85}
              />
            </mesh>
            <pointLight
              position={[0, 1.3, 0]}
              color={isHovered ? "#ffdd00" : "#ffffff"}
              intensity={isHovered ? 2.0 : 0.4}
              distance={isHovered ? 5 : 3}
              decay={2}
            />
          </group>
        );
      })}
      {/* Noise injection mode — green arrows around sculpted nodes */}
      {isNoiseInjecting && hasCommittedRing && Array.from({ length: 8 }).map((_, i) => {
        const angle = (i / 8) * Math.PI * 2;
        const radius = 3.0;
        const x = Math.cos(angle) * radius;
        const z = Math.sin(angle) * radius;
        // Arrow points outward from node center
        const arrowAngle = angle + Math.PI / 2;
        return (
          <group key={`ni-${i}`} position={[x, 0, z]} rotation={[0, -arrowAngle, 0]}>
            {/* Shaft */}
            <mesh
              position={[0, 1.2, 0]}
              onClick={(e) => handleNoiseArrowClick(e, i)}
              onPointerEnter={() => { document.body.style.cursor = 'crosshair'; }}
              onPointerLeave={() => { document.body.style.cursor = 'auto'; }}
            >
              <cylinderGeometry args={[0.04, 0.04, 1.4, 6]} />
              <meshBasicMaterial color="#00ff88" transparent opacity={0.85} />
            </mesh>
            {/* Arrowhead */}
            <mesh
              position={[0, 2.1, 0]}
              onClick={(e) => handleNoiseArrowClick(e, i)}
              onPointerEnter={() => { document.body.style.cursor = 'crosshair'; }}
              onPointerLeave={() => { document.body.style.cursor = 'auto'; }}
            >
              <coneGeometry args={[0.15, 0.4, 6]} />
              <meshBasicMaterial color="#00ff88" transparent opacity={0.9} />
            </mesh>
            <pointLight position={[0, 1.5, 0]} color="#00ff88" intensity={0.6} distance={3} decay={2} />
          </group>
        );
      })}
    </group>
  );
}
