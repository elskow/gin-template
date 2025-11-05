#!/bin/bash

# Project Renaming Script
# This script helps you rename the entire project to your own module name
# Usage: ./rename_project.sh <new-module-path>
# Example: ./rename_project.sh github.com/yourusername/your-project

set -e

NEW_MODULE=$1
OLD_MODULE=$(grep "^module " go.mod | awk '{print $2}')

if [ -z "$NEW_MODULE" ]; then
    echo "Usage: ./rename_project.sh <new-module-path>"
    echo ""
    echo "Example:"
    echo "  ./rename_project.sh github.com/yourusername/your-project"
    echo ""
    echo "Current module: $OLD_MODULE"
    exit 1
fi

echo "Renaming project module..."
echo "From: $OLD_MODULE => To:   $NEW_MODULE"

# Confirm with user
read -p "Continue? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Cancelled"
    exit 1
fi

echo "Updating go.mod..."
sed -i.bak "s|$OLD_MODULE|$NEW_MODULE|g" go.mod && rm go.mod.bak

echo "Updating all Go files..."
find . -name "*.go" -type f ! -path "./.git/*" -exec sed -i.bak "s|$OLD_MODULE|$NEW_MODULE|g" {} \;
find . -name "*.bak" -delete

echo "Tidying modules..."
go mod tidy

echo "Project renamed successfully!"
