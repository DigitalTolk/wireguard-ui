import path from "path"
import { defineConfig } from "vite"
import react from "@vitejs/plugin-react"
import tailwindcss from "@tailwindcss/vite"

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  build: {
    outDir: "../assets",
    emptyOutDir: true,
    assetsDir: "static",
  },
  base: "/",
  server: {
    proxy: {
      "/api": "http://localhost:5000",
    },
  },
})
