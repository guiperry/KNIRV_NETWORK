import { useRef, useEffect } from 'react';
import { useFrame, useThree } from '@react-three/fiber';
import { useKeyboardControls } from '@react-three/drei';
import * as THREE from 'three';

interface CameraProps {
  target?: THREE.Vector3;
  followTarget?: boolean;
}

export default function Camera({ target, followTarget = false }: CameraProps) {
  const { camera } = useThree();
  const cameraTargetRef = useRef(new THREE.Vector3(0, 0, 0));
  const cameraPositionRef = useRef(new THREE.Vector3(15, 20, 15));
  const cameraOffset = useRef(new THREE.Vector3(15, 20, 15));
  
  const [, get] = useKeyboardControls();

  useEffect(() => {
    console.log('RTS Camera controller initialized');
    // Set initial camera position for RTS view
    camera.position.copy(cameraPositionRef.current);
    camera.lookAt(cameraTargetRef.current);
  }, [camera]);

  useFrame((state, delta) => {
    const controls = get();
    const moveSpeed = 15;
    const rotateSpeed = 1.5;
    const zoomSpeed = 8;

    // Follow target if specified
    if (followTarget && target) {
      cameraTargetRef.current.copy(target);
      cameraPositionRef.current.copy(target).add(cameraOffset.current);
    }

    // Camera movement - WASD for panning
    if (controls.forward) {
      console.log('Camera: Moving forward');
      const forward = new THREE.Vector3(0, 0, -1).applyQuaternion(camera.quaternion);
      forward.y = 0; // Keep movement on horizontal plane
      forward.normalize();
      
      cameraPositionRef.current.add(forward.multiplyScalar(moveSpeed * delta));
      cameraTargetRef.current.add(forward.multiplyScalar(moveSpeed * delta));
    }
    
    if (controls.backward) {
      console.log('Camera: Moving backward');
      const backward = new THREE.Vector3(0, 0, 1).applyQuaternion(camera.quaternion);
      backward.y = 0;
      backward.normalize();
      
      cameraPositionRef.current.add(backward.multiplyScalar(moveSpeed * delta));
      cameraTargetRef.current.add(backward.multiplyScalar(moveSpeed * delta));
    }
    
    if (controls.leftward) {
      console.log('Camera: Moving left');
      const left = new THREE.Vector3(-1, 0, 0).applyQuaternion(camera.quaternion);
      left.y = 0;
      left.normalize();
      
      cameraPositionRef.current.add(left.multiplyScalar(moveSpeed * delta));
      cameraTargetRef.current.add(left.multiplyScalar(moveSpeed * delta));
    }
    
    if (controls.rightward) {
      console.log('Camera: Moving right');
      const right = new THREE.Vector3(1, 0, 0).applyQuaternion(camera.quaternion);
      right.y = 0;
      right.normalize();
      
      cameraPositionRef.current.add(right.multiplyScalar(moveSpeed * delta));
      cameraTargetRef.current.add(right.multiplyScalar(moveSpeed * delta));
    }

    // Camera rotation - Q/E for orbital rotation around target
    if (controls.rotateLeft) {
      console.log('Camera: Rotating left');
      const angle = rotateSpeed * delta;
      const cos = Math.cos(angle);
      const sin = Math.sin(angle);
      
      const dx = cameraPositionRef.current.x - cameraTargetRef.current.x;
      const dz = cameraPositionRef.current.z - cameraTargetRef.current.z;
      
      cameraPositionRef.current.x = cameraTargetRef.current.x + dx * cos - dz * sin;
      cameraPositionRef.current.z = cameraTargetRef.current.z + dx * sin + dz * cos;
    }
    
    if (controls.rotateRight) {
      console.log('Camera: Rotating right');
      const angle = -rotateSpeed * delta;
      const cos = Math.cos(angle);
      const sin = Math.sin(angle);
      
      const dx = cameraPositionRef.current.x - cameraTargetRef.current.x;
      const dz = cameraPositionRef.current.z - cameraTargetRef.current.z;
      
      cameraPositionRef.current.x = cameraTargetRef.current.x + dx * cos - dz * sin;
      cameraPositionRef.current.z = cameraTargetRef.current.z + dx * sin + dz * cos;
    }

    // Zoom - Plus/Minus for getting closer/farther from target
    if (controls.zoomIn) {
      console.log('Camera: Zooming in');
      const direction = new THREE.Vector3()
        .subVectors(cameraTargetRef.current, cameraPositionRef.current)
        .normalize()
        .multiplyScalar(zoomSpeed * delta);
      
      const newPos = cameraPositionRef.current.clone().add(direction);
      const distance = newPos.distanceTo(cameraTargetRef.current);
      
      // Minimum zoom distance
      if (distance > 3) {
        cameraPositionRef.current.copy(newPos);
      }
    }
    
    if (controls.zoomOut) {
      console.log('Camera: Zooming out');
      const direction = new THREE.Vector3()
        .subVectors(cameraPositionRef.current, cameraTargetRef.current)
        .normalize()
        .multiplyScalar(zoomSpeed * delta);
      
      const newPos = cameraPositionRef.current.clone().add(direction);
      const distance = newPos.distanceTo(cameraTargetRef.current);
      
      // Maximum zoom distance
      if (distance < 50) {
        cameraPositionRef.current.copy(newPos);
      }
    }

    // Apply smooth camera movement
    camera.position.lerp(cameraPositionRef.current, 0.1);
    
    // Calculate look-at point slightly ahead of target for better perspective
    const lookAtTarget = cameraTargetRef.current.clone();
    camera.lookAt(lookAtTarget);
    
    // Update camera offset for follow mode
    if (followTarget && target) {
      cameraOffset.current.copy(cameraPositionRef.current).sub(target);
    }
  });

  return null;
}
