#!/usr/bin/env python3
"""
EnterpriseRag local MCP demo server.

A minimal, runnable external MCP service for testing client connectivity from
EnterpriseRag (Settings -> MCP Services). By default it listens over Streamable HTTP
on http://127.0.0.1:8010/mcp

To start it:
  export MCP_SERVER_AUTH_TOKEN=enterpriserag-demo-token
  python server.py

EnterpriseRag configuration:
  Transport: HTTP Streamable
  URL: http://127.0.0.1:8010/mcp
  Authentication: Bearer, with the token matching MCP_SERVER_AUTH_TOKEN
"""

from __future__ import annotations

import argparse
import asyncio
import logging
import os
import secrets
import sys
from datetime import datetime, timezone
from typing import Any

from mcp.server import MCPServer

logging.basicConfig(level=logging.INFO, format="%(levelname)s %(message)s")
logger = logging.getLogger("mcp-demo")

mcp = MCPServer("enterpriserag-mcp-demo", version="0.1.0")

# Demo corpus, so that an agent's answers can be compared with the knowledge base.
DEMO_POLICIES: dict[str, str] = {
    "warranty": "The Smart Home Hub Pro carries a 24-month warranty on the unit and 12 months on battery accessories; damage from tampering or liquid ingress is not covered.",
    "offline_voice": "If voice recognition runs in the cloud, only the app and the local touchscreen work once the internet connection is lost; installing the local voice pack keeps basic commands available.",
    "device_limit": "A personal account can bind at most 3 hubs; enterprise accounts are licensed per contract, with a default of 50.",
    "travel_hotel_tier1": "Hotel reimbursement in tier-1 cities (Mumbai, Delhi, Bengaluru, Chennai) is capped at INR 6000 per night, inclusive of tax.",
    "travel_meal": "Meals are not reimbursed separately while travelling; the tier-1 city travel allowance is INR 1500 per day.",
    "poc_owner": "For the after-sales knowledge base POC, the engineering lead is Arjun Mehta, the product contact is Leena Rao, and the QA lead is Rahul Bose.",
    "poc_deadline": "The after-sales knowledge base POC must be demonstrated on the internal network before 2024-03-01.",
    "matter_cert": "Firmware 3.5 is scheduled for a staged rollout by the end of March 2024, completing Matter 1.2 certification.",
}

DEMO_CONTACTS: list[dict[str, str]] = [
    {"name": "Vikram Shah", "role": "Product Director", "department": "Product"},
    {"name": "Arjun Mehta", "role": "Knowledge Base and AI Module Lead", "department": "Engineering"},
    {"name": "Leena Rao", "role": "Product Operations", "department": "Product"},
    {"name": "Sneha Kulkarni", "role": "Interaction Design Lead", "department": "Design"},
    {"name": "Rahul Bose", "role": "QA Manager", "department": "QA"},
]


def network_transport_auth_token() -> str:
    return os.getenv("MCP_SERVER_AUTH_TOKEN", "").strip()


def require_network_transport_auth(transport: str) -> str:
    token = network_transport_auth_token()
    if transport in ("sse", "http") and not token:
        logger.error(
            "MCP_SERVER_AUTH_TOKEN is required for %s transport. "
            "Example: export MCP_SERVER_AUTH_TOKEN=enterpriserag-demo-token",
            transport,
        )
        sys.exit(1)
    return token


class MCPAuthMiddleware:
    """Bearer authentication middleware for the SSE and HTTP transports."""

    def __init__(self, app, token: str):
        self.app = app
        self.token = token

    async def __call__(self, scope, receive, send):
        if scope.get("type") != "http":
            await self.app(scope, receive, send)
            return

        headers = {
            k.decode("latin-1").lower(): v.decode("latin-1")
            for k, v in scope.get("headers", [])
        }
        provided = ""
        auth = headers.get("authorization", "")
        if auth.lower().startswith("bearer "):
            provided = auth[7:].strip()
        elif "x-mcp-auth-token" in headers:
            provided = headers["x-mcp-auth-token"]

        if not provided or not secrets.compare_digest(provided, self.token):
            body = b'{"error":"unauthorized"}'
            await send(
                {
                    "type": "http.response.start",
                    "status": 401,
                    "headers": [[b"content-type", b"application/json"]],
                }
            )
            await send({"type": "http.response.body", "body": body})
            return

        await self.app(scope, receive, send)


@mcp.tool()
def echo(message: str) -> dict[str, Any]:
    """Echo a message back, to verify MCP connectivity."""
    return {"echo": message}


@mcp.tool()
def add(a: float, b: float) -> dict[str, Any]:
    """Return the sum of two numbers."""
    return {"a": a, "b": b, "sum": a + b}


