#!/bin/bash

# Create placeholder icons for PWA
# This creates simple SVG-based icons that can be converted to PNG

create_icon() {
    local size=$1
    local filename="public/icon-${size}.png"
    
    # Create SVG content
    cat > temp_icon.svg << EOF
<svg width="${size}" height="${size}" xmlns="http://www.w3.org/2000/svg">
  <defs>
    <linearGradient id="grad" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" style="stop-color:#8b5cf6;stop-opacity:1" />
      <stop offset="100%" style="stop-color:#06b6d4;stop-opacity:1" />
    </linearGradient>
  </defs>
  <rect width="${size}" height="${size}" fill="url(#grad)" rx="$(($size/8))"/>
  <text x="50%" y="50%" font-family="Arial, sans-serif" font-size="$(($size/4))" 
        fill="white" text-anchor="middle" dominant-baseline="central" font-weight="bold">K</text>
</svg>
EOF
    
    # Convert SVG to PNG if ImageMagick is available
    if command -v convert &> /dev/null; then
        convert temp_icon.svg "$filename"
        echo "Created $filename"
    elif command -v rsvg-convert &> /dev/null; then
        rsvg-convert -w $size -h $size temp_icon.svg -o "$filename"
        echo "Created $filename"
    else
        # Just copy the SVG as a fallback
        cp temp_icon.svg "public/icon-${size}.svg"
        echo "Created public/icon-${size}.svg (install ImageMagick or rsvg-convert for PNG)"
    fi
    
    rm -f temp_icon.svg
}

echo "Creating PWA icons..."

# Create icons in various sizes
create_icon 72
create_icon 96
create_icon 128
create_icon 144
create_icon 152
create_icon 192
create_icon 384
create_icon 512

echo "Icon creation completed!"
echo ""
echo "Note: For production, replace these placeholder icons with proper KNIRV branding."
