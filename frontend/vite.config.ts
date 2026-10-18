import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  appType: 'spa',
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:5151',
        changeOrigin: true,
      },
      '/health': {
        target: 'http://localhost:5151',
        changeOrigin: true,
      },
      '/ready': {
        target: 'http://localhost:5151',
        changeOrigin: true,
      },
    },
  },
});

