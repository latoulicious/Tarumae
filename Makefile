.PHONY: build run dev clean check-commands register-commands delete-commands setup-commands

# Suppress gopus warnings (where applicable)
export CGO_CFLAGS=-Wno-stringop-overread -Wno-format -Wno-unused-parameter -Wno-pragma-messages

# Binary name
BIN_NAME=tarumae

# Source entry
ENTRY=cmd/main.go

# Build the bot (optimized)
build:
	@echo "🔧 Building $(BIN_NAME) with CGO optimizations and warning suppression..."
	CGO_CFLAGS="-O2 -Wno-stringop-overread -Wno-unused-parameter" go build -o $(BIN_NAME) $(ENTRY)

# Run the optimized binary
run: build
	@echo "🚀 Running $(BIN_NAME)..."
	./$(BIN_NAME)

# Fast dev run using `go run` (may show gopus warning)
dev:
	@echo "⚡ Dev running (no build, fast iteration)..."
	go run $(ENTRY)

# Check registered commands
check-commands:
	@echo "🔍 Checking registered slash commands..."
	go run tools/slash-manager.go -action check

# Register slash commands
register-commands:
	@echo "📝 Registering slash commands..."
	go run tools/slash-manager.go -action register

# Delete all slash commands
delete-commands:
	@echo "🗑️ Deleting all slash commands..."
	go run tools/slash-manager.go -action delete-all

# Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	rm -f $(BIN_NAME)
	go clean

# Setup: delete old commands and register new ones
setup-commands: delete-commands register-commands check-commands
	@echo "✅ Slash commands setup complete!"
