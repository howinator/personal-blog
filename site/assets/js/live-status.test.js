import { describe, it, expect, beforeAll } from 'vitest';
import { readFileSync } from 'fs';
import { resolve } from 'path';

let formatTokens, formatTime, escapeHTML;

beforeAll(() => {
  // Set up minimal DOM so the IIFE doesn't bail early
  document.body.innerHTML = '<span class="cc-status-dot"></span>';

  // Stub WebSocket so connect() doesn't throw
  globalThis.WebSocket = class {
    constructor() { this.onmessage = null; this.onclose = null; }
    close() {}
  };

  // Signal to the script that we're in test mode
  globalThis.__TEST__ = true;

  // Evaluate the script in the jsdom context
  const code = readFileSync(resolve(__dirname, 'live-status.js'), 'utf-8');
  const fn = new Function(code);
  fn();

  ({ formatTokens, formatTime, escapeHTML } = globalThis.__liveStatus);
});

describe('formatTokens', () => {
  it('formats numbers under 1000', () => {
    expect(formatTokens(0)).toBe('0');
    expect(formatTokens(999)).toBe('999');
  });

  it('formats thousands with k suffix', () => {
    expect(formatTokens(1000)).toBe('1.0k');
    expect(formatTokens(45200)).toBe('45.2k');
  });

  it('formats millions with M suffix', () => {
    expect(formatTokens(1000000)).toBe('1.0M');
    expect(formatTokens(1500000)).toBe('1.5M');
  });
});

describe('formatTime', () => {
  it('formats seconds only', () => {
    expect(formatTime(0)).toBe('0s');
    expect(formatTime(45)).toBe('45s');
  });

  it('formats minutes', () => {
    expect(formatTime(60)).toBe('1m');
    expect(formatTime(120)).toBe('2m');
    expect(formatTime(125)).toBe('2m 5s');
  });

  it('formats hours', () => {
    expect(formatTime(3600)).toBe('1h');
    expect(formatTime(3725)).toBe('1h 2m');
  });
});

describe('escapeHTML', () => {
  it('escapes angle brackets', () => {
    expect(escapeHTML('<script>alert("xss")</script>')).toBe(
      '&lt;script&gt;alert("xss")&lt;/script&gt;'
    );
  });

  it('escapes ampersands', () => {
    expect(escapeHTML('a & b')).toBe('a &amp; b');
  });

  it('passes through safe strings', () => {
    expect(escapeHTML('hello world')).toBe('hello world');
  });
});
