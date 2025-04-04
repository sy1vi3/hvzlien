import { defineConfig } from 'vite'
import solid from 'vite-plugin-solid'
import tailwindcss from '@tailwindcss/vite';


export default defineConfig({
  plugins: [
    tailwindcss(),
    solid()
  ],
  server: {
    proxy: {
      "/api": "http://serval-wtdrofqg:8080"
    }
  }
})
