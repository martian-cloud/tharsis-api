import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';
import relay from 'vite-plugin-relay';

export default defineConfig({
    plugins: [
        react({
            babel: {
                plugins: ['relay']
            },
        }),
        relay,
    ],
    test: {
        globals: true,
        environment: 'jsdom',
        setupFiles: ['./src/setupTests.ts'],
    },
});
