const fs = require('fs');
const path = require('path');

// Ensure the icons directory exists
const iconsDir = path.join(__dirname, '../public/icons');
if (!fs.existsSync(iconsDir)) {
  fs.mkdirSync(iconsDir, { recursive: true });
}

// Create a simple lock icon SVG for each size
const sizes = [16, 48, 128];

sizes.forEach(size => {
  // Scale the icon elements based on size
  const strokeWidth = Math.max(1, Math.floor(size / 16));
  const lockWidth = Math.floor(size * 0.6);
  const lockHeight = Math.floor(size * 0.8);
  const lockX = Math.floor((size - lockWidth) / 2);
  const lockY = Math.floor((size - lockHeight) / 2);
  const shackleWidth = Math.floor(lockWidth * 0.6);
  const shackleHeight = Math.floor(lockHeight * 0.4);
  const shackleX = Math.floor(lockX + (lockWidth - shackleWidth) / 2);
  const shackleY = Math.floor(lockY - shackleHeight * 0.7);
  const keyhole = Math.floor(size * 0.1);
  const keyholeX = Math.floor(size / 2);
  const keyholeY = Math.floor(size / 2 + size * 0.1);

  // Create SVG content
  const svgContent = `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<svg width="${size}" height="${size}" viewBox="0 0 ${size} ${size}" xmlns="http://www.w3.org/2000/svg">
  <rect x="${lockX}" y="${lockY}" width="${lockWidth}" height="${lockHeight}" rx="${Math.floor(size/16)}" 
        fill="#4285F4" stroke="#2965C9" stroke-width="${strokeWidth}" />
  <path d="M ${shackleX} ${lockY} 
           C ${shackleX} ${shackleY} 
             ${shackleX + shackleWidth} ${shackleY} 
             ${shackleX + shackleWidth} ${lockY}" 
        fill="none" stroke="#2965C9" stroke-width="${strokeWidth}" />
  <circle cx="${keyholeX}" cy="${keyholeY}" r="${keyhole}" fill="#FFFFFF" />
</svg>`;

  // Write the SVG file
  fs.writeFileSync(path.join(iconsDir, `icon${size}.svg`), svgContent);
  
  console.log(`Created icon${size}.svg`);
});

console.log('Icon generation complete. SVG icons have been created in the public/icons directory.');