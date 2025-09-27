import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
    plugins: [react()],
    server: {
        host: "0.0.0.0",
        origin: 'http://localhost:5173', // exposed node container address
        port: 5173,
        proxy: {
        '/api': {
            target: 'https://backend:8080',
            secure: false, // Set to false since backend uses self-signed certs.
        },
        },
    },
})
