#!/bin/bash
# Usage: ./RUNME.sh [--help|-h]
if [[ "$1" == "--help" || "$1" == "-h" ]]; then
  echo "Usage: ./RUNME.sh"
  echo "This script will display the README and guide you through the OpenWebUI WSL2 setup."
  exit 0
fi
cat ./README.md