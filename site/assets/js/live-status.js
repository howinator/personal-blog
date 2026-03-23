(function() {
  var dots = document.querySelectorAll('.cc-status-dot');
  if (!dots.length) return;

  // Read Connect API URL and user ID from Hugo params (data attributes on nav dot)
  var apiEl = document.querySelector('[data-claug-api]');
  var apiUrl = apiEl ? apiEl.getAttribute('data-claug-api') : '';
  var userId = apiEl ? apiEl.getAttribute('data-claug-user') : '';
  if (!apiUrl) {
    apiUrl = location.protocol + '//' + location.host;
  }

  // Read base totals from data attributes (only on claude-log page)
  var baseTotals = {};
  var statEls = {};
  ['sessions', 'tokens', 'active-time', 'tool-calls'].forEach(function(stat) {
    var el = document.querySelector('[data-stat="' + stat + '"]');
    if (el) {
      statEls[stat] = el;
      baseTotals[stat] = Math.round(Number(el.getAttribute('data-raw'))) || 0;
    }
  });
  var toolsTip = document.getElementById('cc-tools-tip');
  var baseTopTools = [];
  if (toolsTip) {
    try {
      baseTopTools = JSON.parse(toolsTip.getAttribute('data-tools') || '[]');
    } catch(e) {}
    statEls['tools-tip'] = toolsTip;
  }

  var tokenTip = document.getElementById('cc-token-tip');
  if (tokenTip) {
    statEls['token-tip'] = tokenTip;
    baseTotals['input-raw'] = Math.round(Number(tokenTip.getAttribute('data-input-raw'))) || 0;
    baseTotals['cache-raw'] = Math.round(Number(tokenTip.getAttribute('data-cache-raw'))) || 0;
    baseTotals['output-raw'] = Math.round(Number(tokenTip.getAttribute('data-output-raw'))) || 0;
  }

  // Collect static session IDs so we know which live sessions are already in the base totals
  var staticSessionIds = {};
  document.querySelectorAll('[data-session-id]').forEach(function(el) {
    staticSessionIds[el.getAttribute('data-session-id')] = true;
  });

  var liveContainer = document.getElementById('cc-live-sessions');
  var liveSessions = {};
  var typewriterTimers = {};
  var lastPromptSeen = {};

  // Claug incremental event tracking
  var activeSessions = {};
  var lastHeartbeatAt = {};

  function setActive(active) {
    for (var i = 0; i < dots.length; i++) {
      if (active) {
        dots[i].classList.add('active');
        dots[i].title = 'Claude Code: active';
      } else {
        dots[i].classList.remove('active');
        dots[i].title = 'Claude Code: offline';
      }
    }
  }

  function cleanToolName(name) {
    var parts = name.split('__');
    if (parts.length >= 3 && parts[0] === 'mcp') {
      var providerParts = parts[1].split('_');
      var service = providerParts[providerParts.length - 1];
      return service + ': ' + parts.slice(2).join('__');
    }
    return name;
  }

  function formatCount(n) {
    return String(n).replace(/\B(?=(\d{3})+(?!\d))/g, ',');
  }

  function formatTokens(n) {
    if (n >= 1000000) return (n / 1000000).toFixed(1) + 'M';
    if (n >= 1000) return (n / 1000).toFixed(1) + 'k';
    return String(n);
  }

  function formatTime(sec) {
    if (sec < 60) return sec + 's';
    var minutes = Math.floor(sec / 60);
    var secs = sec % 60;
    if (minutes < 60) {
      return secs ? minutes + 'm ' + secs + 's' : minutes + 'm';
    }
    var hours = Math.floor(minutes / 60);
    var mins = minutes % 60;
    var parts = [hours + 'h'];
    if (mins) parts.push(mins + 'm');
    return parts.join(' ');
  }

  function updateStatDisplay(el, newText) {
    if (!el) return;
    var oldText = el.textContent;
    if (oldText === newText) return;

    // Pad shorter string on the left so digits align from the right
    var maxLen = Math.max(oldText.length, newText.length);
    var oldChars = oldText.padStart(maxLen).split('');
    var newChars = newText.padStart(maxLen).split('');

    el.innerHTML = '';
    for (var i = 0; i < maxLen; i++) {
      var span = document.createElement('span');
      span.className = 'cc-digit';
      span.textContent = newChars[i];
      if (oldChars[i] !== newChars[i]) {
        span.classList.add('cc-digit-enter');
      }
      el.appendChild(span);
    }

    // Trigger reflow then remove enter class to animate
    el.offsetHeight; // force reflow
    var entering = el.querySelectorAll('.cc-digit-enter');
    for (var j = 0; j < entering.length; j++) {
      entering[j].classList.remove('cc-digit-enter');
    }
  }

  function recalcAggregates() {
    if (!statEls.sessions) return; // not on claude-log page

    var deltaTokens = 0;
    var deltaTime = 0;
    var deltaTools = 0;
    var deltaCount = 0;

    var deltaInput = 0;
    var deltaCacheRead = 0;
    var deltaOutput = 0;
    for (var id in liveSessions) {
      // Skip sessions already counted in the static base totals
      if (staticSessionIds[id]) continue;
      var sess = liveSessions[id];
      deltaCount++;
      deltaTokens += sess.total_tokens || 0;
      deltaInput += sess.input_tokens || 0;
      deltaCacheRead += sess.cache_read_input_tokens || 0;
      deltaOutput += sess.output_tokens || 0;
      deltaTime += sess.active_time_seconds || 0;
      deltaTools += sess.tool_calls || 0;
    }

    updateStatDisplay(statEls['sessions'], String(baseTotals['sessions'] + deltaCount));
    updateStatDisplay(statEls['tokens'], formatTokens(baseTotals['tokens'] + deltaTokens));
    updateStatDisplay(statEls['active-time'], formatTime(baseTotals['active-time'] + deltaTime));
    updateStatDisplay(statEls['tool-calls'], String(baseTotals['tool-calls'] + deltaTools));

    // Recalc top tools tooltip
    if (statEls['tools-tip']) {
      var merged = {};
      // Start with base tools
      for (var ti = 0; ti < baseTopTools.length; ti++) {
        merged[baseTopTools[ti].name] = baseTopTools[ti].count;
      }
      // Merge live session tool_counts
      for (var lid in liveSessions) {
        if (staticSessionIds[lid]) continue;
        var tc = liveSessions[lid].tool_counts;
        if (tc) {
          for (var toolName in tc) {
            merged[toolName] = (merged[toolName] || 0) + tc[toolName];
          }
        }
      }
      // Sort and take top 5
      var toolArr = [];
      for (var tn in merged) {
        toolArr.push({name: tn, count: merged[tn], display: cleanToolName(tn)});
      }
      toolArr.sort(function(a, b) { return b.count - a.count; });
      if (toolArr.length > 5) toolArr = toolArr.slice(0, 5);
      var tipParts = [];
      for (var tj = 0; tj < toolArr.length; tj++) {
        tipParts.push(toolArr[tj].display + ' ' + formatCount(toolArr[tj].count));
      }
      statEls['tools-tip'].innerHTML = tipParts.join('<br>');
    }

    if (statEls['token-tip']) {
      var tipEl = statEls['token-tip'];
      var newInput = (baseTotals['input-raw'] || 0) + deltaInput;
      var newCache = (baseTotals['cache-raw'] || 0) + deltaCacheRead;
      var newOutput = (baseTotals['output-raw'] || 0) + deltaOutput;
      tipEl.innerHTML = formatTokens(newInput) + ' input<br>' +
                        formatTokens(newCache) + ' cached<br>' +
                        formatTokens(newOutput) + ' output';
    }
  }

  // Enhance an existing static card with liveness indicators
  function enhanceStaticCard(card, session) {
    card.classList.add('cc-session-live');

    var summary = card.querySelector('summary');
    if (summary) {
      // Inject green dot + Live label after caret (if not already present)
      if (!summary.querySelector('.cc-live-label')) {
        var dot = document.createElement('span');
        dot.className = 'cc-status-dot cc-status-dot-inline active cc-live-injected';
        var label = document.createElement('span');
        label.className = 'cc-live-label cc-live-injected';
        label.textContent = 'Live';
        var caret = summary.querySelector('.cc-session-caret');
        if (caret && caret.nextSibling) {
          summary.insertBefore(label, caret.nextSibling);
          summary.insertBefore(dot, caret.nextSibling);
        }
      }

      // Update token count in summary
      var tokensEl = summary.querySelector('.cc-session-tokens');
      if (tokensEl) {
        tokensEl.textContent = formatTokens(session.total_tokens || 0) + ' tokens';
      }
    }

    // Add/update typewriter prompt in details
    var details = card.querySelector('.cc-session-details');
    if (details && session.last_prompt) {
      var promptId = 'cc-live-' + session.session_id + '-prompt';
      var promptContainer = details.querySelector('.cc-live-prompt');
      var promptLabel = session.sensitive ? 'Latest Prompt (redacted)' : 'Latest Prompt';
      if (!promptContainer) {
        promptContainer = document.createElement('div');
        promptContainer.className = 'cc-live-prompt cc-live-injected';
        promptContainer.innerHTML = '<span style="font-family:var(--font-mono);font-size:0.8rem;color:var(--muted)">' + promptLabel + '</span>';
        var promptEl = document.createElement('div');
        promptEl.className = 'cc-typewriter' + (session.sensitive ? ' cc-typewriter-redacted' : '');
        promptEl.id = promptId;
        promptContainer.appendChild(promptEl);
        details.appendChild(promptContainer);
      } else {
        var labelEl = promptContainer.querySelector('span');
        if (labelEl) labelEl.textContent = promptLabel;
        var existingPromptEl = document.getElementById(promptId);
        if (existingPromptEl) {
          if (session.sensitive) existingPromptEl.classList.add('cc-typewriter-redacted');
          else existingPromptEl.classList.remove('cc-typewriter-redacted');
        }
      }

      var promptEl = document.getElementById(promptId);
      if (promptEl) {
        if (lastPromptSeen[session.session_id] !== session.last_prompt) {
          lastPromptSeen[session.session_id] = session.last_prompt;
          typewriterAnimate(promptEl, session.last_prompt);
        } else if (promptEl.textContent !== session.last_prompt) {
          // Element was recreated (e.g. after reconnect) — set text without animation
          promptEl.textContent = session.last_prompt;
        }
      }
    }
  }

  // Revert a static card to its original state
  function revertStaticCard(card) {
    card.classList.remove('cc-session-live');
    var injected = card.querySelectorAll('.cc-live-injected');
    for (var i = 0; i < injected.length; i++) {
      injected[i].remove();
    }
  }

  // Create a brand-new live card (for sessions not in static data)
  function createLiveCard(session) {
    if (!liveContainer) return;

    var cardId = 'cc-live-' + session.session_id;
    var card = document.getElementById(cardId);

    if (!card) {
      card = document.createElement('details');
      card.className = 'cc-session cc-session-live';
      card.id = cardId;
      card.open = false;
      liveContainer.appendChild(card);
    }

    var tokenDisplay = formatTokens(session.total_tokens || 0);
    var timeDisplay = formatTime(session.active_time_seconds || 0);

    // Build summary line
    var summaryHTML =
      '<span class="cc-session-caret">&#9654;</span>' +
      '<span class="cc-status-dot cc-status-dot-inline active"></span>' +
      '<span class="cc-live-label">Live</span>' +
      '<span class="cc-session-summary">' + escapeHTML(session.summary || session.project || 'unknown') + '</span>' +
      '<span class="cc-session-tokens">' + tokenDisplay + ' tokens</span>';

    var summary = card.querySelector('summary');
    if (!summary) {
      summary = document.createElement('summary');
      card.appendChild(summary);
    }
    summary.innerHTML = summaryHTML;

    var detailsDiv = card.querySelector('.cc-session-details');
    if (!detailsDiv) {
      detailsDiv = document.createElement('div');
      detailsDiv.className = 'cc-session-details';
      card.appendChild(detailsDiv);
    }

    // Update or create the stats table (without touching the prompt element)
    var tableHTML =
      '<table>' +
      '<tr><td>Model</td><td>' + escapeHTML(session.model || '') + '</td></tr>' +
      '<tr><td>User Prompts</td><td>' + (session.user_prompts || 0) + '</td></tr>' +
      '<tr><td>Tool Calls</td><td>' + (session.tool_calls || 0) + '</td></tr>' +
      '<tr><td>Input Tokens</td><td>' + formatTokens(session.input_tokens || 0) + '</td></tr>' +
      '<tr><td>Cache Read Tokens</td><td>' + formatTokens(session.cache_read_input_tokens || 0) + '</td></tr>' +
      '<tr><td>Output Tokens</td><td>' + formatTokens(session.output_tokens || 0) + '</td></tr>' +
      '<tr><td>Total Tokens</td><td>' + tokenDisplay + '</td></tr>' +
      '<tr><td>Active Time</td><td>' + timeDisplay + '</td></tr>' +
      '</table>';

    var table = detailsDiv.querySelector('table');
    if (table) {
      table.outerHTML = tableHTML;
    } else {
      detailsDiv.insertAdjacentHTML('afterbegin', tableHTML);
    }

    // Ensure prompt container exists (create once, then preserve)
    var promptLabel = session.sensitive ? 'Latest Prompt (redacted)' : 'Latest Prompt';
    var promptContainer = detailsDiv.querySelector('.cc-live-prompt-wrap');
    if (!promptContainer) {
      promptContainer = document.createElement('div');
      promptContainer.className = 'cc-live-prompt-wrap';
      promptContainer.style.marginTop = '0.5rem';
      promptContainer.innerHTML =
        '<span class="cc-live-prompt-label" style="font-family:var(--font-mono);font-size:0.8rem;color:var(--muted)">' + promptLabel + '</span>';
      var promptEl = document.createElement('div');
      promptEl.className = 'cc-typewriter' + (session.sensitive ? ' cc-typewriter-redacted' : '');
      promptEl.id = cardId + '-prompt';
      promptContainer.appendChild(promptEl);
      detailsDiv.appendChild(promptContainer);
    } else {
      // Update label and redacted class
      var labelEl = promptContainer.querySelector('.cc-live-prompt-label');
      if (labelEl) labelEl.textContent = promptLabel;
      var existingPromptEl = document.getElementById(cardId + '-prompt');
      if (existingPromptEl) {
        if (session.sensitive) existingPromptEl.classList.add('cc-typewriter-redacted');
        else existingPromptEl.classList.remove('cc-typewriter-redacted');
      }
    }

    // Typewriter for latest prompt
    var promptEl = document.getElementById(cardId + '-prompt');
    if (promptEl && session.last_prompt) {
      if (lastPromptSeen[session.session_id] !== session.last_prompt) {
        lastPromptSeen[session.session_id] = session.last_prompt;
        typewriterAnimate(promptEl, session.last_prompt);
      } else if (promptEl.textContent !== session.last_prompt) {
        // Element was recreated (e.g. after reconnect) — set text without animation
        promptEl.textContent = session.last_prompt;
      }
    }
  }

  function renderLiveSession(session) {
    // Hide any matching static card and always create a live card at the top
    var staticCard = document.querySelector('[data-session-id="' + session.session_id + '"]');
    if (staticCard) {
      staticCard.style.display = 'none';
    }
    createLiveCard(session);
  }

  function typewriterAnimate(el, text) {
    var timerId = el.id;
    if (typewriterTimers[timerId]) {
      clearInterval(typewriterTimers[timerId]);
    }

    el.textContent = '';
    el.classList.add('cc-typewriter-active');
    var idx = 0;

    typewriterTimers[timerId] = setInterval(function() {
      if (idx < text.length) {
        el.textContent += text[idx];
        idx++;
      } else {
        clearInterval(typewriterTimers[timerId]);
        delete typewriterTimers[timerId];
        // Remove cursor after 2s
        setTimeout(function() {
          el.classList.remove('cc-typewriter-active');
        }, 2000);
      }
    }, 25);
  }

  function escapeHTML(str) {
    var div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
  }

  function removeLiveSession(sessionId) {
    // Keep lastPromptSeen so reconnects don't re-trigger the typewriter

    // Clean up typewriter timer
    var timerId = 'cc-live-' + sessionId + '-prompt';
    if (typewriterTimers[timerId]) {
      clearInterval(typewriterTimers[timerId]);
      delete typewriterTimers[timerId];
    }

    // Un-hide static card
    var staticCard = document.querySelector('[data-session-id="' + sessionId + '"]');
    if (staticCard) {
      staticCard.style.display = '';
    }

    // Remove standalone live card
    if (liveContainer) {
      var liveCard = document.getElementById('cc-live-' + sessionId);
      if (liveCard) {
        liveCard.remove();
      }
    }
  }

  function handleMessage(data) {
    setActive(data.active);

    if (!data.active || !data.sessions || data.sessions.length === 0) {
      // Remove all live cards
      for (var id in liveSessions) {
        removeLiveSession(id);
      }
      liveSessions = {};
      recalcAggregates();
      return;
    }

    // Track which session IDs are in this update
    var currentIds = {};
    for (var i = 0; i < data.sessions.length; i++) {
      var sess = data.sessions[i];
      // Skip null sessions (no tokens, no prompts, no summary)
      if (!sess.total_tokens && !sess.user_prompts && !sess.summary) continue;
      currentIds[sess.session_id] = true;
      liveSessions[sess.session_id] = sess;
      renderLiveSession(sess);
    }

    // Remove sessions no longer present
    for (var sid in liveSessions) {
      if (!currentIds[sid]) {
        removeLiveSession(sid);
        delete liveSessions[sid];
      }
    }

    recalcAggregates();
  }

  // Synthesize a full-state message from the activeSessions map
  function synthesizeState() {
    var sessions = [];
    for (var id in activeSessions) {
      sessions.push(activeSessions[id]);
    }
    handleMessage({
      active: sessions.length > 0,
      sessions: sessions
    });
  }

  // Client-side inactivity timeout (90s) — safety net for crashed daemons
  setInterval(function() {
    var now = Date.now();
    var changed = false;
    for (var id in activeSessions) {
      if (now - (lastHeartbeatAt[id] || 0) > 90000) {
        delete activeSessions[id];
        delete lastHeartbeatAt[id];
        changed = true;
      }
    }
    if (changed) {
      synthesizeState();
    }
  }, 30000);

  // Map proto JSON camelCase fields to snake_case used by rendering code
  function mapProtoSession(proto) {
    return {
      session_id: proto.sessionId || '',
      total_tokens: Number(proto.totalTokens) || 0,
      input_tokens: Number(proto.inputTokens) || 0,
      cache_read_input_tokens: Number(proto.cacheReadInputTokens) || 0,
      output_tokens: Number(proto.outputTokens) || 0,
      tool_calls: Number(proto.toolCalls) || 0,
      tool_counts: proto.toolCounts || {},
      user_prompts: Number(proto.userPrompts) || 0,
      active_time_seconds: Number(proto.activeTimeSeconds) || 0,
      last_prompt: proto.lastPrompt || '',
      project: proto.project || '',
      model: proto.model || '',
      summary: proto.summary || '',
      started_at: proto.startedAt || '',
      ended_at: proto.endedAt || '',
      privacy_level: proto.privacyLevel || '',
      sensitive: proto.privacyLevel === 'metrics_only' || proto.privacyLevel === 'private'
    };
  }

  function handleConnectEvent(event) {
    if (event.type === 'heartbeat' && event.session) {
      var sess = mapProtoSession(event.session);
      activeSessions[sess.session_id] = sess;
      lastHeartbeatAt[sess.session_id] = Date.now();
    }

    if (event.type === 'stop' && event.sessionIds) {
      for (var i = 0; i < event.sessionIds.length; i++) {
        delete activeSessions[event.sessionIds[i]];
        delete lastHeartbeatAt[event.sessionIds[i]];
      }
    }

    synthesizeState();
  }

  function handleDisconnect() {
    setActive(false);
    for (var id in liveSessions) {
      removeLiveSession(id);
    }
    liveSessions = {};
    activeSessions = {};
    lastHeartbeatAt = {};
    recalcAggregates();
    setTimeout(connect, 5000);
  }

  // Parse Connect protocol binary envelopes from a streaming response.
  // Each envelope: [flags: 1 byte][length: 4 bytes big-endian][payload: N bytes]
  // flags 0x00 = data, 0x02 = end-of-stream trailers
  function connect() {
    var url = apiUrl + '/sessions.v1.SessionService/WatchSessions';

    fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/connect+json',
        'Connect-Protocol-Version': '1'
      },
      body: JSON.stringify({ scope: 'public_user', userId: userId })
    }).then(function(response) {
      if (!response.ok) {
        throw new Error('HTTP ' + response.status);
      }
      var reader = response.body.getReader();
      var buffer = new Uint8Array(0);

      function read() {
        reader.read().then(function(result) {
          if (result.done) {
            handleDisconnect();
            return;
          }

          // Append chunk to buffer
          var newBuf = new Uint8Array(buffer.length + result.value.length);
          newBuf.set(buffer);
          newBuf.set(result.value, buffer.length);
          buffer = newBuf;

          // Parse complete envelopes
          while (buffer.length >= 5) {
            var flags = buffer[0];
            var length = (buffer[1] << 24) | (buffer[2] << 16) | (buffer[3] << 8) | buffer[4];

            if (buffer.length < 5 + length) break;

            var payload = buffer.slice(5, 5 + length);
            buffer = buffer.slice(5 + length);

            // flags 0x02 = end-of-stream (trailers); skip
            if (flags & 0x02) continue;

            var text = new TextDecoder().decode(payload);
            var event = JSON.parse(text);
            handleConnectEvent(event);
          }

          read();
        }).catch(function() {
          handleDisconnect();
        });
      }

      read();
    }).catch(function() {
      handleDisconnect();
    });
  }

  connect();

  if (typeof globalThis.__TEST__ !== 'undefined') {
    globalThis.__liveStatus = { formatTokens, formatTime, escapeHTML };
  }
})();
