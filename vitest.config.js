import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    include: ['site/assets/js/**/*.test.js'],
    environment: 'jsdom',
  },
});
