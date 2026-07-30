# Security Policy

## Reporting a Vulnerability

Please report security issues privately via **GitHub Security Advisories**:

1. Go to https://github.com/miniwater/click/security/advisories
2. Click **New draft security advisory**
3. Provide a clear description and steps to reproduce

Do **not** open a public issue for a security vulnerability.

## Known Security Considerations

- WebSocket `CheckOrigin` currently accepts all origins. In production behind a reverse proxy, restrict access at the proxy layer.
- `X-Forwarded-For` / `X-Real-IP` headers are unconditionally trusted. Only expose the server directly if these headers cannot be spoofed (e.g., behind a trusted reverse proxy).
- There is no per-client rate limiting. Malicious clients could spam clicks or chat. Consider adding a reverse-proxy rate limiter for public deployments.
- Chat messages and masked IPs are persisted in SQLite. No message-level moderation or deletion API is provided.
