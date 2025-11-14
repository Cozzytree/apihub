APP_NAME = apihub
VERSION  = v0.1.1

# Default build (for your current OS)
build:
	@echo "Building $(APP_NAME) for current system..."
	@go build -o $(APP_NAME)

# Cross-compile for Linux
build-linux:
	@echo "Building $(APP_NAME) for Linux amd64..."
	@GOOS=linux GOARCH=amd64 go build -o $(APP_NAME)-linux-amd64 .

# Cross-compile for Windows
build-windows:
	@echo "Building $(APP_NAME) for Windows amd64..."
	@GOOS=windows GOARCH=amd64 go build -o $(APP_NAME)-windows-amd64.exe .

# Cross-compile for macOS
build-macos:
	@echo "Building $(APP_NAME) for macOS amd64..."
	@GOOS=darwin GOARCH=amd64 go build -o $(APP_NAME)-darwin-amd64 .

# Build all platforms
build-all: build-linux build-windows build-macos

# Create a release (using gh)
release: build-all
	@gh release create $(VERSION) \
		./$(APP_NAME)-linux-amd64 \
		./$(APP_NAME)-windows-amd64.exe \
		./$(APP_NAME)-darwin-amd64 \
		--title "$(VERSION)" \
		--notes "Release $(VERSION)"

clean:
	@rm -f $(APP_NAME) $(APP_NAME)-*
	@echo "Cleaned build artifacts."
