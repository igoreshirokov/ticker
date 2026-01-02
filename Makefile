BINARY_NAME=website-checker.exe
APP_DIR=./cmd/website-checker
LDFLAGS=-ldflags="-s -w -H windowsgui"

all: build

build:
	@echo "Compilation..."
	go build $(LDFLAGS) -o dist/$(BINARY_NAME) $(APP_DIR)
	@echo "Compile is completed. File: dist/$(BINARY_NAME)"

clean:
	@echo "Cleaning..."
	-@if exist dist\$(BINARY_NAME) del /Q dist\$(BINARY_NAME)
	@echo "Cleaning is completed."

help:
	@echo "Commands:"
	@echo "  make build   - compile application"
	@echo "  make clean   - remove binary file"
	@echo "  make help    - show this message"