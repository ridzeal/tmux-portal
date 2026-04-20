# Tmux Portal

Web-based tmux session manager accessible via browser. Control and monitor tmux sessions from anywhere through Cloudflare tunnel.

## Features

- 🌐 Browser-based terminal interface
- 📋 List, create, and kill tmux sessions
- 🔗 Attach to any active session
- 🎨 Modern dark UI with xterm.js
- 🔄 Auto-refresh session list
- 🚀 Single Go binary deployment

## Prerequisites

- Go 1.21+
- tmux
- systemd (for service management)

## Installation

1. **Clone or create project:**
   ```bash
   cd /path/to/tmux-portal
   ```

2. **Install dependencies:**
   ```bash
   make deps
   ```

3. **Build the binary:**
   ```bash
   make build
   ```

4. **Install as systemd service:**
   ```bash
   make install-service
   ```

5. **Start the service:**
   ```bash
   make start
   ```

## Usage

### Development Mode

```bash
make run
```

Then open http://localhost:7777 in your browser.

### Production Mode (Service)

```bash
# Install service
make install-service

# Start service
make start

# Check status
make status

# View logs
make logs

# Restart service
make restart
```

### Cloudflare Tunnel Setup

Expose the portal via Cloudflare tunnel:

```bash
cloudflared tunnel --url http://localhost:7777
```

Then configure `local.zealhaven.net` to point to the tunnel.

## Configuration

Set environment variables or use CLI flags:

- `PORT` - Server port (default: 7777)
- Or run: `./tmux-portal --port=8080`

## Project Structure

```
tmux-portal/
├── main.go              # Main server
├── tmux.go              # Tmux control functions
├── handlers.go          # HTTP handlers
├── websocket.go         # WebSocket handler
├── static/
│   ├── index.html       # Main UI
│   └── app.js           # Frontend logic
├── Makefile             # Build commands
└── tmux-portal.service  # systemd service file
```

## API Endpoints

- `GET /` - Web interface
- `GET /api/sessions` - List all sessions
- `POST /api/sessions` - Create new session
- `DELETE /api/sessions/:name` - Kill session
- `GET /ws?session=name` - WebSocket connection

## Security

- Runs as unprivileged user
- No authentication (handled by Cloudflare)
- NoNewPrivileges in service
- Only accessible through Cloudflare tunnel

## License

MIT
