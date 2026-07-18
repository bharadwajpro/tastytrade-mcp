#!/usr/bin/env python3
"""Exercise the MCP OAuth + Streamable HTTP path the way Hermes does.

Flow:
  1. OAuth authorization-code + PKCE (print URL, paste callback)
  2. Token exchange via /oauth/token
  3. MCP initialize + tools/list over Streamable HTTP (MCP Python SDK)
  4. Probe GET /mcp (expect 405 after WithDisableStreaming — Hermes OAuth
     otherwise deadlocks holding its auth lock on a long-lived SSE GET)

Usage (Hermes venv already has the MCP SDK):

  ~/.hermes/hermes-agent/venv/bin/python scripts/test_oauth_mcp.py

  # or, if mcp+httpx+anyio are on PATH:
  python3 scripts/test_oauth_mcp.py

Reads SERVER_URL / OAUTH_CLIENT_ID / OAUTH_CLIENT_SECRET from .env in the
repo root (or the current working directory). Never prints secrets.
"""

from __future__ import annotations

import argparse
import asyncio
import os
import sys
import time
import webbrowser
from pathlib import Path
from typing import Any
from urllib.parse import parse_qs, urlparse


def _repo_root() -> Path:
    return Path(__file__).resolve().parent.parent


def load_dotenv(path: Path) -> dict[str, str]:
    env: dict[str, str] = {}
    if not path.is_file():
        return env
    for line in path.read_text().splitlines():
        line = line.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        k, v = line.split("=", 1)
        env[k.strip()] = v.strip().strip('"').strip("'")
    return env


def require_deps() -> None:
    missing = []
    for mod in ("httpx", "mcp", "anyio"):
        try:
            __import__(mod)
        except ImportError:
            missing.append(mod)
    if missing:
        print(
            "Missing packages: "
            + ", ".join(missing)
            + "\n\nInstall/run with the Hermes venv:\n"
            "  ~/.hermes/hermes-agent/venv/bin/python scripts/test_oauth_mcp.py",
            file=sys.stderr,
        )
        sys.exit(2)


class MemoryTokenStorage:
    """In-memory token/client storage for one test run (same role as HermesTokenStorage)."""

    def __init__(self) -> None:
        self._tokens: Any = None
        self._client_info: Any = None

    async def get_tokens(self) -> Any:
        return self._tokens

    async def set_tokens(self, tokens: Any) -> None:
        self._tokens = tokens
        print(
            f"  tokens saved (access_len={len(tokens.access_token or '')}, "
            f"refresh_len={len(tokens.refresh_token or '')}, "
            f"expires_in={tokens.expires_in})",
            flush=True,
        )

    async def get_client_info(self) -> Any:
        return self._client_info

    async def set_client_info(self, info: Any) -> None:
        self._client_info = info


def _parse_callback(raw: str) -> tuple[str, str | None]:
    raw = raw.strip()
    if not raw.startswith("http"):
        # allow pasting "?code=...&state=..." only
        if raw.startswith("?"):
            raw = "http://127.0.0.1/callback" + raw
        else:
            raw = "http://127.0.0.1/callback?" + raw
    q = parse_qs(urlparse(raw).query)
    if q.get("error"):
        raise RuntimeError(f"OAuth error in callback: {q['error'][0]}")
    code = (q.get("code") or [None])[0]
    state = (q.get("state") or [None])[0]
    if not code:
        raise RuntimeError("callback URL missing code=")
    return code, state


