import * as THREE from 'three';

declare module 'three' {
  interface Object3D {
    object_type: string;
    name: string;
  }
}