.PHONY: build run clean install deps test lint vet install-service uninstall-service start stop restart status logs

# Configuration
PORT ?= 7777
BINARY_NAME ?= tmux-portal

# Build the Go binary
build:
	go build -o $(BINARY_NAME) .

# Run the server (development)
run:
	go run .

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME)

# Install dependencies
deps:
	go mod tidy
	go mod download

# Run tests
test:
	go test ./... -v -race -cover

# Run linter
lint:
	golangci-lint run ./...

# Run go vet
vet:
	go vet ./...

# Install as systemd service
install-service:
	sed 's|ExecStart=.*|ExecStart=$(PWD)/$(BINARY_NAME) --port $(PORT)|' tmux-portal.service.example > tmux-portal.service
	systemctl --user link $(PWD)/tmux-portal.service
	systemctl --user daemon-reload
	systemctl --user enable tmux-portal
	@echo "Service installed. Start with: systemctl --user start tmux-portal"

# Uninstall systemd service
uninstall-service:
	systemctl --user disable tmux-portal
	systemctl --user daemon-reload
	@echo "Service uninstalled"

# Start service
start:
	systemctl --user start tmux-portal

# Stop service
stop:
	systemctl --user stop tmux-portal

# Restart service
restart:
	systemctl --user restart tmux-portal

# Check service status
status:
	systemctl --user status tmux-portal

# View logs
logs:
	journalctl --user -u tmux-portal -f
