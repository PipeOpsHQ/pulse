#!/bin/bash
# Generate favicon.ico from SVG (requires ImageMagick or Inkscape)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
FRONTEND_PUBLIC="$PROJECT_ROOT/frontend/public"

# Check for ImageMagick
if command -v convert &> /dev/null; then
    echo "Using ImageMagick to generate favicon.ico"
    convert -background none -resize 32x32 "$FRONTEND_PUBLIC/favicon.svg" "$FRONTEND_PUBLIC/favicon.ico"
    echo "✅ Generated favicon.ico"
elif command -v inkscape &> /dev/null; then
    echo "Using Inkscape to generate favicon.ico"
    inkscape --export-type=ico --export-filename="$FRONTEND_PUBLIC/favicon.ico" \
             --export-width=32 --export-height=32 "$FRONTEND_PUBLIC/favicon.svg"
    echo "✅ Generated favicon.ico"
else
    echo "⚠️  ImageMagick or Inkscape not found. Using SVG favicon only."
    echo "   Modern browsers support SVG favicons, but for best compatibility,"
    echo "   install ImageMagick: brew install imagemagick (macOS) or apt-get install imagemagick (Linux)"
fi
