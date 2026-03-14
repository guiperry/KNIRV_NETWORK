import js from '@eslint/js';
import tseslint from 'typescript-eslint';
import react from 'eslint-plugin-react';
import reactHooks from 'eslint-plugin-react-hooks';
import reactRefresh from 'eslint-plugin-react-refresh';
import globals from 'globals';

export default tseslint.config(
  { ignores: [
    'dist',
    'node_modules',
    'coverage',
    'test-results',
    'src/core/agent-core-compiler/build/**/*.ts',
    'src/core/agent-core-compiler/build/**/*.js',
    'assembly/index.ts',
    'browser-bridge/packages/knirvwallet-extension/.storybook/**'
  ] },
  {
    extends: [js.configs.recommended, ...tseslint.configs.recommended],
    files: ['**/*.{ts,tsx}'],
    languageOptions: {
      ecmaVersion: 2020,
      globals: globals.browser,
      parserOptions: {
        ecmaFeatures: {
          jsx: true,
        },
      },
    },
    plugins: {
      'react': react,
      'react-hooks': reactHooks,
      'react-refresh': reactRefresh,
    },
    settings: {
      react: {
        version: 'detect',
      },
    },
    rules: {
      ...react.configs.recommended.rules,
      ...reactHooks.configs.recommended.rules,
      'react-refresh/only-export-components': [
        'warn',
        { allowConstantExport: true },
      ],
      'react/no-unknown-property': ['error', { ignore: [
        // SVG props
        'stroke', 'strokeWidth', 'strokeLinecap', 'viewBox', 'xmlns', 'fill',
        'x1', 'y1', 'x2', 'y2', 'cx', 'cy', 'r', 'd', 'transform',
        'clipPath', 'clipPathUnits', 'mask', 'maskUnits', 'patternUnits', 'patternContentUnits',
        // Three.js/React Three Fiber props
        'rotation', 'position', 'scale', 'args', 'attach', 'count', 'array', 'itemSize',
        'receiveShadow', 'castShadow', 'intensity', 'distance', 'decay', 'angle', 'penumbra',
        'emissive', 'emissiveIntensity', 'roughness', 'metalness', 'wireframe', 'transparent',
        'opacity', 'geometry', 'material', 'vertexColors', 'sizeAttenuation', 'fog',
        'shadow-mapSize-width', 'shadow-mapSize-height', 'shadow-camera-far',
        'shadow-camera-left', 'shadow-camera-right', 'shadow-camera-top', 'shadow-camera-bottom',
        'shadow-bias', 'shadow-radius', 'shadow-map', 'shadow-mapSize', 'shadow-camera',
        'dispose', 'onUpdate', 'onPointerOver', 'onPointerOut', 'onClick', 'visible',
        'userData', 'layers', 'matrix', 'quaternion', 'up', 'lookAt', 'add', 'remove',
        'traverse', 'raycast', 'updateMatrix', 'updateMatrixWorld', 'clone', 'copy'
      ] }],
      '@typescript-eslint/no-unused-vars': ['error', { argsIgnorePattern: '^_' }],
      '@typescript-eslint/no-explicit-any': 'warn',
    },
  },
);
