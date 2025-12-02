#!/bin/bash

# Package Chrome Extension for Web Store

VERSION=$(grep '"version":' chrome-extension/manifest.json | cut -d '"' -f 4)
ZIP_NAME="pfc-extension-v$VERSION.zip"

echo "📦 Packaging PFC Extension v$VERSION..."

# Create zip file excluding unnecessary files
cd chrome-extension
zip -r "../$ZIP_NAME" . -x "*.DS_Store"

cd ..

echo "✅ Created $ZIP_NAME"
echo "👉 Upload this file to Chrome Web Store Developer Dashboard"
