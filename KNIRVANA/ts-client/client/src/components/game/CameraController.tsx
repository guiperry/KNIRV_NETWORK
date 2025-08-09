import { useRef, useEffect } from "react";
import { useFrame, useThree } from "@react-three/fiber";
import { useKeyboardControls } from "@react-three/drei";
import * as THREE from "three";

export default function CameraController() {
  const { camera } = useThree();
  const cameraTargetRef = useRef(new THREE.Vector3(0, 0, 0));
  const cameraPositionRef = useRef(new THREE.Vector3(15, 20, 15));
  
  const [, get] = useKeyboardControls();

  useEffect(() => {
    console.log("Camera controller initialized");
  }, []);

  useFrame((state, delta) => {
    const controls = get();
    const moveSpeed = 10;
    const rotateSpeed = 1;
    const zoomSpeed = 5;

    // Movement
    if (controls.forward) {
      console.log("Moving camera forward");
      cameraPositionRef.current.z -= moveSpeed * delta;
      cameraTargetRef.current.z -= moveSpeed * delta;
    }
    if (controls.backward) {
      console.log("Moving camera backward");
      cameraPositionRef.current.z += moveSpeed * delta;
      cameraTargetRef.current.z += moveSpeed * delta;
    }
    if (controls.leftward) {
      console.log("Moving camera left");
      cameraPositionRef.current.x -= moveSpeed * delta;
      cameraTargetRef.current.x -= moveSpeed * delta;
    }
    if (controls.rightward) {
      console.log("Moving camera right");
      cameraPositionRef.current.x += moveSpeed * delta;
      cameraTargetRef.current.x += moveSpeed * delta;
    }

    // Rotation around target
    if (controls.rotateLeft) {
      console.log("Rotating camera left");
      const angle = rotateSpeed * delta;
      const cos = Math.cos(angle);
      const sin = Math.sin(angle);
      const dx = cameraPositionRef.current.x - cameraTargetRef.current.x;
      const dz = cameraPositionRef.current.z - cameraTargetRef.current.z;
      
      cameraPositionRef.current.x = cameraTargetRef.current.x + dx * cos - dz * sin;
      cameraPositionRef.current.z = cameraTargetRef.current.z + dx * sin + dz * cos;
    }
    if (controls.rotateRight) {
      console.log("Rotating camera right");
      const angle = -rotateSpeed * delta;
      const cos = Math.cos(angle);
      const sin = Math.sin(angle);
      const dx = cameraPositionRef.current.x - cameraTargetRef.current.x;
      const dz = cameraPositionRef.current.z - cameraTargetRef.current.z;
      
      cameraPositionRef.current.x = cameraTargetRef.current.x + dx * cos - dz * sin;
      cameraPositionRef.current.z = cameraTargetRef.current.z + dx * sin + dz * cos;
    }

    // Zoom
    if (controls.zoomIn) {
      console.log("Zooming camera in");
      const direction = new THREE.Vector3()
        .subVectors(cameraTargetRef.current, cameraPositionRef.current)
        .normalize()
        .multiplyScalar(zoomSpeed * delta);
      cameraPositionRef.current.add(direction);
    }
    if (controls.zoomOut) {
      console.log("Zooming camera out");
      const direction = new THREE.Vector3()
        .subVectors(cameraPositionRef.current, cameraTargetRef.current)
        .normalize()
        .multiplyScalar(zoomSpeed * delta);
      cameraPositionRef.current.add(direction);
    }

    // Apply camera transformations
    camera.position.copy(cameraPositionRef.current);
    camera.lookAt(cameraTargetRef.current);
  });

  return null;
}
