import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [vue()],
  server: {
    host: '127.0.0.1',
    port: parseInt(process.env.WAILS_VITE_PORT || '9245', 10),
    strictPort: true,
  },
})
