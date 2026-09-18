import path from 'path';
import { fileURLToPath } from 'url';
import { defineConfig, loadEnv } from 'vite';
import react from '@vitejs/plugin-react';
import relay from 'vite-plugin-relay';
import checker from 'vite-plugin-checker';
import eslint from 'vite-plugin-eslint2';
import mkcert from 'vite-plugin-mkcert';

const __dirname = fileURLToPath(new URL('.', import.meta.url));

export default defineConfig(({ mode }) => {
    // Load env variables based on the current mode (development, production, etc.)
    const env = loadEnv(mode, process.cwd(), '');

    const host = env.VITE_HOST;
    const port = Number(env.VITE_PORT) || 3000;
    const plugins = env.VITE_ENABLE_HTTPS === 'true' ? [mkcert({ hosts: host ? [host] : [] })] : [];

    return {
        resolve: {
            alias: {
                '@': path.resolve(__dirname, 'src')
            }
        },
        server: {
            open: true,
            port: port,
            host: host
        },
        build: {
            sourcemap: false
        },
        define: {
            'import.meta.env.VITE_THARSIS_API_URL': JSON.stringify(
                env.VITE_THARSIS_API_URL || ''
            ),
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
                eslint: {
                    lintCommand: 'eslint --max-warnings=0 .',
                    useFlatConfig: true
                }
            }),
        ]
    };
});
