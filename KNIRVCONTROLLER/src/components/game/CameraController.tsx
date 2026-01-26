import { useThree, useFrame } from "@react-three/fiber";
import { useRef, useEffect } from "react";
import * as THREE from "three";

export default function CameraController() {
  const { camera, gl } = useThree();
  const targetPosition = useRef(new THREE.Vector3(15, 20, 15));
  const targetLookAt = useRef(new THREE.Vector3(0, 0, 0));

  useFrame((state, delta) => {
    // Smooth camera movement
    camera.position.lerp(targetPosition.current, delta * 2);
    
    // Camera always looks at center
    const currentTarget = new THREE.Vector3(0, 0, 0);
    camera.lookAt(currentTarget);
    
    // Very subtle camera float effect
    const time = state.clock.elapsedTime;
    camera.position.y += Math.sin(time * 0.5) * 0.1;
  });

  return null;
}