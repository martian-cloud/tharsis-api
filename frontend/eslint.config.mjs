import eslint from '@eslint/js';
import tseslint from '@typescript-eslint/eslint-plugin';
import tsparser from '@typescript-eslint/parser';

export default [
  eslint.configs.recommended,
  {
    files: ['**/*.ts', '**/*.tsx'],
    languageOptions: {
      parser: tsparser,
      parserOptions: {
        ecmaVersion: 'latest',
        sourceType: 'module',
        ecmaFeatures: {
          jsx: true,
        },
      },
      globals: {
        // Browser globals
        window: 'readonly',
        document: 'readonly',
        navigator: 'readonly',
        console: 'readonly',
        fetch: 'readonly',
        localStorage: 'readonly',
        sessionStorage: 'readonly',
        setTimeout: 'readonly',
        setInterval: 'readonly',
        clearTimeout: 'readonly',
        clearInterval: 'readonly',
        location: 'readonly',
        atob: 'readonly',
        btoa: 'readonly',
        // DOM types
        HTMLElement: 'readonly',
        HTMLInputElement: 'readonly',
        HTMLButtonElement: 'readonly',
        HTMLDivElement: 'readonly',
        HTMLLIElement: 'readonly',
        HTMLTextAreaElement: 'readonly',
        Element: 'readonly',
        Event: 'readonly',
        // Web APIs
        URL: 'readonly',
        URLSearchParams: 'readonly',
        Blob: 'readonly',
        Worker: 'readonly',
        ResizeObserver: 'readonly',
        RequestInfo: 'readonly',
        RequestInit: 'readonly',
        Response: 'readonly',
        // Node globals
        process: 'readonly',
        Buffer: 'readonly',
        __dirname: 'readonly',
        __filename: 'readonly',
        exports: 'writable',
        module: 'readonly',
        require: 'readonly',
        NodeJS: 'readonly',
        // React
        React: 'readonly',
        JSX: 'readonly',
        // Web Workers
        self: 'readonly',
      },
    },
    plugins: {
      '@typescript-eslint': tseslint,
    },
    rules: {
      ...tseslint.configs.recommended.rules,
      '@typescript-eslint/no-explicit-any': 'off',
    },
  },
  {
    ignores: ['dist/*', 'node_modules/*', '**/__generated__/**'],
  },
];
