import { useRef, useEffect } from "react";
import { useFrame } from "@react-three/fiber";
import * as THREE from "three";
import { useKnirvana } from "./stores/useKnirvana";
import ErrorNode from "./ErrorNode";
import SkillNode from "./SkillNode";
import AIAgent from "./AIAgent";

export default function KnirvGraph() {
  const graphRef = useRef<THREE.Group>(null);
  const { errorNodes, skillNodes, agents, selectedErrorNode, selectedAgent } = useKnirvana();
  const isStraighteningAnchors = useKnirvana(s => s.isStraighteningAnchors);

  const deployedAgents = agents.filter(agent => !agent.staged);
  const stagedAgents = agents.filter(agent => agent.staged);

  return (
    <group ref={graphRef}>
      {/* Render error nodes */}
      {errorNodes.map((node) => (
        <ErrorNode
          key={node.id}
          node={node}
          isSelected={selectedErrorNode === node.id}
          onSelect={() => useKnirvana.getState().selectErrorNode(node.id)}
        />
      ))}
      
      {/* Render skill nodes */}
      {skillNodes.map((node) => (
        <SkillNode
          key={node.id}
          node={node}
        />
      ))}
      
      {/* Staged agents at the arena edge — hidden during straightening (proxies take over) */}
      {!isStraighteningAnchors && stagedAgents.map((agent, index) => (
        <AIAgent
          key={agent.id}
          agent={agent}
          isSelected={selectedAgent === agent.id}
          onSelect={() => useKnirvana.getState().selectAgent(agent.id)}
          isStaged={true}
          stagingIndex={index}
        />
      ))}

      {/* Render AI agents (only deployed) */}
      {deployedAgents.map((agent) => (
        <AIAgent
          key={agent.id}
          agent={agent}
          isSelected={selectedAgent === agent.id}
          onSelect={() => useKnirvana.getState().selectAgent(agent.id)}
          onStage={() => console.log(`Staging agent ${agent.id}`)}
        />
      ))}
      
      {/* Connection lines between nodes and agents */}
      {deployedAgents.filter(agent => agent.target).map((agent) => {
        const targetNode = errorNodes.find(node => node.id === agent.target);
        if (!targetNode) return null;
        
        return (
          <line key={`connection-${agent.id}-${agent.target}`}>
            <bufferGeometry>
              <bufferAttribute
                attach="attributes-position"
                count={2}
                array={new Float32Array([
                  agent.position.x, agent.position.y, agent.position.z,
                  targetNode.position.x, targetNode.position.y, targetNode.position.z
                ])}
                itemSize={3}
              />
            </bufferGeometry>
            <lineBasicMaterial color="#00ffff" opacity={0.6} transparent />
          </line>
        );
      })}
    </group>
  );
}