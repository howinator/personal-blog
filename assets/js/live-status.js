(function() {
  var dots = document.querySelectorAll('.cc-status-dot');
  if (!dots.length) return;

  var wsUrl = (location.protocol === 'https:' ? 'wss://' : 'ws://') + location.host + '/ws/live';

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

  // Collect static session IDs so we know which live sessions are already in the base totals
  var staticSessionIds = {};
  document.querySelectorAll('[data-session-id]').forEach(function(el) {
    staticSessionIds[el.getAttribute('data-session-id')] = true;
  });

  var liveContainer = document.getElementById('cc-live-sessions');
  var liveSessions = {};
  var typewriterTimers = {};
  var lastPromptSeen = {};

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

    for (var id in liveSessions) {
      // Skip sessions already counted in the static base totals
      if (staticSessionIds[id]) continue;
      var sess = liveSessions[id];
      deltaCount++;
      deltaTokens += sess.total_tokens || 0;
      deltaTime += sess.active_time_seconds || 0;
      deltaTools += sess.tool_calls || 0;
    }

    updateStatDisplay(statEls['sessions'], String(baseTotals['sessions'] + deltaCount));
    updateStatDisplay(statEls['tokens'], formatTokens(baseTotals['tokens'] + deltaTokens));
    updateStatDisplay(statEls['active-time'], formatTime(baseTotals['active-time'] + deltaTime));
    updateStatDisplay(statEls['tool-calls'], String(baseTotals['tool-calls'] + deltaTools));
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
      if (!promptContainer) {
        promptContainer = document.createElement('div');
        promptContainer.className = 'cc-live-prompt cc-live-injected';
        promptContainer.innerHTML = '<span style="font-family:var(--font-mono);font-size:0.8rem;color:var(--muted)">Latest Prompt</span>';
        var promptEl = document.createElement('div');
        promptEl.className = 'cc-typewriter';
        promptEl.id = promptId;
        promptContainer.appendChild(promptEl);
        details.appendChild(promptContainer);
      }

      var promptEl = document.getElementById(promptId);
      if (promptEl) {
        if (lastPromptSeen[session.session_id] !== session.last_prompt) {
          lastPromptSeen[session.session_id] = session.last_prompt;
          typewriterAnimate(promptEl, session.last_prompt);
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
      '<span class="cc-session-summary">' + escapeHTML(session.project || 'unknown') + '</span>' +
      '<span class="cc-session-tokens">' + tokenDisplay + ' tokens</span>';

    // Build details
    var detailsHTML =
      '<table>' +
      '<tr><td>Model</td><td>' + escapeHTML(session.model || '') + '</td></tr>' +
      '<tr><td>User Prompts</td><td>' + (session.user_prompts || 0) + '</td></tr>' +
      '<tr><td>Tool Calls</td><td>' + (session.tool_calls || 0) + '</td></tr>' +
      '<tr><td>Total Tokens</td><td>' + tokenDisplay + '</td></tr>' +
      '<tr><td>Active Time</td><td>' + timeDisplay + '</td></tr>' +
      '</table>' +
      '<div style="margin-top:0.5rem"><span style="font-family:var(--font-mono);font-size:0.8rem;color:var(--muted)">Latest Prompt</span>' +
      '<div class="cc-typewriter" id="' + cardId + '-prompt"></div></div>';

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
    detailsDiv.innerHTML = detailsHTML;

    // Typewriter for latest prompt
    var promptEl = document.getElementById(cardId + '-prompt');
    if (promptEl && session.last_prompt) {
      if (lastPromptSeen[session.session_id] !== session.last_prompt) {
        lastPromptSeen[session.session_id] = session.last_prompt;
        typewriterAnimate(promptEl, session.last_prompt);
      }
    }
  }

  function renderLiveSession(session) {
    // Check if this session already has a static card
    var staticCard = document.querySelector('[data-session-id="' + session.session_id + '"]');
    if (staticCard) {
      enhanceStaticCard(staticCard, session);
    } else {
      createLiveCard(session);
    }
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
    delete lastPromptSeen[sessionId];

    // Clean up typewriter timer
    var timerId = 'cc-live-' + sessionId + '-prompt';
    if (typewriterTimers[timerId]) {
      clearInterval(typewriterTimers[timerId]);
      delete typewriterTimers[timerId];
    }

    // Revert enhanced static card
    var staticCard = document.querySelector('[data-session-id="' + sessionId + '"]');
    if (staticCard && staticCard.classList.contains('cc-session-live')) {
      revertStaticCard(staticCard);
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

  function connect() {
    var ws = new WebSocket(wsUrl);
    ws.onmessage = function(e) {
      var data = JSON.parse(e.data);
      handleMessage(data);
    };
    ws.onclose = function() {
      setActive(false);
      // Clear live sessions on disconnect
      for (var id in liveSessions) {
        removeLiveSession(id);
      }
      liveSessions = {};
      recalcAggregates();
      setTimeout(connect, 5000);
    };
  }

  connect();
})();
