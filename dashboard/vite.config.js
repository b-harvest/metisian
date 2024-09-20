import path from 'path';

import { defineConfig, loadEnv } from 'vite';
import react from '@vitejs/plugin-react';

export default ({ mode }) => {
  process.env = { ...process.env, ...loadEnv(mode, process.cwd()) };

  const config = {
    plugins: [react()],
    resolve: {
      base: '/',
      alias: {
        '@': path.resolve(__dirname, 'src'),
      },
      extensions: ['.mjs', '.js', '.ts', '.jsx', '.tsx', '.json', '.vue'],
    },
    server: {
      port: 5173,
      host: "0.0.0.0"
    },
  };
  return defineConfig(config);
};
