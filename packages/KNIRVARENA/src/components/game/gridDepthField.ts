// Shared, frame-animated depth field describing where the arena grid has
// sunk beneath error nodes with committed validation rings. ArenaGrid writes
// the animated depths each frame; GridParticleSystem (and anything else that
// rides the grid surface) samples it so particles follow the deformation.

export interface DepthCenter {
  id: string;
  x: number;
  z: number;
  depth: number; // current animated depth (world units, positive = down)
}

export const GRID_SINK_DEPTH = 2.2;  // max depression at the node center
export const GRID_SINK_RADIUS = 5;   // bowl radius around the node

export const depthCenters: DepthCenter[] = [];

export function setDepthCenters(next: DepthCenter[]) {
  depthCenters.length = 0;
  depthCenters.push(...next);
}

/** Parabolic dip at (x, z): depth * (1 - (r/R)^2) inside each bowl. */
export function sampleGridDip(x: number, z: number): number {
  let dip = 0;
  const r2max = GRID_SINK_RADIUS * GRID_SINK_RADIUS;
  for (const c of depthCenters) {
    const dx = x - c.x;
    const dz = z - c.z;
    const r2 = (dx * dx + dz * dz) / r2max;
    if (r2 < 1) dip += c.depth * (1 - r2);
  }
  return dip;
}
