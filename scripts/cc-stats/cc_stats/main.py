"""Process Claude Code session transcripts into blog stats."""

import json
import os
import sys
import tempfile
import urllib.request
from datetime import datetime, timezone
from pathlib import Path

BLOG_ROOT = Path(os.environ.get("CC_STATS_BLOG_ROOT", Path.home() / "projects" / "personal-blog"))
DATA_FILE = BLOG_ROOT / "data" / "cc_sessions.json"
MAX_IDLE_SECONDS = 300  # 5 minutes — cap gaps between entries


def format_tokens(n: int) -> str:
    if n >= 1_000_000:
        return f"{n / 1_000_000:.1f}M"
    if n >= 1_000:
        return f"{n / 1_000:.1f}k"
    return str(n)


def format_time(seconds: int) -> str:
    if seconds < 60:
        return f"{seconds}s"
    minutes, secs = divmod(seconds, 60)
    if minutes < 60:
        return f"{minutes}m {secs}s" if secs else f"{minutes}m"
    hours, mins = divmod(minutes, 60)
    parts = [f"{hours}h"]
    if mins:
        parts.append(f"{mins}m")
    return " ".join(parts)


def parse_transcript(path: str) -> list[dict]:
    entries = []
    with open(path) as f:
        for line in f:
            line = line.strip()
            if line:
                entries.append(json.loads(line))
    return entries


def compute_stats(entries: list[dict]) -> dict | None:
    user_prompts = 0
    tool_calls = 0
    input_tokens = 0
    output_tokens = 0
    user_texts = []
    timestamps = []

    for entry in entries:
        entry_type = entry.get("type")
        msg = entry.get("message", {})
        if not isinstance(msg, dict):
            continue

        ts = entry.get("timestamp")
        if ts:
            timestamps.append(ts)

        content = msg.get("content", [])

        if entry_type == "user":
            # Count user prompts: entries with at least one text block (string content)
            # or text-type dict block. Exclude bare tool_result entries.
            if isinstance(content, str) and content.strip():
                user_prompts += 1
                user_texts.append(content.strip())
            elif isinstance(content, list):
                has_text = any(
                    isinstance(c, dict) and c.get("type") == "text" and c.get("text", "").strip()
                    for c in content
                )
                if has_text:
                    user_prompts += 1
                    for c in content:
                        if isinstance(c, dict) and c.get("type") == "text":
                            user_texts.append(c["text"].strip())

        elif entry_type == "assistant":
            # Count tool_use blocks
            if isinstance(content, list):
                for c in content:
                    if isinstance(c, dict) and c.get("type") == "tool_use":
                        tool_calls += 1

            # Sum tokens from usage
            usage = msg.get("usage", {})
            if usage:
                input_tokens += usage.get("input_tokens", 0)
                input_tokens += usage.get("cache_creation_input_tokens", 0)
                input_tokens += usage.get("cache_read_input_tokens", 0)
                output_tokens += usage.get("output_tokens", 0)

    if user_prompts == 0:
        return None

    total_tokens = input_tokens + output_tokens
    if total_tokens == 0:
        return None

    # Compute active time
    active_seconds = 0
    parsed_times = []
    for ts in timestamps:
        try:
            # Handle ISO 8601 with Z suffix
            t = ts.replace("Z", "+00:00")
            parsed_times.append(datetime.fromisoformat(t))
        except (ValueError, TypeError):
            continue

    for i in range(1, len(parsed_times)):
        gap = (parsed_times[i] - parsed_times[i - 1]).total_seconds()
        active_seconds += int(min(max(gap, 0), MAX_IDLE_SECONDS))

    # Get session metadata from first entry with these fields
    session_id = ""
    cwd = ""
    version = ""
    date_str = ""
    for entry in entries:
        if not session_id and entry.get("sessionId"):
            session_id = entry["sessionId"]
        if not cwd and entry.get("cwd"):
            cwd = entry["cwd"]
        if not version and entry.get("version"):
            version = entry["version"]
        if not date_str and entry.get("timestamp"):
            date_str = entry["timestamp"]

    project = Path(cwd).name if cwd else "unknown"

    # Parse date for display
    date_display = ""
    if date_str:
        try:
            t = date_str.replace("Z", "+00:00")
            dt = datetime.fromisoformat(t)
            date_display = dt.strftime("%b %d, %Y")
        except (ValueError, TypeError):
            date_display = date_str[:10]

    total_tokens = input_tokens + output_tokens

    return {
        "session_id": session_id,
        "date": date_str,
        "date_display": date_display,
        "summary": "",  # filled in later
        "project": project,
        "cwd": cwd,
        "num_user_prompts": user_prompts,
        "num_tool_calls": tool_calls,
        "total_input_tokens": input_tokens,
        "total_output_tokens": output_tokens,
        "total_tokens": total_tokens,
        "total_tokens_display": format_tokens(total_tokens),
        "active_time_seconds": active_seconds,
        "active_time_display": format_time(active_seconds),
        "cc_version": version,
        "_user_texts": user_texts,  # temporary, for summary generation
    }


