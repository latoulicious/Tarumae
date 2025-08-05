.PHONY: build run clean check-commands register-commands delete-commands

# Suppress gopus warnings
export CGO_CFLAGS=-Wno-stringop-overread -Wno-format -Wno-unused-parameter -Wno-pragma-messages

# Build the bot
build:
	@echo "Building Tarumae bot..."
	go build -o tarumae cmd/main.go

# Run the bot
run:
	@echo "Running Tarumae bot..."
	go run cmd/main.go

# Check registered commands
check-commands:
	@echo "Checking registered slash commands..."
	go run tools/slash-manager.go -action check

# Register slash commands
register-commands:
	@echo "Registering slash commands..."
	go run tools/slash-manager.go -action register

# Delete all slash commands
delete-commands:
	@echo "Deleting all slash commands..."
	go run tools/slash-manager.go -action delete-all

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f tarumae
	go clean

# Setup: delete old commands and register new ones
setup-commands: delete-commands register-commands check-commands
	@echo "✅ Slash commands setup complete!" 