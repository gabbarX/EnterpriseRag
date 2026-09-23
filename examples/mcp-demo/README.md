# EnterpriseRag Local MCP Demo

A minimal external MCP service, used to test EnterpriseRag **acting as an MCP client**
against a third-party tool server.

It exposes six demo tools:

| Tool | Purpose |
| --- | --- |
| `echo` | Connectivity self-check |
| `add` | Adds two numbers |
| `server_time` | Returns the server's UTC time |
| `lookup_policy` | Looks up demo policies (warranty, reimbursement, POC and so on) |
| `list_team_contacts` | Lists the demo project team members |
| `send_demo_alert` | Simulates an outbound notification (handy for testing manual tool approval) |

## 1. Start the server

```bash
cd examples/mcp-demo
chmod +x start.sh
./start.sh
```

`start.sh` creates a `.venv` and installs the dependencies for you. By default it
listens on `http://127.0.0.1:8010/mcp` with the auth token `enterpriserag-demo-token`.

To customise it:

```bash
export MCP_SERVER_AUTH_TOKEN=my-secret
export MCP_PORT=9000
./start.sh
```

## 2. Self-check

In a second terminal:

```bash
cd examples/mcp-demo
source .venv/bin/activate
python test_tools.py
```

This should list all six tools.

## 3. Connect it to EnterpriseRag

1. Open **Settings -> MCP Services -> New**
2. Fill in:

| Field | Value |
| --- | --- |
| Name | `Local MCP Demo` |
| Transport | **HTTP Streamable** |
| URL | `http://127.0.0.1:8010/mcp` |
| Authentication | **Bearer** |
| Token | `enterpriserag-demo-token` (must match `MCP_SERVER_AUTH_TOKEN`) |

3. Save, then click **Test Connection**. Six tools should be discovered.
4. In the **Agent** configuration, tick this MCP service (or select all its tools).
5. (Optional) Enable **manual approval** for `send_demo_alert`, so a confirmation
   prompt appears before the agent calls it during a conversation.

## 4. Questions worth trying

Ask these in an agent conversation:

- "Use an MCP tool to check how long the smart home hub warranty lasts" -> should trigger `lookup_policy`
- "Who is the engineering POC lead?" -> `lookup_policy` or `list_team_contacts`
- "What time is it on the MCP demo server?" -> `server_time`

If you have also imported the matching demo documents, you can compare the
**knowledge base retrieval answer** with the **MCP tool response** and check that
they agree.

## 5. Things to keep in mind

- The EnterpriseRag UI **does not support the stdio transport**; you must use
  **HTTP Streamable** or **SSE**.
- The demo binds only to `127.0.0.1`. Do not expose it to the public internet.
- `send_demo_alert` never sends a real message; it only returns a simulated result.

## 6. SSE mode (optional)

```bash
MCP_TRANSPORT=sse MCP_PORT=8011 ./start.sh
```

In EnterpriseRag, choose the **SSE** transport and set the URL to
`http://127.0.0.1:8011/sse`.
