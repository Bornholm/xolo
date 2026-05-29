import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  define: {
    'process.env.NODE_ENV': JSON.stringify('production'),
  },
  build: {
    outDir: '../internal/http/handler/webui/pipeline/assets/dist',
    emptyOutDir: false, // keep .gitkeep placeholder
    lib: {
      entry: 'src/main.tsx',
      name: 'PipelineEditor',
      fileName: () => 'pipeline-editor.js',
      formats: ['iife'],
    },
    cssCodeSplit: false,
    rollupOptions: {
      output: {
        assetFileNames: 'pipeline-editor[extname]',
      },
    },
  },
})
