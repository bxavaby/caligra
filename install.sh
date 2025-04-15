#!/bin/bash

# BYZRA ⸻ install.sh
# CALIGRA installation script

set -e

BOLD="\033[1m"
RED="\033[31m"
GREEN="\033[32m"
YELLOW="\033[33m"
BLUE="\033[34m"
RESET="\033[0m"

echo -e "${BOLD}${BLUE}CALIGRA${RESET} Installation Script"
echo -e "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

echo -e "\n${BOLD}Checking dependencies...${RESET}"

if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: Go is not installed${RESET}"
    echo -e "Please install Go 1.21 or higher from https://golang.org/dl/"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
echo -e "✓ Go ${GO_VERSION} found"

if ! command -v exiftool &> /dev/null; then
    echo -e "${YELLOW}Warning: ExifTool not found${RESET}"
    echo -e "ExifTool is required for metadata extraction."
    echo -e "\nWould you like to install ExifTool now? (Y/n)"
    read -r response
    if [[ "$response" =~ ^([yY][eE][sS]|[yY]|)$ ]]; then
        if command -v apt-get &> /dev/null; then
            echo -e "Installing with apt..."
            sudo apt-get update
            sudo apt-get install -y libimage-exiftool-perl
        elif command -v dnf &> /dev/null; then
            echo -e "Installing with dnf..."
            sudo dnf install -y perl-Image-ExifTool
        elif command -v pacman &> /dev/null; then
            echo -e "Installing with pacman..."
            sudo pacman -S --noconfirm perl-image-exiftool
        else
            echo -e "${RED}Could not automatically install ExifTool.${RESET}"
            echo -e "Please install ExifTool manually and try again."
            echo -e "Visit: https://exiftool.org/install.html for instructions."
            exit 1
        fi
    else
        echo -e "${YELLOW}Warning: CALIGRA requires ExifTool for metadata operations.${RESET}"
        echo -e "Continuing installation, but functionality will be limited."
    fi
fi

if command -v exiftool &> /dev/null; then
    EXIFTOOL_VERSION=$(exiftool -ver)
    echo -e "✓ ExifTool ${EXIFTOOL_VERSION} found"
fi

if ! command -v ffmpeg &> /dev/null; then
    echo -e "${YELLOW}Warning: FFmpeg not found${RESET}"
    echo -e "Audio and video processing will be limited. Install FFmpeg with:"
    echo -e "  sudo apt install ffmpeg  # For Debian/Ubuntu"
    echo -e "  sudo dnf install ffmpeg  # For Fedora"
    echo -e "  sudo pacman -S ffmpeg    # For Arch Linux"
fi

echo -e "\n${BOLD}Building CALIGRA...${RESET}"
go build -o caligra cmd/caligra/main.go

if [ $? -ne 0 ]; then
    echo -e "${RED}Build failed!${RESET}"
    exit 1
fi

echo -e "${GREEN}✓ Build successful${RESET}"

echo -e "\n${BOLD}Setting up configuration...${RESET}"
CONFIG_DIR="$HOME/.caligra/config"
mkdir -p "$CONFIG_DIR"
mkdir -p "$HOME/.caligra/logs"

if [ ! -f "$CONFIG_DIR/profile.lua" ]; then
    cp -n config/profile.lua "$CONFIG_DIR/" 2>/dev/null || true
    echo -e "✓ Installed default profile"
fi

if [ ! -f "$CONFIG_DIR/scroud.toml" ]; then
    cp -n config/scroud.toml "$CONFIG_DIR/" 2>/dev/null || true
    echo -e "✓ Installed default daemon configuration"
fi

if [ ! -f "$CONFIG_DIR/yogra.toml" ]; then
    cp -n config/yogra.toml "$CONFIG_DIR/" 2>/dev/null || true
    echo -e "✓ Installed default color theme"
fi

echo -e "\n${BOLD}Installing CALIGRA...${RESET}"
if [ -d "$HOME/.local/bin" ]; then
    if [[ ":$PATH:" != *":$HOME/.local/bin:"* ]]; then
        echo -e "${YELLOW}Warning: ~/.local/bin is not in your PATH${RESET}"
        echo -e "Add the following to your ~/.bashrc or ~/.zshrc:"
        echo -e "  export PATH=\$HOME/.local/bin:\$PATH"
    fi

    cp caligra "$HOME/.local/bin/"
    echo -e "${GREEN}✓ Installed to ~/.local/bin/caligra${RESET}"
elif [ -w "/usr/local/bin" ]; then
    cp caligra "/usr/local/bin/"
    echo -e "${GREEN}✓ Installed to /usr/local/bin/caligra${RESET}"
else
    echo -e "Installing system-wide (requires sudo)..."
    sudo cp caligra "/usr/local/bin/"
    echo -e "${GREEN}✓ Installed to /usr/local/bin/caligra${RESET}"
fi

echo -e "\n${BOLD}${GREEN}Installation complete!${RESET}"
echo -e "Run '${BOLD}caligra help${RESET}' to get started."
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
