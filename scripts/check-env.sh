#!/bin/bash

# Development Environment Check Script for web3signer-go
# Ensures all prerequisites are met for development

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🔍 Checking web3signer-go development environment...${NC}"

# Check Go version
echo -e "\n${BLUE}📦 Checking Go version...${NC}"
if command -v go &> /dev/null; then
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    REQUIRED_VERSION="1.25"
    
    if [[ "$(printf '%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V | head -n1)" = "$REQUIRED_VERSION" ]]; then
        echo -e "${GREEN}✅ Go version $GO_VERSION meets requirement (>= $REQUIRED_VERSION)${NC}"
    else
        echo -e "${RED}❌ Go version $GO_VERSION is too old. Required: >= $REQUIRED_VERSION${NC}"
        echo -e "${YELLOW}💡 Please install Go $REQUIRED_VERSION or later from https://golang.org/dl/${NC}"
        exit 1
    fi
else
    echo -e "${RED}❌ Go is not installed${NC}"
    echo -e "${YELLOW}💡 Please install Go $REQUIRED_VERSION or later from https://golang.org/dl/${NC}"
    exit 1
fi

# Check golangci-lint
echo -e "\n${BLUE}🔧 Checking golangci-lint...${NC}"
if command -v golangci-lint &> /dev/null; then
    echo -e "${GREEN}✅ golangci-lint is installed${NC}"
else
    echo -e "${YELLOW}⚠️  golangci-lint is not installed. Run 'make install-tools' to install it${NC}"
fi

# Check project files
echo -e "\n${BLUE}📁 Checking project structure...${NC}"
if [[ -f "go.mod" ]]; then
    echo -e "${GREEN}✅ go.mod found${NC}"
else
    echo -e "${RED}❌ go.mod not found${NC}"
    exit 1
fi

if [[ -f "Makefile" ]]; then
    echo -e "${GREEN}✅ Makefile found${NC}"
else
    echo -e "${RED}❌ Makefile not found${NC}"
    exit 1
fi

# Check if we can build
echo -e "\n${BLUE}🏗️  Testing build...${NC}"
if go build -o /tmp/web3signer ./cmd/web3signer 2>/dev/null; then
    echo -e "${GREEN}✅ Build successful${NC}"
    rm -f /tmp/web3signer
else
    echo -e "${RED}❌ Build failed${NC}"
    exit 1
fi

echo -e "\n${GREEN}🎉 Environment check passed! You're ready to develop web3signer-go${NC}"
echo -e "${BLUE}💡 Useful commands:${NC}"
echo -e "   make help        - Show available commands"
echo -e "   make check       - Run tests and code quality checks"
echo -e "   make test        - Run tests only"
echo -e "   make lint        - Run code quality checks only"