# Makefile

# Directories
GO_DIR := server
UI_DIR := ui

# Commands
GO_BUILD_CMD := GOOS=linux go build -o go-serverless ./main.go
UI_BUILD_CMD := npm install && npm run build  # Added npm install here

# Default target to build both frontend and backend
all: build

# Build the React frontend
build-ui:
    @echo "Building React frontend..."
    cd $(UI_DIR) && $(UI_BUILD_CMD)

# Build the Go serverless backend
build-go:
    @echo "Building Go serverless backend..."
    cd $(GO_DIR) && $(GO_BUILD_CMD)

# Full build (both frontend and backend)
build: build-ui build-go

# Deploy (assuming using Netlify CLI to deploy)
deploy: build
    @echo "Deploying to Netlify..."
    netlify deploy --prod

# Clean build files
clean:
    @echo "Cleaning build files..."
    rm -rf $(UI_DIR)/build $(GO_DIR)/go-serverless
