# KNIRV.com Build Scripts

This directory contains scripts for generating static documentation from the Docsify site to solve CloudFlare Worker deployment issues.

## Problem Solved

CloudFlare Workers cannot run the live Docsify server (`npm run docs:serve`) because it's a static hosting environment. These scripts convert the dynamic Docsify documentation into static HTML files that work perfectly with CloudFlare Workers.

## Scripts

### `simple-static-generator.js`
**Main static generation script (RECOMMENDED)**

- Directly converts Markdown files to static HTML
- Preserves docsify styling and navigation structure
- Generates complete static site with sidebar and navigation
- Much faster and more reliable than browser-based approach
- Saves static HTML files to `/public/documentation/static/`

**Usage:**
```bash
node scripts/simple-static-generator.js
```

### `copy-docsify-assets.js`
**Browser-based static generation script (LEGACY)**

- Starts a local Docsify server
- Uses Puppeteer to crawl and render all documentation pages
- More complex but captures exact docsify rendering
- Use only if simple generator doesn't meet requirements

**Usage:**
```bash
node scripts/copy-docsify-assets.js
```

### `update-docsify-links.js`
**Link updater script**

- Scans all HTML files in the public directory
- Updates links pointing to `/documentation/docsify/` to point to `/documentation/static/`
- Ensures proper `.html` extensions for static files

**Usage:**
```bash
node scripts/update-docsify-links.js
```

### `test-static-generation.js`
**Validation script**

- Tests that static generation completed successfully
- Validates essential elements are present in generated files
- Counts generated files and reports status

**Usage:**
```bash
node scripts/test-static-generation.js
```

## Build Process

The build process is integrated into the main package.json:

```bash
# Generate static documentation (used in build)
npm run docs:generate-static

# Test static generation
npm run docs:test-static

# Full build (includes static generation)
npm run build
```

## How It Works

1. **Start Docsify Server**: Launches docsify locally on port 3000
2. **Crawl Pages**: Uses Puppeteer to visit and render all documentation pages
3. **Extract Content**: Captures fully-rendered HTML with all styling and navigation
4. **Process Links**: Converts dynamic docsify links to static HTML links
5. **Save Static Files**: Writes processed HTML to `/public/documentation/static/`
6. **Update References**: Updates all site links to point to static version

## Benefits

- ✅ **CloudFlare Worker Compatible**: No server-side rendering required
- ✅ **Faster Loading**: Pre-rendered HTML loads instantly
- ✅ **Better SEO**: Search engines can index static HTML
- ✅ **Maintains Styling**: All Docsify themes and navigation preserved
- ✅ **Automated**: Runs as part of build process

## Dependencies

- `glob`: For file pattern matching (required)
- `docsify-cli`: For running the local server (documentation only)
- `puppeteer`: For headless browser automation (legacy script only)

## Output

Static files are generated in:
```
/public/documentation/static/
├── index.html
├── knirvchain.html
├── knirvgraph.html
├── knirvserver.html
└── ... (all documentation pages)
```

## Troubleshooting

**Server won't start:**
- Ensure docsify-cli is installed in `/public/documentation/`
- Check that port 3000 is available

**Missing pages:**
- Check console output for crawling errors
- Verify all internal links are properly formatted

**Styling issues:**
- Static CSS is injected to maintain layout
- Check browser console for missing assets

## Development

To modify the generation process:

1. Edit the respective script files
2. Test with `npm run docs:test-static`
3. Run full generation with `npm run docs:generate-static`
4. Deploy with `npm run build`
