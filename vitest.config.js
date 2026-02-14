import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    include: ['assets/js/**/*.test.js'],
    environment: 'jsdom',
  },
});
