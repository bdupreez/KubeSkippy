#!/bin/bash
# Prerequisites checking functions

check_command() {
    if ! command -v $1 >/dev/null 2>&1; then
        echo -e "${RED}❌ $1 is required but not installed.${NC}"
        exit 1
    fi
}

check_prerequisites() {
    check_command docker
    check_command kubectl
    check_command kind
    check_command curl
    check_command kustomize
}