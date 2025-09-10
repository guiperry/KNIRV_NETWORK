#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const glob = require('glob');

class SimpleStaticGenerator {
  constructor() {
    this.docsifyDir = path.join(__dirname, '../public/documentation/docsify');
    this.staticDir = path.join(__dirname, '../public/documentation/static');
    this.templatePath = path.join(this.docsifyDir, 'index.html');
  }

  async init() {
    console.log('🚀 Starting Simple Static Generator...');
    
    // Ensure static directory exists
    if (!fs.existsSync(this.staticDir)) {
      fs.mkdirSync(this.staticDir, { recursive: true });
    }

    // Read the template
    this.template = fs.readFileSync(this.templatePath, 'utf8');
    console.log('✅ Template loaded');
  }

  async generateStaticSite() {
    console.log('📄 Generating static pages...');
    
    // Find all markdown files
    const markdownFiles = glob.sync('**/*.md', { 
      cwd: this.docsifyDir,
      ignore: ['node_modules/**/*']
    });

    console.log(`📚 Found ${markdownFiles.length} markdown files`);

    // Generate index page
    await this.generateIndexPage();

    // Generate pages for each markdown file
    for (const mdFile of markdownFiles) {
      await this.generatePageFromMarkdown(mdFile);
    }

    console.log(`✅ Generated ${markdownFiles.length + 1} static pages`);
  }

  async generateIndexPage() {
    console.log('🏠 Generating index page...');
    
    // Read the sidebar to get navigation
    const sidebarPath = path.join(this.docsifyDir, '_sidebar.md');
    let sidebarContent = '';
    
    if (fs.existsSync(sidebarPath)) {
      sidebarContent = fs.readFileSync(sidebarPath, 'utf8');
    }

    // Read the main README
    const readmePath = path.join(this.docsifyDir, 'README.md');
    let mainContent = '<h1>KNIRV Network Documentation</h1><p>Welcome to the KNIRV Network documentation.</p>';
    
    if (fs.existsSync(readmePath)) {
      mainContent = this.convertMarkdownToHtml(fs.readFileSync(readmePath, 'utf8'));
    }

    const html = this.createStaticHtml('KNIRV Network Documentation', mainContent, sidebarContent);
    
    const indexPath = path.join(this.staticDir, 'index.html');
    fs.writeFileSync(indexPath, html, 'utf8');
    console.log(`💾 Saved: index.html`);
  }

  async generatePageFromMarkdown(mdFile) {
    const mdPath = path.join(this.docsifyDir, mdFile);
    const content = fs.readFileSync(mdPath, 'utf8');
    
    // Convert markdown to HTML (basic conversion)
    const htmlContent = this.convertMarkdownToHtml(content);
    
    // Get title from first heading or filename
    const title = this.extractTitle(content) || path.basename(mdFile, '.md');
    
    // Read sidebar
    const sidebarPath = path.join(this.docsifyDir, '_sidebar.md');
    let sidebarContent = '';
    if (fs.existsSync(sidebarPath)) {
      sidebarContent = fs.readFileSync(sidebarPath, 'utf8');
    }

    const html = this.createStaticHtml(title, htmlContent, sidebarContent);
    
    // Create output path
    const outputPath = mdFile.replace(/\.md$/, '.html');
    const fullOutputPath = path.join(this.staticDir, outputPath);
    
    // Ensure directory exists
    const dir = path.dirname(fullOutputPath);
    if (!fs.existsSync(dir)) {
      fs.mkdirSync(dir, { recursive: true });
    }
    
    fs.writeFileSync(fullOutputPath, html, 'utf8');
    console.log(`💾 Saved: ${outputPath}`);
  }

