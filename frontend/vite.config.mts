import { defineConfig, loadEnv } from 'vite';
import react from '@vitejs/plugin-react';
import relay from 'vite-plugin-relay';
import checker from 'vite-plugin-checker';
import mkcert from 'vite-plugin-mkcert';

export default defineConfig(({ mode }) => {
    // Load env variables based on the current mode (development, production, etc.)
    const env = loadEnv(mode, process.cwd(), '');

    const host = env.VITE_HOST;
    const plugins = env.VITE_ENABLE_HTTPS === 'true' ? [mkcert({ hosts: host ? [host] : [] })] : [];

    return {
        server: {
            open: true,
            port: 3000,
            host: host
        },
        build: {
            sourcemap: false
        },
        plugins: [
            ...plugins,
            relay,
            react({
                babel: {
                    plugins: ['relay']
                },
            }),
            checker({
                typescript: true,
            })
        ]
    };
});
