#!/usr/bin/env python3
"""Generate a local WAV file through the running Word Agent service."""

from __future__ import annotations

import argparse
import json
import re
import sys
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path
from typing import Any

DEFAULT_WORD_AGENT_URL = "http://127.0.0.1:6017"
DEFAULT_FORMAT = "wav"
DEFAULT_STYLE = (
    "Clear American English pronunciation for vocabulary learning. "
    "Speak only the word, naturally and slowly."
)


def safe_filename(text: str) -> str:
    name = re.sub(r"[^A-Za-z0-9._-]+", "_", text.strip()).strip("._-")
    return name or "tts_audio"


def generate_tts_audio(
    *,
    word: str,
    word_agent_url: str,
    style: str | None,
    file_name: str,
    timeout: int,
) -> bytes:
    base_url = word_agent_url.strip().rstrip("/")
    if not base_url:
        raise ValueError("Word Agent URL cannot be empty")

    payload: dict[str, Any] = {
        "text": word,
        "style": style,
        "fileName": file_name,
        "format": DEFAULT_FORMAT,
    }
    request = urllib.request.Request(
        f"{base_url}/v1/tts/generate",
        data=json.dumps(payload, ensure_ascii=False).encode("utf-8"),
        headers={"Content-Type": "application/json"},
        method="POST",
    )

    try:
        with urllib.request.urlopen(request, timeout=timeout) as response:
            response_data = json.loads(response.read().decode("utf-8"))
    except urllib.error.HTTPError as exc:
        detail = exc.read().decode("utf-8", errors="replace")
        raise RuntimeError(f"Word Agent returned HTTP {exc.code}: {detail}") from exc
    except urllib.error.URLError as exc:
        raise RuntimeError(f"Failed to call Word Agent: {exc}") from exc
    except (UnicodeDecodeError, json.JSONDecodeError) as exc:
        raise RuntimeError("Word Agent returned invalid JSON") from exc

    download_url = str(response_data.get("downloadUrl") or "").strip()
    if not download_url:
        raise RuntimeError("Word Agent response is missing downloadUrl")
    absolute_download_url = urllib.parse.urljoin(f"{base_url}/", download_url.lstrip("/"))
    download_request = urllib.request.Request(absolute_download_url, method="GET")
    try:
        with urllib.request.urlopen(download_request, timeout=timeout) as response:
            return response.read()
    except urllib.error.HTTPError as exc:
        detail = exc.read().decode("utf-8", errors="replace")
        raise RuntimeError(f"Word Agent audio download returned HTTP {exc.code}: {detail}") from exc
    except urllib.error.URLError as exc:
        raise RuntimeError(f"Failed to download Word Agent audio: {exc}") from exc


def parse_args(argv: list[str] | None = None) -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Generate a local TTS audio file through Word Agent."
    )
    parser.add_argument("word", help='English word or short phrase, for example "apple".')
    parser.add_argument(
        "-o",
        "--output",
        type=Path,
        help="Output WAV path. Defaults to tts_audio/<word>.wav.",
    )
    parser.add_argument(
        "--word-agent-url",
        default=DEFAULT_WORD_AGENT_URL,
        help=f"Word Agent service URL. Default: {DEFAULT_WORD_AGENT_URL}.",
    )
    parser.add_argument(
        "--style",
        default=DEFAULT_STYLE,
        help="Optional style instruction. Use an empty string to omit it.",
    )
    parser.add_argument("--timeout", type=int, default=60, help="HTTP timeout in seconds.")
    return parser.parse_args(argv)


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv)
    word = args.word.strip()
    if not word:
        print("Word cannot be empty.", file=sys.stderr)
        return 2

    output_path = args.output or Path("tts_audio") / f"{safe_filename(word)}.wav"
    audio_bytes = generate_tts_audio(
        word=word,
        word_agent_url=args.word_agent_url,
        style=args.style.strip() or None,
        file_name=output_path.name,
        timeout=args.timeout,
    )
    output_path.parent.mkdir(parents=True, exist_ok=True)
    output_path.write_bytes(audio_bytes)
    print(f"Saved {len(audio_bytes)} bytes to {output_path}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
