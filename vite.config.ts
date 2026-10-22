import { defineConfig } from "vite";

export default defineConfig({
  root: "frontend",
  build: {
    outDir: "../public/dist",
    emptyOutDir: true
  },
  server: {
    port: 5173,
    proxy: { "/ws": { target: "ws://127.0.0.1:4000", ws: true } }
  }
});
