# Security Policy

## Supported Versions

Only the latest release is supported with security updates.

## Reporting a Vulnerability

If you find a security vulnerability, please do NOT open a public issue.
Instead, report it privately so we can address it before disclosure.

Please open a private security advisory on GitHub:
https://github.com/ridzeal/tmux-portal/security/advisories/new

## Security Considerations

- This application is designed to run behind Cloudflare tunnel
- No built-in authentication (relies on Cloudflare Access)
- Runs as an unprivileged user via systemd
- Only expose via trusted network/Cloudflare tunnel