  convertMarkdownToHtml(markdown) {
    // Basic markdown to HTML conversion
    let html = markdown
      // Headers
      .replace(/^### (.*$)/gim, '<h3>$1</h3>')
      .replace(/^## (.*$)/gim, '<h2>$1</h2>')
      .replace(/^# (.*$)/gim, '<h1>$1</h1>')
      // Bold
      .replace(/\*\*(.*)\*\*/gim, '<strong>$1</strong>')
      // Italic
      .replace(/\*(.*)\*/gim, '<em>$1</em>')
      // Links
      .replace(/\[([^\]]*)\]\(([^\)]*)\)/gim, '<a href="$2">$1</a>')
      // Code blocks
      .replace(/```([^`]*)```/gim, '<pre><code>$1</code></pre>')
      // Inline code
      .replace(/`([^`]*)`/gim, '<code>$1</code>')
      // Line breaks
      .replace(/\n/gim, '<br>');

    return html;
  }

  extractTitle(content) {
    const match = content.match(/^#\s+(.*)$/m);
    return match ? match[1] : null;
  }

  createStaticHtml(title, content, sidebar) {
    // Convert sidebar markdown to HTML navigation
    const sidebarHtml = this.convertSidebarToHtml(sidebar);
    
    return `<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>${title} - KNIRV Network</title>
  <style>
    body {
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
      margin: 0;
      padding: 0;
      background-color: #0a0a0a;
      color: #e0e0e0;
      line-height: 1.6;
    }
    .container {
      display: flex;
      min-height: 100vh;
    }
    .sidebar {
      width: 300px;
      background-color: #1a1a2e;
      border-right: 1px solid #333;
      padding: 20px;
      padding-bottom: 120px;
      overflow-y: auto;
      position: fixed;
      height: 100vh;
      box-sizing: border-box;
    }
    .content {
      flex: 1;
      margin-left: 300px;
      padding: 40px;
      max-width: 800px;
    }
    .sidebar h1, .sidebar h2, .sidebar h3 {
      color: #4a9eff;
      margin-top: 30px;
      margin-bottom: 10px;
    }
    .sidebar a {
      color: #e0e0e0;
      text-decoration: none;
      display: block;
      padding: 5px 0;
      border-left: 3px solid transparent;
      padding-left: 10px;
    }
    .sidebar a:hover {
      color: #4a9eff;
      border-left-color: #4a9eff;
    }
    .nav-section {
      margin-bottom: 20px;
    }
    .sidebar-footer {
      position: absolute;
      bottom: 20px;
      left: 20px;
      right: 20px;
    }
    .sidebar-footer hr {
      border: 1px solid #333;
      margin: 20px 0;
    }
    .sidebar-footer p {
      font-size: 0.9em;
      color: #888;
      text-align: center;
      margin: 0;
    }
    .sidebar-footer a {
      color: #4a9eff !important;
      border: none !important;
      padding: 0 !important;
    }
    .content h1, .content h2, .content h3 {
      color: #4a9eff;
    }
    .content a {
      color: #4a9eff;
    }
    .content a:hover {
      color: #6fb5ff;
    }
    .content code {
      background-color: #16213e;
      padding: 2px 6px;
      border-radius: 3px;
      font-family: 'Monaco', 'Consolas', monospace;
    }
    .content pre {
      background-color: #16213e;
      padding: 15px;
      border-radius: 5px;
      overflow-x: auto;
    }
    .content pre code {
      background: none;
      padding: 0;
    }
    @media (max-width: 768px) {
      .sidebar {
        transform: translateX(-300px);
        transition: transform 0.3s ease;
      }
      .content {
        margin-left: 0;
      }
    }
  </style>
</head>
<body>
  <div class="container">
    <div class="sidebar">
      <h1>KNIRV Network</h1>
      ${sidebarHtml}
    </div>
    <div class="content">
      ${content}
    </div>
  </div>
</body>
</html>`;
  }

  convertSidebarToHtml(sidebar) {
    if (!sidebar) return this.getDefaultSidebar();

    // Clean up the sidebar content first
    let cleanSidebar = sidebar
      .replace(/<div class="sidebar-footer">[\s\S]*?<\/div>/gim, '') // Remove existing footer
      .trim();

    // Convert markdown sidebar to proper HTML navigation
    let html = cleanSidebar
      .split('\n')
      .map(line => {
        line = line.trim();

        // Handle headings
        if (line.startsWith('## ')) {
          return `<h2>${line.substring(3)}</h2>`;
        }

        // Handle list items with links
        if (line.startsWith('* [')) {
          const match = line.match(/^\* \[(.*?)\]\((.*?)\)$/);
          if (match) {
            const [, text, url] = match;
            let staticUrl = url;

            // Handle different URL patterns
            if (url.endsWith('/')) {
              // Directory links - point to index.html
              staticUrl = url + 'index.html';
            } else if (url.endsWith('.md')) {
              // Markdown files - convert to .html
              staticUrl = url.replace(/\.md$/, '.html');
            } else if (!url.includes('.')) {
              // No extension - assume it's a directory or needs .html
              staticUrl = url + '.html';
            }

            return `<a href="${staticUrl}">${text}</a>`;
          }
        }

        // Skip empty lines and other content
        if (line === '' || line.startsWith('#') || line.startsWith('---')) {
          return '';
        }

        return line;
      })
      .filter(line => line !== '')
      .join('\n');

    // Wrap in proper navigation structure
    html = `<div class="nav-section">${html}</div>`;

    // Add footer
    html += this.getSidebarFooter();

    return html;
  }

  getDefaultSidebar() {
    return `
      <div class="nav-section">
        <h2>📚 Documentation</h2>
        <a href="index.html">🏠 Home</a>
        <a href="whitepapers/index.html">📄 Whitepapers</a>
      </div>
      ${this.getSidebarFooter()}
    `;
  }

  getSidebarFooter() {
    return `
      <div class="sidebar-footer">
        <hr style="border: 1px solid #333; margin: 20px 0;">
        <p style="font-size: 0.9em; color: #888; text-align: center;">
          © 2025 KNIRV Network<br>
          <a href="https://knirv.com" style="color: #4a9eff;">knirv.com</a>
        </p>
      </div>
    `;
  }

  async run() {
    try {
      await this.init();
      await this.generateStaticSite();
      console.log('🎉 Static site generation complete!');
      console.log(`📁 Static files saved to: ${this.staticDir}`);
    } catch (error) {
      console.error('❌ Error:', error);
      process.exit(1);
    }
  }
}

// Run if called directly
if (require.main === module) {
  const generator = new SimpleStaticGenerator();
  generator.run();
}

module.exports = SimpleStaticGenerator;
