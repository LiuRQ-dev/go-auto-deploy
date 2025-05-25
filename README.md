# Go Auto Deploy System

A lightweight webhook server written in Go for automatic deployment of GitHub repositories. It handles push events, runs `git pull`, and triggers custom post-deploy commands. Includes Telegram notification support.

## Features

* GitHub webhook receiver with secret verification
* Auto `git pull` and rebuild
* Custom post-deploy command support
* Built-in Makefile for easy commands

---

## Quick Start

### 1. Clone and Build

```bash
git clone https://github.com/your-user/go-auto-deploy.git
cd go-auto-deploy
make init
make build
```

### 2. Configuration

Edit `config.yaml`:

```yaml
port: 3020
secret: your-webhook-secret
repo_path: /path/to/your/project
post_deploy: go build .

```

### 3. Run

```bash
make run
```

Or run in development mode:

```bash
make dev
```

---

## GitHub Webhook Setup

1. Go to **Repo Settings → Webhooks**
2. Set Payload URL: `https://your-ngrok-url/webhook`
3. Choose `application/json`
4. Use the same secret as in `config.yaml`

---

To uninstall:

```bash
make service-uninstall
```

---


## Development & Tools

```bash
make test         # Run tests
make logs         # View deployment logs
make fmt          # Format code
make lint         # Lint check
```

---

## License

MIT License
