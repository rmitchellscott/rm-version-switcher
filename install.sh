#!/bin/bash

# rm-version-switcher installer script
# Detects architecture and downloads the correct binary from latest GitHub release

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# GitHub repository
REPO="rmitchellscott/rm-version-switcher"
BINARY_NAME="rm-version-switcher"

# Function to print colored output
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Detect architecture
detect_arch() {
    local arch=$(uname -m)
    
    case $arch in
        armv7l)
            echo "armv7"
            ;;
        aarch64)
            echo "aarch64"
            ;;
        *)
            print_error "Unsupported architecture: $arch"
            print_error "This script is designed for reMarkable devices (armv7l or aarch64)"
            exit 1
            ;;
    esac
}

# Get latest release version from GitHub API
get_latest_version() {
    local api_url="https://api.github.com/repos/$REPO/releases/latest"
    
    local version=$(wget -qO- "$api_url" 2>/dev/null | grep -o '"tag_name": "[^"]*' | grep -o '[^"]*$' 2>/dev/null || echo "")
    
    if [ -z "$version" ]; then
        print_error "Failed to fetch latest version information"
        exit 1
    fi
    
    echo "$version"
}

# Download and install binary
install_binary() {
    local arch=$1
    local version=$2
    local binary_name="${BINARY_NAME}-${arch}"
    local download_url="https://github.com/$REPO/releases/download/$version/${binary_name}.tar.gz"
    
    print_status "Downloading $binary_name version $version..."
    
    # Download the binary
    wget -O "${binary_name}.tar.gz" "$download_url"
    
    # Extract the binary
    print_status "Extracting binary..."
    tar -xzf "${binary_name}.tar.gz"
    
    # Rename to generic name and make executable
    mv "$binary_name" "$BINARY_NAME"
    chmod +x "$BINARY_NAME"
    
    # Clean up
    rm "${binary_name}.tar.gz"
    
    print_status "Downloaded and extracted $BINARY_NAME"
    print_status "You can now run ./$BINARY_NAME"
}

# Main installation process
main() {
    print_status "reMarkable Version Switcher Installer"
    print_status "======================================"
    
    # Detect architecture
    local arch=$(detect_arch)
    print_status "Detected architecture: $arch"
    
    # Get latest version
    print_status "Fetching latest release information..."
    local version=$(get_latest_version)
    print_status "Latest version: $version"
    
    # Check if binary already exists
    if [ -f "$BINARY_NAME" ]; then
        print_warning "rm-version-switcher already exists in current directory"
        read -p "Do you want to continue and overwrite? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            print_status "Installation cancelled."
            exit 0
        fi
    fi
    
    # Install the binary
    install_binary "$arch" "$version"
}

# Run main function
main "$@"