# Operations Guide: Voice Agent

## Deployment

The Voice Agent is containerized using Docker. To build and run:

```bash
docker build -t voice-agent .
docker run -p 8080:8080 --env-file .env voice-agent
```

## Configuration

Settings are managed via `config.yaml`. Key parameters:
- `gemini.api_key`: Required for AI communication.
- `twilio.auth_token`: Required for webhook security.
- `database_url`: Postgres connection string.

## Monitoring

- **Metrics:** Available at `/metrics` in Prometheus format.
- **Health Checks:** `/healthz` (liveness) and `/readyz` (readiness).
- **Logs:** Structured JSON output to stdout.

## Adding New MCP Servers

1. Edit `config/mcp.yaml`.
2. Add a new entry under `servers`.
3. Restart the Voice Agent service.

```yaml
- name: my-new-tool
  transport: stdio
  command: node
  args: ["dist/index.js"]
  enabled: true
```
