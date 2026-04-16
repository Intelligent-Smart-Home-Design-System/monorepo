import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'
import { createRequire } from 'module'
const require = createRequire(import.meta.url)

export default defineConfig({
  plugins: [react({ jsxRuntime: 'classic' })],
  resolve: {
    alias: {
      'react-icons/fa': path.resolve(__dirname, './src/dummy-icons.js'),
      'react-icons/md': path.resolve(__dirname, './src/dummy-icons.js'),
      'react-icons/ti': path.resolve(__dirname, './src/dummy-icons.js'),
      'react-svg-pan-zoom': require.resolve('react-svg-pan-zoom'),
      
      'react': path.resolve(__dirname, 'node_modules/react'),
      'react-dom': path.resolve(__dirname, 'node_modules/react-dom'),
      'react-redux': path.resolve(__dirname, 'node_modules/react-redux'),
      'three': path.resolve(__dirname, 'node_modules/three'),
      'immutable': path.resolve(__dirname, 'node_modules/immutable'),
    },
    dedupe: ['react', 'react-dom', 'react-redux', 'three', 'immutable']
  },
  optimizeDeps: {
    include: [
      'react-planner',
      'react-redux',
      'immutable',
      'three',
      'prop-types',
      'create-react-class',
      'react-svg-pan-zoom'
    ]
  }
})