#!/bin/bash
set -e

echo "🚀 KubeSkippy Demo Setup"
echo "========================"
echo "🧠 AI-driven healing with organized manifests and dashboard fixes"
echo ""

# Get script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Run the clean setup
exec "$SCRIPT_DIR/scripts/setup.sh"