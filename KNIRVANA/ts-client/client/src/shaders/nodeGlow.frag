uniform float time;
uniform float intensity;
uniform vec3 color;
varying vec2 vUv;
varying vec3 vPosition;

void main() {
  // Create pulsing glow effect
  float pulse = sin(time * 3.0) * 0.5 + 0.5;
  
  // Distance from center for radial gradient
  float dist = distance(vUv, vec2(0.5));
  
  // Create glow
  float glow = 1.0 - smoothstep(0.0, 0.7, dist);
  glow = pow(glow, 2.0);
  
  // Apply color and intensity
  vec3 finalColor = color * glow * intensity * (0.8 + pulse * 0.4);
  
  // Add some transparency based on glow
  float alpha = glow * intensity;
  
  gl_FragColor = vec4(finalColor, alpha);
}
