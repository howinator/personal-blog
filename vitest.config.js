import { defineConfig } from 'vitest/config';
import { resolve } from 'path';
import { existsSync } from 'fs';

const genPath = resolve(__dirname, 'site/assets/js/gen/sessions/v1/sessions_pb.ts');
const stubPath = resolve(__dirname, 'site/assets/js/gen-stub.js');

export default defineConfig({
  test: {
    include: ['site/assets/js/**/*.test.js'],
    environment: 'jsdom',
  },
  resolve: {
    alias: existsSync(genPath) ? {} : {
      './gen/sessions/v1/sessions_pb': stubPath,
    },
  },
});
