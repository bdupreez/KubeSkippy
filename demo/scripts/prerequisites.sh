#!/bin/bash
# Prerequisites checking functions

check_command() {
    if ! command -v $1 >/dev/null 2>&1; then
        echo -e "${RED}❌ $1 is required but not installed.${NC}"
        exit 1
    fi
}

check_docker_running() {
    echo -n "Checking Docker daemon... "
    if docker info >/dev/null 2>&1; then
        echo -e "${GREEN}✓ Running${NC}"
        return 0
    else
        echo -e "${RED}✗ Not running${NC}"
        echo ""
        echo -e "${YELLOW}Docker is not running. Please start Docker:${NC}"
        echo "  • On macOS: Open Docker Desktop from Applications"
        echo "  • On Linux: sudo systemctl start docker"
        echo "  • Wait for Docker to fully start (usually 30-60 seconds)"
        echo ""
        echo "Once Docker is running, re-run this script."
        exit 1
    fi
}

check_prerequisites() {
    check_command docker
    check_command kubectl
    check_command kind
    check_command curl
    check_command kustomize
    check_docker_running
}