import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 3000,
    // Serve the data folder
    fs: {
      allow: ['..'],
    },
  },
  // Make data folder accessible
  publicDir: 'public',
})
