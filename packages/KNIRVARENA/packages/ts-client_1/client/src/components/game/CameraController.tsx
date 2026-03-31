import { useRef, useEffect, useState } from "react";
import { useFrame, useThree } from "@react-three/fiber";
import { useKeyboardControls } from "@react-three/drei";
import * as THREE from "three";
import { useKnirvana } from "../../lib/stores/useKnirvana";

export default function CameraController() {
  const { camera, gl } = useThree();
  const cameraTargetRef = useRef(new THREE.Vector3(0, 0, 0));
  const cameraPositionRef = useRef(new THREE.Vector3(15, 20, 15));

  const [, get] = useKeyboardControls();
  const { selectedAgent, agents } = useKnirvana();

  // Mouse/pointer control state
  const [isPointerDown, setIsPointerDown] = useState(false);
  const [lastPointerPosition, setLastPointerPosition] = useState({ x: 0, y: 0 });
  const [isDragging, setIsDragging] = useState(false);
  const [userControlled, setUserControlled] = useState(false); // Track if user has manually controlled camera
  const [lastClickTime, setLastClickTime] = useState(0);

  useEffect(() => {
    console.log("Camera controller initialized");

    const canvas = gl.domElement;

    const handlePointerDown = (event: PointerEvent) => {
      if (event.button === 0) { // Left mouse button
        // Check if the event target is the canvas (not a 3D object)
        if (event.target === canvas) {
          setIsPointerDown(true);
          setLastPointerPosition({ x: event.clientX, y: event.clientY });
          setIsDragging(false);
          canvas.setPointerCapture(event.pointerId);
        }
      }
    };

    const handlePointerMove = (event: PointerEvent) => {
      if (isPointerDown) {
        const deltaX = event.clientX - lastPointerPosition.x;
        const deltaY = event.clientY - lastPointerPosition.y;

        // Only start dragging if we've moved a minimum distance
        if (!isDragging && (Math.abs(deltaX) > 3 || Math.abs(deltaY) > 3)) {
          setIsDragging(true);
          setUserControlled(true);
        }

        if (isDragging) {
          // Horizontal movement - rotate around Y axis
          const rotateSpeed = 0.005;
          const angle = deltaX * rotateSpeed;

          // Get current camera position relative to target
          const dx = cameraPositionRef.current.x - cameraTargetRef.current.x;
          const dz = cameraPositionRef.current.z - cameraTargetRef.current.z;

          // Rotate around target
          const cos = Math.cos(angle);
          const sin = Math.sin(angle);

          cameraPositionRef.current.x = cameraTargetRef.current.x + dx * cos - dz * sin;
          cameraPositionRef.current.z = cameraTargetRef.current.z + dx * sin + dz * cos;

          // Vertical movement - adjust height and zoom
          const zoomSpeed = 0.02;
          const heightSpeed = 0.05;

          cameraPositionRef.current.y += deltaY * heightSpeed;

          // Zoom in/out based on vertical movement
          const direction = new THREE.Vector3()
            .subVectors(cameraTargetRef.current, cameraPositionRef.current)
            .normalize()
            .multiplyScalar(deltaY * zoomSpeed);
          cameraPositionRef.current.add(direction);
        }

        setLastPointerPosition({ x: event.clientX, y: event.clientY });
      }
    };

    const handlePointerUp = (event: PointerEvent) => {
      if (event.button === 0) {
        const currentTime = Date.now();

        // Check for double-click (within 300ms and not dragging)
        if (!isDragging && currentTime - lastClickTime < 300) {
          // Double-click detected - reset camera
          setUserControlled(false);
          const defaultTarget = new THREE.Vector3(0, 0, 0);
          const defaultPosition = new THREE.Vector3(15, 20, 15);
          cameraTargetRef.current.copy(defaultTarget);
          cameraPositionRef.current.copy(defaultPosition);
        }

        setLastClickTime(currentTime);
        setIsPointerDown(false);
        setIsDragging(false);
        canvas.releasePointerCapture(event.pointerId);
      }
    };

    const handleWheel = (event: WheelEvent) => {
      event.preventDefault();
      setUserControlled(true);

      const zoomSpeed = 0.002;
      const direction = new THREE.Vector3()
        .subVectors(cameraTargetRef.current, cameraPositionRef.current)
        .normalize()
        .multiplyScalar(event.deltaY * zoomSpeed);
      cameraPositionRef.current.add(direction);
    };

    // Add event listeners
    canvas.addEventListener('pointerdown', handlePointerDown);
    canvas.addEventListener('pointermove', handlePointerMove);
    canvas.addEventListener('pointerup', handlePointerUp);
    canvas.addEventListener('wheel', handleWheel, { passive: false });

    return () => {
      canvas.removeEventListener('pointerdown', handlePointerDown);
      canvas.removeEventListener('pointermove', handlePointerMove);
      canvas.removeEventListener('pointerup', handlePointerUp);
      canvas.removeEventListener('wheel', handleWheel);
    };
  }, [gl.domElement, isPointerDown, lastPointerPosition, isDragging, lastClickTime]);

  useFrame((state, delta) => {
    const controls = get();
    const moveSpeed = 10;
    const rotateSpeed = 1;
    const zoomSpeed = 5;

    // TEMPORARILY DISABLED: Handle agent selection - only if user hasn't manually controlled camera
    // This is disabled to prevent crashes - agent selection will work but won't auto-focus camera
    /*
    if (selectedAgent && !userControlled) {
      const agent = agents.find(a => a.id === selectedAgent);
      if (agent && agent.position) {
        console.log("Focusing on agent:", selectedAgent, "at position:", agent.position);

        // Ensure agent position is valid
        const agentX = isFinite(agent.position.x) ? agent.position.x : 0;
        const agentY = isFinite(agent.position.y) ? agent.position.y : 0;
        const agentZ = isFinite(agent.position.z) ? agent.position.z : 0;

        const targetPos = new THREE.Vector3(agentX, agentY + 2, agentZ);
        // Smoothly interpolate camera target towards the selected agent
        cameraTargetRef.current.lerp(targetPos, delta * 1);

        // Adjust camera position to maintain good viewing angle
        const idealCameraPos = new THREE.Vector3(
          agentX + 8,
          agentY + 12,
          agentZ + 8
        );
        cameraPositionRef.current.lerp(idealCameraPos, delta * 0.8);
      } else {
        console.warn("Agent not found or invalid position:", selectedAgent, agent);
      }
    } else if (!selectedAgent && !userControlled) {
      // Only return to default view if user hasn't manually controlled camera
      const defaultTarget = new THREE.Vector3(0, 0, 0);
      const defaultPosition = new THREE.Vector3(15, 20, 15);
      cameraTargetRef.current.lerp(defaultTarget, delta * 0.5);
      cameraPositionRef.current.lerp(defaultPosition, delta * 0.5);
    }
    */

    // Simple logging for debugging
    if (selectedAgent) {
      console.log("Agent selected:", selectedAgent, "- Camera auto-focus disabled for stability");
    }

    // Keyboard movement (also sets user controlled flag)
    if (controls.forward) {
      setUserControlled(true);
      cameraPositionRef.current.z -= moveSpeed * delta;
      cameraTargetRef.current.z -= moveSpeed * delta;
    }
    if (controls.backward) {
      setUserControlled(true);
      cameraPositionRef.current.z += moveSpeed * delta;
      cameraTargetRef.current.z += moveSpeed * delta;
    }
    if (controls.leftward) {
      setUserControlled(true);
      cameraPositionRef.current.x -= moveSpeed * delta;
      cameraTargetRef.current.x -= moveSpeed * delta;
    }
    if (controls.rightward) {
      setUserControlled(true);
      cameraPositionRef.current.x += moveSpeed * delta;
      cameraTargetRef.current.x += moveSpeed * delta;
    }

    // Rotation around target
    if (controls.rotateLeft) {
      setUserControlled(true);
      const angle = rotateSpeed * delta;
      const cos = Math.cos(angle);
      const sin = Math.sin(angle);
      const dx = cameraPositionRef.current.x - cameraTargetRef.current.x;
      const dz = cameraPositionRef.current.z - cameraTargetRef.current.z;

      cameraPositionRef.current.x = cameraTargetRef.current.x + dx * cos - dz * sin;
      cameraPositionRef.current.z = cameraTargetRef.current.z + dx * sin + dz * cos;
    }
    if (controls.rotateRight) {
      setUserControlled(true);
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
      setUserControlled(true);
      const direction = new THREE.Vector3()
        .subVectors(cameraTargetRef.current, cameraPositionRef.current)
        .normalize()
        .multiplyScalar(zoomSpeed * delta);
      cameraPositionRef.current.add(direction);
    }
    if (controls.zoomOut) {
      setUserControlled(true);
      const direction = new THREE.Vector3()
        .subVectors(cameraPositionRef.current, cameraTargetRef.current)
        .normalize()
        .multiplyScalar(zoomSpeed * delta);
      cameraPositionRef.current.add(direction);
    }

    // Reset camera view (R key twice or double-click)
    if (controls.deploy) { // R key - reuse the deploy key for reset
      setUserControlled(false);
      // Reset to default view
      const defaultTarget = new THREE.Vector3(0, 0, 0);
      const defaultPosition = new THREE.Vector3(15, 20, 15);
      cameraTargetRef.current.copy(defaultTarget);
      cameraPositionRef.current.copy(defaultPosition);
    }

    // Ensure camera position stays within reasonable bounds and is valid
    if (isFinite(cameraPositionRef.current.x) && isFinite(cameraPositionRef.current.y) && isFinite(cameraPositionRef.current.z)) {
      cameraPositionRef.current.x = Math.max(-50, Math.min(50, cameraPositionRef.current.x));
      cameraPositionRef.current.y = Math.max(5, Math.min(100, cameraPositionRef.current.y));
      cameraPositionRef.current.z = Math.max(-50, Math.min(50, cameraPositionRef.current.z));
    } else {
      // Reset to default if position becomes invalid
      cameraPositionRef.current.set(15, 20, 15);
    }

    // Ensure target stays within reasonable bounds and is valid
    if (isFinite(cameraTargetRef.current.x) && isFinite(cameraTargetRef.current.y) && isFinite(cameraTargetRef.current.z)) {
      cameraTargetRef.current.x = Math.max(-30, Math.min(30, cameraTargetRef.current.x));
      cameraTargetRef.current.y = Math.max(-5, Math.min(20, cameraTargetRef.current.y));
      cameraTargetRef.current.z = Math.max(-30, Math.min(30, cameraTargetRef.current.z));
    } else {
      // Reset to default if target becomes invalid
      cameraTargetRef.current.set(0, 0, 0);
    }

    // Apply camera transformations safely
    try {
      camera.position.copy(cameraPositionRef.current);
      camera.lookAt(cameraTargetRef.current);
      camera.updateProjectionMatrix();
    } catch (error) {
      console.warn("Camera update error:", error);
      // Reset camera to safe defaults
      camera.position.set(15, 20, 15);
      camera.lookAt(0, 0, 0);
      camera.updateProjectionMatrix();
    }
  });

  return null;
}
