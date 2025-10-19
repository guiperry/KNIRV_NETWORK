const { execSync } = require('child_process');
const fs = require('fs');
const path = require('path');

// Install tailwindcss and its dependencies
console.log('Installing tailwindcss and its dependencies...');
try {
  execSync('npm install -D tailwindcss postcss autoprefixer tailwindcss-animate', { stdio: 'inherit' });
  console.log('Tailwindcss and its dependencies installed successfully.');
} catch (error) {
  console.error('Error installing tailwindcss:', error);
  process.exit(1);
}

// Create tailwind.config.js
console.log('Creating tailwind.config.js...');
const tailwindConfig = `/** @type {import('tailwindcss').Config} */
module.exports = {
  darkMode: ["class"],
  content: [
    "./pages/**/*.{ts,tsx}",
    "./components/**/*.{ts,tsx}",
    "./app/**/*.{ts,tsx}",
    "./src/**/*.{ts,tsx}",
  ],
  prefix: "",
  theme: {
    container: {
      center: true,
      padding: "2rem",
      screens: {
        "2xl": "1400px",
      },
    },
    extend: {
      colors: {
        bgPrimaryColor: "var(--bg)",
        bgSecondaryColor: "var(--bgSecondary)",
        softTextColor: "var(--softTextColor)",

        border: "hsl(var(--border))",
        input: "hsl(var(--input))",
        ring: "hsl(var(--ring))",
        background: "hsl(var(--background))",
        foreground: "hsl(var(--foreground))",
        primary: {
          DEFAULT: "hsl(var(--primary))",
          foreground: "hsl(var(--primary-foreground))",
        },
        secondary: {
          DEFAULT: "hsl(var(--secondary))",
          foreground: "hsl(var(--secondary-foreground))",
        },
        destructive: {
          DEFAULT: "hsl(var(--destructive))",
          foreground: "hsl(var(--destructive-foreground))",
        },
        muted: {
          DEFAULT: "hsl(var(--muted))",
          foreground: "hsl(var(--muted-foreground))",
        },
        accent: {
          DEFAULT: "hsl(var(--accent))",
          foreground: "hsl(var(--accent-foreground))",
        },
        popover: {
          DEFAULT: "hsl(var(--popover))",
          foreground: "hsl(var(--popover-foreground))",
        },
        card: {
          DEFAULT: "hsl(var(--card))",
          foreground: "hsl(var(--card-foreground))",
        },
      },
      borderRadius: {
        lg: "var(--radius)",
        md: "calc(var(--radius) - 2px)",
        sm: "calc(var(--radius) - 4px)",
      },
      keyframes: {
        "accordion-down": {
          from: { height: "0" },
          to: { height: "var(--radix-accordion-content-height)" },
        },
        "accordion-up": {
          from: { height: "var(--radix-accordion-content-height)" },
          to: { height: "0" },
        },
      },
      animation: {
        "accordion-down": "accordion-down 0.2s ease-out",
        "accordion-up": "accordion-up 0.2s ease-out",
      },
    },
  },
  plugins: [require("tailwindcss-animate")],
};`;

fs.writeFileSync(path.join(__dirname, '..', 'tailwind.config.js'), tailwindConfig);
console.log('tailwind.config.js created successfully.');

// Create postcss.config.js
console.log('Creating postcss.config.js...');
const postcssConfig = `module.exports = {
  plugins: {
    tailwindcss: {},
    autoprefixer: {},
  },
};`;

fs.writeFileSync(path.join(__dirname, '..', 'postcss.config.js'), postcssConfig);
console.log('postcss.config.js created successfully.');

console.log('Tailwind setup completed successfully.');