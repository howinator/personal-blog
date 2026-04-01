import { describe, it, expect, beforeAll, beforeEach, afterEach, vi } from 'vitest';

vi.mock('@connectrpc/connect', () => ({
  createClient: () => ({
    watchSessions: () => ({ [Symbol.asyncIterator]: () => ({ next: () => new Promise(() => {}) }) })
  })
}));
vi.mock('@connectrpc/connect-web', () => ({
  createConnectTransport: () => ({})
}));
vi.mock('./gen/sessions/v1/sessions_pb', () => ({
  SessionService: {}
}));

import { formatTokens, formatTime, escapeHTML } from './live-status.js';

beforeAll(() => {
  // Set up minimal DOM so init() runs
  document.body.innerHTML = '<span class="cc-status-dot"></span>';
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

describe('sessionStorage hydration', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
    sessionStorage.clear();
  });

  it('clears stale cached sessions after inactivity timeout', async () => {
    // Seed sessionStorage with a stale session
    var staleSession = {
      'test-session-123': {
        session_id: 'test-session-123',
        total_tokens: 1000,
        input_tokens: 500,
        output_tokens: 500,
        cache_read_input_tokens: 0,
        tool_calls: 5,
        user_prompts: 2,
        active_time_seconds: 60,
        last_prompt: 'test prompt',
        project: 'test',
        model: 'claude',
        summary: 'test session',
      }
    };
    sessionStorage.setItem('claug-sessions', JSON.stringify(staleSession));

    // Set up DOM with config element so init() runs, then re-import
    document.body.innerHTML =
      '<span id="cc-config" data-claug-api="http://localhost" data-claug-user="user1"></span>' +
      '<span class="cc-status-dot"></span>';

    vi.resetModules();

    // Re-mock dependencies before re-import
    vi.doMock('@connectrpc/connect', () => ({
      createClient: () => ({
        watchSessions: () => ({ [Symbol.asyncIterator]: () => ({ next: () => new Promise(() => {}) }) })
      })
    }));
    vi.doMock('@connectrpc/connect-web', () => ({
      createConnectTransport: () => ({})
    }));
    vi.doMock('./gen/sessions/v1/sessions_pb', () => ({
      SessionService: {}
    }));

    await import('./live-status.js');

    // Let scheduleSynthesize (requestAnimationFrame) fire
    vi.advanceTimersByTime(0);
    await vi.runAllTicksAsync();

    var dot = document.querySelector('.cc-status-dot');
    expect(dot.classList.contains('active')).toBe(true);

    // Advance past the 30s inactivity check interval — the hydrated session's
    // heartbeat was set to now-80000, so it's now >90s stale and should expire
    vi.advanceTimersByTime(30000);
    await vi.runAllTicksAsync();

    expect(dot.classList.contains('active')).toBe(false);
    expect(sessionStorage.getItem('claug-sessions')).toBeNull();
  });
});
