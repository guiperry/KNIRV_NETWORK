uniform float time;
uniform float intensity;
varying vec2 vUv;
varying vec3 vPosition;

void main() {
  vUv = uv;
  vPosition = position;
  
  // Add subtle vertex displacement for glow effect
  vec3 newPosition = position;
  newPosition += normal * sin(time * 2.0 + position.x * 10.0) * 0.01 * intensity;
  
  gl_Position = projectionMatrix * modelViewMatrix * vec4(newPosition, 1.0);
}