async def run(server_url: str, client_id: str, client_secret: str, *, open_browser: bool) -> int:
    import httpx
    from mcp.client.auth import OAuthClientProvider
    from mcp.client.auth.oauth2 import OAuthClientMetadata
    from mcp.client.session import ClientSession
    from mcp.client.streamable_http import streamable_http_client
    from pydantic import AnyUrl

    base = server_url.rstrip("/")
    mcp_url = base if base.endswith("/mcp") else base + "/mcp"
    # redirect_uri is only used for the authorize→callback binding; we paste
    # the URL manually (same headless path Hermes documents).
    redirect_uri = "http://127.0.0.1:9/callback"

    pasted: dict[str, str | None] = {"code": None, "state": None}

    async def redirect_handler(authorization_url: str) -> None:
        print("\n  Open this URL in your browser:\n", flush=True)
        print(f"    {authorization_url}\n", flush=True)
        if open_browser:
            try:
                webbrowser.open(authorization_url)
            except Exception:
                pass
        print(
            "  After authorize, copy the full redirect URL from the address bar\n"
            "  (it will fail to connect to 127.0.0.1 — that's expected) and paste it here.\n",
            flush=True,
        )
        raw = await asyncio.to_thread(input, "  Paste redirect URL: ")
        code, state = _parse_callback(raw)
        pasted["code"], pasted["state"] = code, state
        print("  callback accepted", flush=True)

    async def callback_handler() -> tuple[str, str | None]:
        if not pasted["code"]:
            raise RuntimeError("no authorization code captured")
        return pasted["code"], pasted["state"]

    # Pre-register the fixed client (our server has no dynamic registration).
    from mcp.shared.auth import OAuthClientInformationFull

    storage = MemoryTokenStorage()
    await storage.set_client_info(
        OAuthClientInformationFull(
            client_id=client_id,
            client_secret=client_secret,
            redirect_uris=[AnyUrl(redirect_uri)],
            grant_types=["authorization_code", "refresh_token"],
            response_types=["code"],
            token_endpoint_auth_method="client_secret_post",
        )
    )

    metadata = OAuthClientMetadata(
        client_name="tastytrade-mcp-oauth-test",
        redirect_uris=[AnyUrl(redirect_uri)],
        grant_types=["authorization_code", "refresh_token"],
        response_types=["code"],
        scope="mcp",
        token_endpoint_auth_method="client_secret_post",
    )

    auth = OAuthClientProvider(
        server_url=mcp_url,
        client_metadata=metadata,
        storage=storage,
        redirect_handler=redirect_handler,
        callback_handler=callback_handler,
        timeout=300.0,
    )

    print(f"MCP URL:  {mcp_url}", flush=True)
    print(f"client_id set: {bool(client_id)} (len={len(client_id)})", flush=True)
    t0 = time.time()

    timeout = httpx.Timeout(30.0, read=60.0)
    async with httpx.AsyncClient(follow_redirects=True, timeout=timeout, auth=auth) as client:
        # Probe GET before session — after the fix this should be 405 quickly.
        # Use a throwaway bearer-less GET first after tokens exist; for now we
        # probe inside the session once we have a token.

        async with streamable_http_client(mcp_url, http_client=client) as (read, write, get_sid):
            async with ClientSession(read, write) as session:
                print("\n  initialize...", flush=True)
                init = await asyncio.wait_for(session.initialize(), timeout=30)
                print(
                    f"  initialize OK  protocol={init.protocolVersion}  "
                    f"session={get_sid()}  ({time.time() - t0:.2f}s)",
                    flush=True,
                )

                print("  tools/list...", flush=True)
                tools = await asyncio.wait_for(session.list_tools(), timeout=30)
                names = sorted(t.name for t in tools.tools)
                print(
                    f"  tools/list OK  count={len(names)}  ({time.time() - t0:.2f}s)",
                    flush=True,
                )
                for n in names:
                    print(f"    - {n}")

        # After tokens exist, probe GET the way Hermes opens the SSE stream.
        tokens = await storage.get_tokens()
        if tokens and tokens.access_token:
            print("\n  probing GET /mcp (Hermes SSE listen)...", flush=True)
            gt0 = time.time()
            try:
                r = await asyncio.wait_for(
                    client.get(
                        mcp_url,
                        headers={
                            "Authorization": f"Bearer {tokens.access_token}",
                            "Accept": "text/event-stream",
                        },
                    ),
                    timeout=10,
                )
                print(
                    f"  GET status={r.status_code}  "
                    f"body={r.text[:80]!r}  ({time.time() - gt0:.2f}s)",
                    flush=True,
                )
                if r.status_code == 405:
                    print(
                        "  GET=405 as expected (streaming disabled; avoids OAuth SSE deadlock)",
                        flush=True,
                    )
                elif r.status_code == 200:
                    print(
                        "  WARNING: GET=200 means SSE streaming is enabled; "
                        "Hermes OAuth may deadlock on tools/list",
                        flush=True,
                    )
            except Exception as exc:
                print(f"  GET probe failed: {type(exc).__name__}: {exc}", flush=True)

    print(f"\nSUCCESS in {time.time() - t0:.2f}s", flush=True)
    return 0


def main() -> int:
    require_deps()
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument(
        "--env-file",
        type=Path,
        default=_repo_root() / ".env",
        help="path to .env (default: repo .env)",
    )
    ap.add_argument(
        "--server-url",
        default="",
        help="override SERVER_URL (public base or .../mcp)",
    )
    ap.add_argument(
        "--open-browser",
        action="store_true",
        help="try to open the authorize URL automatically",
    )
    args = ap.parse_args()

    file_env = load_dotenv(args.env_file)
    # process env wins over file
    def get(key: str) -> str:
        return os.environ.get(key) or file_env.get(key) or ""

    server_url = args.server_url or get("SERVER_URL")
    client_id = get("OAUTH_CLIENT_ID")
    client_secret = get("OAUTH_CLIENT_SECRET")
    if not server_url or not client_id or not client_secret:
        print(
            "Need SERVER_URL, OAUTH_CLIENT_ID, OAUTH_CLIENT_SECRET "
            f"(from {args.env_file} or environment)",
            file=sys.stderr,
        )
        return 2

    try:
        return asyncio.run(
            run(server_url, client_id, client_secret, open_browser=args.open_browser)
        )
    except KeyboardInterrupt:
        print("\naborted", file=sys.stderr)
        return 130
    except Exception as exc:
        print(f"\nFAILED: {type(exc).__name__}: {exc}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