def generate_summary(user_texts: list[str]) -> str:
    """Generate a 1-sentence summary via the Anthropic API."""
    if not user_texts:
        return "Empty session"

    api_key = os.environ.get("ANTHROPIC_API_KEY", "")
    if not api_key:
        return _fallback_summary(user_texts)

    # Build context from user prompts (truncate to keep prompt reasonable)
    cleaned = _clean_user_texts(user_texts)
    if not cleaned:
        return "Short session"
    context = "\n---\n".join(cleaned[:20])
    if len(context) > 4000:
        context = context[:4000] + "..."

    prompt = (
        "Below are the user prompts from a Claude Code coding session. "
        "Write a single sentence (max 120 characters) summarizing what the user worked on. "
        "Be specific and concise. Do not start with 'The user'. "
        "Just output the summary sentence, nothing else.\n\n"
        f"{context}"
    )

    body = json.dumps({
        "model": "claude-haiku-4-5-20251001",
        "max_tokens": 100,
        "messages": [{"role": "user", "content": prompt}],
    }).encode()

    req = urllib.request.Request(
        "https://api.anthropic.com/v1/messages",
        data=body,
        headers={
            "x-api-key": api_key,
            "anthropic-version": "2023-06-01",
            "content-type": "application/json",
        },
    )

    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            data = json.load(resp)
            text = data.get("content", [{}])[0].get("text", "").strip()
            if text:
                if len(text) > 150:
                    text = text[:147] + "..."
                return text
    except Exception:
        pass

    return _fallback_summary(user_texts)


def _clean_user_texts(user_texts: list[str]) -> list[str]:
    """Filter out system-injected text from user prompts."""
    skip_prefixes = ("[Request interrupted", "<local-command-caveat>", "<system-reminder>")
    return [t for t in user_texts if not any(t.startswith(p) for p in skip_prefixes)]


def _fallback_summary(user_texts: list[str]) -> str:
    """Fallback: first real user prompt, truncated."""
    cleaned = _clean_user_texts(user_texts)
    if not cleaned:
        return "Short session"
    fallback = cleaned[0]
    if len(fallback) > 120:
        fallback = fallback[:117] + "..."
    return fallback


def load_data() -> dict:
    if DATA_FILE.exists():
        try:
            with open(DATA_FILE) as f:
                return json.load(f)
        except (json.JSONDecodeError, OSError):
            pass
    return {"sessions": [], "totals": _empty_totals()}


def _empty_totals() -> dict:
    return {
        "session_count": 0,
        "total_tokens": 0,
        "total_tokens_display": "0",
        "total_tool_calls": 0,
        "total_active_time_seconds": 0,
        "total_active_time_display": "0s",
    }


def recompute_totals(sessions: list[dict]) -> dict:
    total_tokens = sum(s["total_tokens"] for s in sessions)
    total_tool_calls = sum(s["num_tool_calls"] for s in sessions)
    total_active = sum(s["active_time_seconds"] for s in sessions)
    return {
        "session_count": len(sessions),
        "total_tokens": total_tokens,
        "total_tokens_display": format_tokens(total_tokens),
        "total_tool_calls": total_tool_calls,
        "total_active_time_seconds": total_active,
        "total_active_time_display": format_time(total_active),
    }


def save_data(data: dict) -> None:
    DATA_FILE.parent.mkdir(parents=True, exist_ok=True)
    # Atomic write: temp file + rename
    fd, tmp_path = tempfile.mkstemp(dir=DATA_FILE.parent, suffix=".json")
    try:
        with os.fdopen(fd, "w") as f:
            json.dump(data, f, indent=2)
            f.write("\n")
        os.replace(tmp_path, DATA_FILE)
    except Exception:
        try:
            os.unlink(tmp_path)
        except OSError:
            pass
        raise


def main() -> None:
    try:
        _main()
    except Exception as e:
        # Never block CC — always exit 0
        print(f"cc-stats error: {e}", file=sys.stderr)
        sys.exit(0)


def _main() -> None:
    # Read hook payload from stdin
    payload = json.load(sys.stdin)
    transcript_path = payload.get("transcript_path")
    if not transcript_path or not Path(transcript_path).exists():
        return

    entries = parse_transcript(transcript_path)
    if not entries:
        return

    stats = compute_stats(entries)
    if stats is None:
        return

    # Generate LLM summary
    user_texts = stats.pop("_user_texts")
    stats["summary"] = generate_summary(user_texts)

    # Load existing data, upsert session, recompute totals
    data = load_data()
    sessions = data.get("sessions", [])

    # Upsert: replace if session_id already exists
    existing_idx = None
    for i, s in enumerate(sessions):
        if s.get("session_id") == stats["session_id"]:
            existing_idx = i
            break

    if existing_idx is not None:
        sessions[existing_idx] = stats
    else:
        sessions.append(stats)

    # Sort by date descending (newest first)
    sessions.sort(key=lambda s: s.get("date", ""), reverse=True)

    data["sessions"] = sessions
    data["totals"] = recompute_totals(sessions)

    save_data(data)


if __name__ == "__main__":
    main()
