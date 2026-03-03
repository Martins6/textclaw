#!/bin/bash

# TextClaw Installation Script
# Auto-builds from source if no release is available
# For Unix systems only (macOS and Linux)

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_success() { echo -e "${GREEN}✓${NC} $1"; }
print_error() { echo -e "${RED}✗${NC} $1"; }
print_info() { echo -e "${YELLOW}ℹ${NC} $1"; }
print_cmd() { echo -e "${BLUE}$1${NC}"; }

case "$(uname -s)" in
    Linux|Darwin) ;;
    *) 
        print_error "Unsupported operating system: $(uname -s)"
        print_error "TextClaw only supports Unix systems (macOS and Linux)"
        exit 1
        ;;
esac

ARCH=$(uname -m)
case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *) 
        print_error "Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

LOCAL_DIR="$HOME/.local/bin"
GO_BIN="$(go env GOPATH 2>/dev/null)/bin" || "$HOME/go/bin"
BINARY_NAME="textclaw"
REPO_URL="https://github.com/Martins6/textclaw"

get_install_dir() {
    if [ -d "$LOCAL_DIR" ] || mkdir -p "$LOCAL_DIR" 2>/dev/null; then
        echo "$LOCAL_DIR"
        return 0
    fi
    
    if [ -d "$GO_BIN" ]; then
        echo "$GO_BIN"
        return 0
    fi
    
    if mkdir -p "$GO_BIN" 2>/dev/null; then
        echo "$GO_BIN"
        return 0
    fi
    
    if [ -w /usr/local/bin ]; then
        echo "/usr/local/bin"
        return 0
    fi
    
    return 1
}

build_from_source() {
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed"
        print_info "Install Go from: https://golang.org/doc/install"
        exit 1
    fi
    
    TEMP_DIR=$(mktemp -d)
    print_info "Cloning repository..."
    git clone "$REPO_URL" "$TEMP_DIR"
    
    cd "$TEMP_DIR"
    go build -o "$BINARY_NAME" ./cmd/textclaw
    cd -
    
    mv "$TEMP_DIR/$BINARY_NAME" .
    rm -rf "$TEMP_DIR"
    print_success "Built $BINARY_NAME"
}

main() {
    print_info "Installing TextClaw"
    print_info "==================="
    print_info "Platform: $(uname -s)/$ARCH"
    echo ""

    TARGET_DIR=$(get_install_dir)
    if [ $? -ne 0 ]; then
        print_error "Could not find a suitable installation directory."
        echo ""
        echo "Please create one of these directories manually:"
        print_cmd "  mkdir -p ~/.local/bin"
        echo "or"
        print_cmd "  mkdir -p ~/go/bin"
        echo ""
        print_cmd "  export PATH=\"\$PATH:\$HOME/.local/bin\""
        exit 1
    fi

    if [ -f "go.mod" ] && [ -d "cmd" ]; then
        print_info "Building from source..."
        
        if ! command -v go &> /dev/null; then
            print_error "Go is not installed"
            print_info "Install Go from: https://golang.org/doc/install"
            exit 1
        fi
        
        go build -o "$BINARY_NAME" ./cmd/textclaw
        print_success "Built $BINARY_NAME"
    else
        print_info "Downloading pre-built binary..."
        
        OS=$(uname -s | tr '[:upper:]' '[:lower:]')
        BINARY_URL="$REPO_URL/releases/latest/download/textclaw-${OS}-${ARCH}"
        
        if command -v curl &> /dev/null; then
            if ! curl -fsSL "$BINARY_URL" -o "$BINARY_NAME" 2>/dev/null; then
                print_info "No pre-built release found, building from source..."
                build_from_source
            fi
        elif command -v wget &> /dev/null; then
            if ! wget -q "$BINARY_URL" -O "$BINARY_NAME" 2>/dev/null; then
                print_info "No pre-built release found, building from source..."
                build_from_source
            fi
        else
            print_error "Need curl or wget to download"
            exit 1
        fi
        
        if [ -f "$BINARY_NAME" ]; then
            chmod +x "$BINARY_NAME"
            print_success "Downloaded $BINARY_NAME"
        fi
    fi

    print_info "Installing to $TARGET_DIR..."
    cp -f "$BINARY_NAME" "$TARGET_DIR/"
    chmod +x "$TARGET_DIR/$BINARY_NAME"
    print_success "Installed to $TARGET_DIR/$BINARY_NAME"

    rm -f "$BINARY_NAME"

    if [ -x "$TARGET_DIR/$BINARY_NAME" ]; then
        print_success "Installation complete!"
        echo ""
        echo "Quick start:"
        print_cmd "  textclaw init         # Initialize TextClaw"
        print_cmd "  textclaw --help       # Show all commands"
        echo ""
        
        if [[ ":$PATH:" != *":$TARGET_DIR:"* ]]; then
            print_info "Add $TARGET_DIR to your PATH:"
            print_cmd "  export PATH=\"\$PATH:$TARGET_DIR\""
            echo ""
            print_info "Add this to your ~/.bashrc or ~/.zshrc to make it permanent."
        fi
    else
        print_error "Installation verification failed"
        exit 1
    fi
}

main "$@"
