import { useRef, useMemo } from "react";
import { useFrame } from "@react-three/fiber";
import * as THREE from "three";
import ErrorNode from "./ErrorNode";
import AIAgent from "./AIAgent";
import { useKnirvana } from "../../lib/stores/useKnirvana";

export default function KnirvGraph() {
  const graphRef = useRef<THREE.Group>(null);
  const { errorNodes, agents, skillNodes } = useKnirvana();

  // Generate interconnecting lines between nodes
  const connectionLines = useMemo(() => {
    const lines: JSX.Element[] = [];
    const positions: THREE.Vector3[] = [];
    
    // Create connections between error nodes and skill nodes
    errorNodes.forEach((errorNode, i) => {
      skillNodes.forEach((skillNode, j) => {
        if (Math.random() > 0.7) { // Random connections for demo
          const start = new THREE.Vector3(errorNode.position.x, 0.1, errorNode.position.z);
          const end = new THREE.Vector3(skillNode.position.x, 0.1, skillNode.position.z);
          
          positions.push(start, end);
          
          lines.push(
            <line key={`connection-${i}-${j}`}>
              <bufferGeometry>
                <bufferAttribute
                  attach="attributes-position"
                  count={2}
                  array={new Float32Array([
                    start.x, start.y, start.z,
                    end.x, end.y, end.z
                  ])}
                  itemSize={3}
                />
              </bufferGeometry>
              <lineBasicMaterial 
                color="#00aaff" 
                transparent={true}
                opacity={0.3}
                linewidth={2}
              />
            </line>
          );
        }
      });
    });
    
    return lines;
  }, [errorNodes, skillNodes]);

  useFrame((state) => {
    if (graphRef.current) {
      // Subtle floating animation
      graphRef.current.position.y = Math.sin(state.clock.elapsedTime * 0.5) * 0.1;
    }
  });

  return (
    <group ref={graphRef}>
      {/* Render connection lines */}
      {connectionLines}
      
      {/* Render ErrorNodes */}
      {errorNodes.map((node) => (
        <ErrorNode
          key={node.id}
          id={node.id}
          position={node.position}
          type={node.type}
          difficulty={node.difficulty}
          bounty={node.bounty}
          isBeingSolved={node.isBeingSolved}
          progress={node.progress}
        />
      ))}
      
      {/* Render SkillNodes */}
      {skillNodes.map((node) => (
        <mesh
          key={node.id}
          position={[node.position.x, 0.5, node.position.z]}
        >
          <octahedronGeometry args={[0.3]} />
          <meshStandardMaterial 
            color="#00ff88"
            emissive="#002211"
            transparent={true}
            opacity={0.8}
          />
        </mesh>
      ))}
      
      {/* Render AI Agents */}
      {agents.map((agent) => (
        <AIAgent
          key={agent.id}
          id={agent.id}
          position={agent.position}
          target={agent.target}
          status={agent.status}
          type={agent.type}
          efficiency={agent.efficiency}
          experience={agent.experience || 0}
        />
      ))}
    </group>
  );
}