@mcp.tool()
def server_time() -> dict[str, str]:
    """Return the current UTC time on the MCP demo server."""
    now = datetime.now(timezone.utc)
    return {
        "iso": now.isoformat(),
        "unix": str(int(now.timestamp())),
    }


@mcp.tool()
def lookup_policy(topic: str) -> dict[str, Any]:
    """Look up demo policy or project information.

    Args:
        topic: One of warranty, offline_voice, device_limit,
            travel_hotel_tier1, travel_meal, poc_owner, poc_deadline or
            matter_cert. A keyword such as "warranty", "reimbursement" or
            "POC" also works.

    Returns:
        The matching policy text, or the list of available topics when the
        query cannot be resolved.
    """
    key = topic.strip().lower().replace(" ", "_")
    aliases = {
        "warranty": "warranty",
        "guarantee": "warranty",
        "offline": "offline_voice",
        "voice": "offline_voice",
        "device limit": "device_limit",
        "hotel": "travel_hotel_tier1",
        "reimbursement": "travel_hotel_tier1",
        "meal": "travel_meal",
        "allowance": "travel_meal",
        "owner": "poc_owner",
        "lead": "poc_owner",
        "poc": "poc_owner",
        "deadline": "poc_deadline",
        "matter": "matter_cert",
        "certification": "matter_cert",
    }
    for alias, mapped in aliases.items():
        if alias in topic:
            key = mapped
            break

    if key in DEMO_POLICIES:
        return {"topic": key, "answer": DEMO_POLICIES[key], "source": "mcp-demo/static"}

    matches = {
        k: v
        for k, v in DEMO_POLICIES.items()
        if key in k or any(ch in k for ch in key if len(key) >= 2)
    }
    if len(matches) == 1:
        only_key = next(iter(matches))
        return {"topic": only_key, "answer": matches[only_key], "source": "mcp-demo/static"}

    return {
        "topic": topic,
        "available_topics": sorted(DEMO_POLICIES.keys()),
        "hint": "Pass topic as one of the keys above, or a keyword such as warranty, reimbursement or POC.",
    }


@mcp.tool()
def list_team_contacts(department: str = "") -> dict[str, Any]:
    """List the demo project team members.

    Args:
        department: Optional department filter (Product / Engineering /
            Design / QA). An empty string returns everyone.
    """
    rows = DEMO_CONTACTS
    if department.strip():
        needle = department.strip()
        rows = [c for c in rows if needle in c["department"]]
    return {"count": len(rows), "contacts": rows}


@mcp.tool()
def send_demo_alert(channel: str, message: str) -> dict[str, Any]:
    """Simulate sending a notification to an external channel.

    Nothing is actually sent; this is for demonstration only. It is useful for
    testing manual approval of MCP tools inside EnterpriseRag, so marking this tool
    as requiring approval is recommended.
    """
    return {
        "ok": True,
        "simulated": True,
        "channel": channel,
        "message": message,
        "sent_at": datetime.now(timezone.utc).isoformat(),
    }


async def run_http(host: str, port: int) -> None:
    auth_token = require_network_transport_auth("http")
    try:
        import uvicorn
    except ImportError as e:
        raise ImportError("HTTP transport requires: pip install starlette uvicorn") from e

    starlette_app = MCPAuthMiddleware(
        mcp.streamable_http_app(host=host, stateless_http=True),
        auth_token,
    )
    logger.info("Streamable HTTP MCP demo listening on http://%s:%d/mcp", host, port)
    config = uvicorn.Config(starlette_app, host=host, port=port, log_level="info")
    server = uvicorn.Server(config)
    await server.serve()


async def run_sse(host: str, port: int) -> None:
    auth_token = require_network_transport_auth("sse")
    try:
        import uvicorn
    except ImportError as e:
        raise ImportError("SSE transport requires: pip install starlette uvicorn") from e

    starlette_app = MCPAuthMiddleware(
        mcp.sse_app(host=host, message_path="/sse/messages/"),
        auth_token,
    )
    logger.info("SSE MCP demo listening on http://%s:%d/sse", host, port)
    config = uvicorn.Config(starlette_app, host=host, port=port, log_level="info")
    server = uvicorn.Server(config)
    await server.serve()


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="EnterpriseRag local MCP demo server")
    parser.add_argument(
        "--transport",
        choices=["http", "sse"],
        default=os.getenv("MCP_TRANSPORT", "http"),
        help="Network transport (default: http / Streamable HTTP)",
    )
    parser.add_argument("--host", default=os.getenv("MCP_HOST", "127.0.0.1"))
    parser.add_argument("--port", type=int, default=int(os.getenv("MCP_PORT", "8010")))
    return parser.parse_args()


async def main() -> None:
    args = parse_args()
    if args.transport == "http":
        await run_http(args.host, args.port)
    else:
        await run_sse(args.host, args.port)


if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        logger.info("stopped")
