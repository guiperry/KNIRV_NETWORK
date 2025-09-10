#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const glob = require('glob');

class DocsifyLinkUpdater {
  constructor() {
    this.publicDir = path.join(__dirname, '../public');
    this.staticDocsPath = '/documentation/static';
  }

  async updateLinks() {
    console.log('🔗 Updating docsify links to point to static version...');

    // Find all HTML files in public directory
    const htmlFiles = glob.sync('**/*.html', {
      cwd: this.publicDir,
      ignore: ['documentation/static/**/*', 'documentation/docsify/**/*']
    });

    // Also find markdown files in docsify directory to update source files
    const markdownFiles = glob.sync('documentation/docsify/**/*.md', {
      cwd: this.publicDir
    });

    for (const file of htmlFiles) {
      await this.processFile(path.join(this.publicDir, file));
    }

    for (const file of markdownFiles) {
      await this.processFile(path.join(this.publicDir, file));
    }

    console.log(`✅ Updated ${htmlFiles.length + markdownFiles.length} files`);
  }

  async processFile(filePath) {
    try {
      let content = fs.readFileSync(filePath, 'utf8');
      let modified = false;

      // Update links to docsify documentation (both absolute and relative paths)
      const docsifyLinkRegex = /href="([^"]*documentation\/docsify[^"]*)"/g;
      content = content.replace(docsifyLinkRegex, (match, url) => {
        // Convert docsify URL to static URL
        let staticUrl = url.replace(/documentation\/docsify/, 'documentation/static');

        // Remove docsify hash fragment (#/) if present
        staticUrl = staticUrl.replace('/#/', '/');

        // Ensure .html extension for non-directory paths
        if (!staticUrl.endsWith('/') && !staticUrl.endsWith('.html')) {
          staticUrl += '.html';
        }

        modified = true;
        return `href="${staticUrl}"`;
      });

      // Update any remaining docsify references (including hash fragments, both absolute and relative)
      const docsifyRefRegex = /documentation\/docsify\/#?\//g;
      content = content.replace(docsifyRefRegex, (match) => {
        modified = true;
        return `documentation/static/`;
      });

      // Handle standalone docsify hash references (both absolute and relative)
      const docsifyHashRegex = /documentation\/docsify\/#\/([^"'\s]+)/g;
      content = content.replace(docsifyHashRegex, (match, path) => {
        modified = true;
        let staticPath = `documentation/static/${path}`;
        if (!staticPath.endsWith('.html')) {
          staticPath += '.html';
        }
        return staticPath;
      });

      // Handle direct docsify hash references (like href="#/legal/...")
      const directHashRegex = /href="#\/([^"]+)"/g;
      content = content.replace(directHashRegex, (match, path) => {
        modified = true;
        let staticPath = `documentation/static/${path}`;
        if (!staticPath.endsWith('.html')) {
          staticPath += '.html';
        }
        return `href="${staticPath}"`;
      });

      if (modified) {
        fs.writeFileSync(filePath, content, 'utf8');
        console.log(`📝 Updated: ${path.relative(this.publicDir, filePath)}`);
      }

    } catch (error) {
      console.error(`❌ Error processing ${filePath}:`, error.message);
    }
  }

  async run() {
    try {
      await this.updateLinks();
      console.log('🎉 Link update complete!');
    } catch (error) {
      console.error('❌ Error:', error);
      process.exit(1);
    }
  }
}

// Run if called directly
if (require.main === module) {
  const updater = new DocsifyLinkUpdater();
  updater.run();
}

module.exports = DocsifyLinkUpdater;
