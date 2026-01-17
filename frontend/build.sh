#!/bin/bash
# Build script that ensures correct Node version

export NVM_DIR="$HOME/.nvm"
[ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"

# Use Node 22 from .nvmrc
cd "$(dirname "$0")"
nvm use 22 || nvm use 20

# Get the full path to the Node binary
NODE_BIN=$(which node)
if [ -z "$NODE_BIN" ]; then
  echo "Error: Node.js not found. Please install Node.js 20.19+ or 22.12+"
  exit 1
fi

# Verify Node version
NODE_VERSION=$($NODE_BIN --version | cut -d'v' -f2)
NODE_MAJOR=$(echo $NODE_VERSION | cut -d'.' -f1)

if [ "$NODE_MAJOR" -lt 20 ]; then
  echo "Error: Node.js version $NODE_VERSION is too old. Please run: nvm use 22"
  exit 1
fi

echo "Using Node.js $($NODE_BIN --version)"
# Use the Node binary directly to run vite (bypassing npm which might use wrong Node)
$NODE_BIN node_modules/.bin/vite build
