(function() {
  var dots = document.querySelectorAll('.cc-status-dot');
  if (!dots.length) return;

  var wsUrl = (location.protocol === 'https:' ? 'wss://' : 'ws://') + location.host + '/ws/live';

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

  function connect() {
    var ws = new WebSocket(wsUrl);
    ws.onmessage = function(e) {
      var data = JSON.parse(e.data);
      setActive(data.active);
    };
    ws.onclose = function() {
      setActive(false);
      setTimeout(connect, 5000);
    };
  }

  connect();
})();
