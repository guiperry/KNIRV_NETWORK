#!/usr/bin/env node

import fs from 'fs';
import path from 'path';
import { glob } from 'glob';
import { fileURLToPath } from 'url';

class DocsifyLinkUpdater {
  constructor() {
    const __filename = fileURLToPath(import.meta.url);
    const __dirname = path.dirname(__filename);
    this.publicDir = path.join(__dirname, '../public');
    this.staticDocsPath = '/documentation/static';
  }

  // Function to properly convert file paths to HTML extensions
  convertToHtmlPath(filePath) {
    // Fix .md.html to just .html (this is the main issue we're solving)
    if (filePath.endsWith('.md.html')) {
      return filePath.replace(/\.md\.html$/, '.html');
    }
    // Remove .md extension if present, then add .html
    if (filePath.endsWith('.md')) {
      return filePath.replace(/\.md$/, '.html');
    }
    // If it doesn't end with .html, add .html
    if (!filePath.endsWith('.html')) {
      return filePath + '.html';
    }
    // If it already ends with .html, return as is
    return filePath;
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

        // Ensure proper .html extension for non-directory paths
        if (!staticUrl.endsWith('/')) {
          // Extract the path part after the last slash
          const pathParts = staticUrl.split('/');
          const lastPart = pathParts[pathParts.length - 1];
          if (lastPart && !lastPart.includes('.')) {
            // If no extension, add .html
            staticUrl += '.html';
          } else if (lastPart) {
            // Use the convertToHtmlPath function to handle .md -> .html conversion
            const convertedPart = this.convertToHtmlPath(lastPart);
            pathParts[pathParts.length - 1] = convertedPart;
            staticUrl = pathParts.join('/');
          }
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

      // Handle absolute documentation/static paths that need to be made relative
      const absoluteStaticRegex = /href="documentation\/static\/([^"]+)"/g;

      // Calculate relative path based on current file location (outside the regex callback)
      const relativePath = path.relative(this.publicDir, filePath);
      const isStaticFile = relativePath.includes('documentation/static/');
      let relativePrefix = '';

      if (isStaticFile) {
        // For static files, calculate relative path to static root
        const staticRelativePath = relativePath.replace('documentation/static/', '');
        const staticDepth = staticRelativePath.split('/').filter(p => p && p !== '.').length - 1; // -1 because we don't count the file itself
        relativePrefix = staticDepth > 0 ? '../'.repeat(staticDepth) : '';
      }

      content = content.replace(directHashRegex, (match, urlPath) => {
        modified = true;
        let staticPath = this.convertToHtmlPath(urlPath);

        if (isStaticFile) {
          // For static files, use relative path
          staticPath = relativePrefix + staticPath;
        } else {
          // For source files, use absolute path from documentation root
          staticPath = `documentation/static/${staticPath}`;
        }

        return `href="${staticPath}"`;
      });

      // Handle absolute documentation/static paths
      content = content.replace(absoluteStaticRegex, (match, urlPath) => {
        modified = true;
        let staticPath = this.convertToHtmlPath(urlPath);

        if (isStaticFile) {
          // For static files, convert to relative path
          staticPath = relativePrefix + staticPath;
        } else {
          // For source files, keep the absolute path
          staticPath = `documentation/static/${staticPath}`;
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
if (import.meta.url === `file://${process.argv[1]}`) {
  const updater = new DocsifyLinkUpdater();
  updater.run();
}

export default DocsifyLinkUpdater;
