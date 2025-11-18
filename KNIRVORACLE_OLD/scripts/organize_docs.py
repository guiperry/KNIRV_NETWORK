#!/usr/bin/env python3
"""
Documentation Organizer for KNIRVORACLE

This script processes the existing documentation in the docs/ folder
and transforms it into an organized, cohesive documentation system
in a new documentation/ folder, all written in markdown.

The script is designed to be idempotent - running it multiple times
will produce the same result as running it once.
"""

import os
import re
import shutil
from pathlib import Path

# Configuration
SCRIPT_DIR = Path(__file__).parent
ROOT_DIR = SCRIPT_DIR.parent
SOURCE_DIR = ROOT_DIR / 'docs'
TARGET_DIR = ROOT_DIR / 'documentation' / 'docsify'
README_PATH = ROOT_DIR / 'README.md'

# Documentation structure
STRUCTURE = {
    'index.md': {'title': 'KNIRVORACLE Documentation', 'content': ''},
    'getting-started': {
        'index.md': {'title': 'Getting Started', 'content': ''},
        'installation.md': {'title': 'Installation Guide', 'content': ''},
        'configuration.md': {'title': 'Configuration', 'content': ''},
        'quick-start.md': {'title': 'Quick Start Guide', 'content': ''}
    },
    'core-concepts': {
        'index.md': {'title': 'Core Concepts', 'content': ''},
        'blockchain.md': {'title': 'Blockchain Architecture', 'content': ''},
        'mcp.md': {'title': 'Model Context Protocol (MCP)', 'content': ''},
        'capabilities.md': {'title': 'Capabilities System', 'content': ''},
        'uri-scheme.md': {'title': 'URI Scheme', 'content': ''}
    },
    'protocols': {
        'index.md': {'title': 'Protocols', 'content': ''}
        # Will be populated from docs/protocols
    },
    'api-reference': {
        'index.md': {'title': 'API Reference', 'content': ''},
        'blockchain-api.md': {'title': 'Blockchain API', 'content': ''},
        'wallet-api.md': {'title': 'Wallet API', 'content': ''},
        'mcp-api.md': {'title': 'MCP API', 'content': ''}
    },
    'components': {
        'index.md': {'title': 'Components', 'content': ''},
        'agent-tunnel-registry.md': {'title': 'Agent Tunnel Registry', 'content': ''},
        'agent-bootnode-registry.md': {'title': 'Agent Bootnode Registry', 'content': ''},
        'agent-payment-gateway.md': {'title': 'Agent Payment Gateway', 'content': ''},
        'developer-portal.md': {'title': 'Developer Portal', 'content': ''},
        'altgui.md': {'title': 'Alternative GUI', 'content': ''}
    },
    'guides': {
        'index.md': {'title': 'Guides', 'content': ''},
        'running-a-node.md': {'title': 'Running a Node', 'content': ''},
        'developing-plugins.md': {'title': 'Developing Plugins', 'content': ''},
        'creating-capabilities.md': {'title': 'Creating Capabilities', 'content': ''}
    },
    'troubleshooting': {
        'index.md': {'title': 'Troubleshooting', 'content': ''}
        # Will be populated from docs/Troubleshooting
    },
    'sdk': {
        'index.md': {'title': 'SDK Documentation', 'content': ''}
        # Will be populated from docs/SDK
    }
}

# Category mapping for existing documentation
CATEGORY_MAPPING = {
    'protocols': {
        'folder': 'protocols',
        'pattern': r'.*\.md$'
    },
    'troubleshooting': {
        'folder': 'Troubleshooting',
        'pattern': r'.*\.md$'
    },
    'sdk': {
        'folder': 'SDK',
        'pattern': r'.*\.md$'
    },
    'core-concepts': {
        'folder': '',
        'pattern': r'(Agent_Focus|agent_inferencer).*\.md$'
    },
    'guides': {
        'folder': 'completedImplementations',
        'pattern': r'.*\.md$'
    }
}

# Content mapping for specific files
CONTENT_MAPPING = {
    'core-concepts/blockchain.md': ['protocols/Blockchain_README.md'],
    'core-concepts/mcp.md': ['protocols/context_record_scenarios.md'],
    'core-concepts/capabilities.md': ['protocols/Capabilities_Protocol.md'],
    'core-concepts/uri-scheme.md': ['protocols/URI_Generation_Protocol.md'],
    'api-reference/blockchain-api.md': ['protocols/Blockchain_README.md'],
    'api-reference/wallet-api.md': ['protocols/Blockchain_README.md'],
    'api-reference/mcp-api.md': ['protocols/Blockchain_README.md'],
    'getting-started/installation.md': ['protocols/Blockchain_README.md'],
    'getting-started/configuration.md': ['protocols/Blockchain_README.md'],
    'getting-started/quick-start.md': ['protocols/Blockchain_README.md'],
    'components/agent-tunnel-registry.md': ['protocols/tunnel_relay_implementation_summary.md'],
    'components/agent-bootnode-registry.md': ['protocols/Bootnode_Registry_Protocol.md'],
    'guides/running-a-node.md': ['protocols/Blockchain_README.md'],
    'guides/developing-plugins.md': ['protocols/Plugin_Updater_Protocol.md']
}

# Keywords to identify content for specific sections
KEYWORD_MAPPING = {
    'core-concepts/blockchain.md': ['blockchain', 'consensus', 'p2p', 'libp2p'],
    'core-concepts/mcp.md': ['mcp', 'model context protocol', 'contextrecord'],
    'core-concepts/capabilities.md': ['capabilities', 'plugins', 'tools', 'prompts'],
    'core-concepts/uri-scheme.md': ['uri', 'agent://', 'scheme'],
    'api-reference/blockchain-api.md': ['api', 'endpoints', 'http', 'get', 'post'],
    'api-reference/wallet-api.md': ['wallet', 'transaction', 'signing'],
    'api-reference/mcp-api.md': ['mcp', 'api', '/mcp/'],
    'getting-started/installation.md': ['prerequisites', 'building', 'install'],
    'getting-started/configuration.md': ['config', 'configuration', 'settings'],
    'getting-started/quick-start.md': ['running', 'getting started', 'quick'],
    'guides/running-a-node.md': ['running', 'node', 'startup'],
    'guides/developing-plugins.md': ['plugin', 'develop', 'create']
}


def create_directory_structure():
    """Creates the directory structure for the new documentation"""
    print('Creating directory structure...')
    
    # Create the main documentation directory
    if not os.path.exists(TARGET_DIR):
        os.makedirs(TARGET_DIR)
    
    # Create subdirectories and empty index files
    for dir_name, content in STRUCTURE.items():
        if isinstance(content, dict):
            # This is a directory or file
            if dir_name != 'index.md':
                # This is a directory
                dir_path = TARGET_DIR / dir_name
                if not os.path.exists(dir_path):
                    os.makedirs(dir_path)
                
                # Create files in this directory
                for file_name, file_content in content.items():
                    file_path = dir_path / file_name
                    if not os.path.exists(file_path):
                        initial_content = f"# {file_content['title']}\n\n{file_content['content']}"
                        with open(file_path, 'w') as f:
                            f.write(initial_content)
            else:
                # This is the root index.md
                file_path = TARGET_DIR / dir_name
                if not os.path.exists(file_path):
                    initial_content = f"# {content['title']}\n\n{content['content']}"
                    with open(file_path, 'w') as f:
                        f.write(initial_content)
    
    print('Directory structure created successfully.')


def process_readme():
    """Extracts content from README.md for the main index page"""
    print('Processing README.md...')
    
    try:
        with open(README_PATH, 'r') as f:
            content = f.read()
        
        # Extract relevant sections from README
        lines = content.split('\n')
        extracted_content = []
        in_relevant_section = False
        
        for line in lines:
            # Skip the title as we'll add our own
            if line.startswith('# KNIRVORACLE'):
                continue
            
            # Include content after the title
            if line.startswith('KNIRVORACLE is'):
                in_relevant_section = True
            
            if in_relevant_section:
                extracted_content.append(line)
            
            # Stop at the API Endpoints section or other technical details
            if line.startswith('## API Endpoints') or line.startswith('## Data Storage'):
                break
        
        # Update the main index.md
        index_path = TARGET_DIR / 'index.md'
        index_content = f"# KNIRVORACLE Documentation\n\n{''.join(extracted_content)}\n\n## Documentation Sections\n\n"
        
        # Add links to main sections
        section_links = []
        for dir_name, content in STRUCTURE.items():
            if dir_name != 'index.md' and isinstance(content, dict):
                title = content['index.md']['title']
                section_links.append(f"- [{title}](./{dir_name}/)")
        
        with open(index_path, 'w') as f:
            f.write(index_content + '\n'.join(section_links))
        
        print('README processed and index.md updated.')
        
    except Exception as e:
        print(f'Error processing README: {e}')


def process_protocols():
    """Processes protocol documentation files"""
    print('Processing protocol documentation...')
    
    try:
        protocols_dir = SOURCE_DIR / 'protocols'
        if not os.path.exists(protocols_dir):
            print(f'Protocols directory {protocols_dir} does not exist. Skipping.')
            return
            
        files = os.listdir(protocols_dir)
        
        # Create entries in the protocols section
        for file in files:
            if file.endswith('.md'):
                source_path = protocols_dir / file
                target_path = TARGET_DIR / 'protocols' / file
                
                # Read the protocol file
                with open(source_path, 'r') as f:
                    content = f.read()
                
                # Extract title from the first line
                lines = content.split('\n')
                title = file.replace('.md', '').replace('_', ' ')
                
                if lines[0].startswith('# '):
                    title = lines[0][2:]
                
                # Add to the structure
                if 'protocols' in STRUCTURE:
                    STRUCTURE['protocols'][file] = {'title': title, 'content': ''}
                
                # Copy the file
                with open(target_path, 'w') as f:
                    f.write(content)
                
                # Update the protocols index
                index_path = TARGET_DIR / 'protocols' / 'index.md'
                with open(index_path, 'r') as f:
                    index_content = f.read()
                
                if f"- [{title}](./{file})" not in index_content:
                    with open(index_path, 'a') as f:
                        f.write(f"- [{title}](./{file})\n")
        
        print('Protocol documentation processed.')
        
    except Exception as e:
        print(f'Error processing protocols: {e}')


def process_category_mappings():
    """Processes documentation based on category mappings"""
    print('Processing category mappings...')
    
    for category, mapping in CATEGORY_MAPPING.items():
        try:
            source_dir = SOURCE_DIR / mapping['folder']
            
            # Skip if the source directory doesn't exist
            if not os.path.exists(source_dir):
                print(f'Source directory {source_dir} does not exist. Skipping.')
                continue
            
            files = os.listdir(source_dir)
            pattern = re.compile(mapping['pattern'])
            
            for file in files:
                if pattern.match(file):
                    source_path = source_dir / file
                    
                    # Check if it's a directory
                    if os.path.isdir(source_path):
                        continue
                    
                    # Read the file
                    with open(source_path, 'r') as f:
                        content = f.read()
                    
                    # Extract title from the first line
                    lines = content.split('\n')
                    title = file.replace('.md', '').replace('_', ' ')
                    
                    if lines[0].startswith('# '):
                        title = lines[0][2:]
                    
                    # Create target file name
                    target_file = re.sub(r'[^a-z0-9]+', '-', file.lower())
                    target_path = TARGET_DIR / category / target_file
                    
                    # Add to the structure if not already there
                    if category in STRUCTURE and target_file not in STRUCTURE[category]:
                        STRUCTURE[category][target_file] = {'title': title, 'content': ''}
                    
                    # Copy the file
                    with open(target_path, 'w') as f:
                        f.write(content)
                    
                    # Update the category index
                    index_path = TARGET_DIR / category / 'index.md'
                    with open(index_path, 'r') as f:
                        index_content = f.read()
                    
                    if f"- [{title}](./{target_file})" not in index_content:
                        with open(index_path, 'a') as f:
                            f.write(f"- [{title}](./{target_file})\n")
            
        except Exception as e:
            print(f'Error processing category {category}: {e}')
    
    print('Category mappings processed.')


def extract_relevant_sections(content, keywords, target_file):
    """Extracts relevant sections from content based on keywords"""
    # Split content into sections by headings
    sections = re.split(r'^##\s+', content, flags=re.MULTILINE)
    
    relevant_content = []
    found_relevant = False
    
    # Check each section for keywords
    for i, section in enumerate(sections):
        # Skip the first section if it contains the title
        if i == 0 and section.startswith('# '):
            continue
        
        # Check if this section contains any of the keywords
        contains_keyword = any(keyword.lower() in section.lower() for keyword in keywords)
        
        # Special handling for specific target files
        if target_file == 'api-reference/blockchain-api.md' and 'API Endpoints' in section:
            relevant_content.append('## API Endpoints\n\n' + section)
            found_relevant = True
        elif target_file == 'getting-started/installation.md' and ('Prerequisites' in section or 'Building' in section):
            relevant_content.append('## ' + section)
            found_relevant = True
        elif target_file == 'getting-started/configuration.md' and 'Configuration' in section:
            relevant_content.append('## ' + section)
            found_relevant = True
        elif target_file == 'guides/running-a-node.md' and 'Running' in section:
            relevant_content.append('## ' + section)
            found_relevant = True
        elif contains_keyword:
            relevant_content.append('## ' + section)
            found_relevant = True
    
    return '\n\n'.join(relevant_content) if found_relevant else None


def process_content_mappings():
    """Processes specific content mappings"""
    print('Processing content mappings...')
    
    for target_file, source_paths in CONTENT_MAPPING.items():
        try:
            target_path = TARGET_DIR / target_file
            
            # Read the target file to get its current content
            with open(target_path, 'r') as f:
                target_content = f.read()
            
            # Process each source file
            for source_path in source_paths:
                full_source_path = SOURCE_DIR / source_path
                
                # Skip if the source file doesn't exist
                if not os.path.exists(full_source_path):
                    print(f'Source file {full_source_path} does not exist. Skipping.')
                    continue
                
                # Read the source file
                with open(full_source_path, 'r') as f:
                    source_content = f.read()
                
                # Extract relevant content based on keywords
                keywords = KEYWORD_MAPPING.get(target_file, [])
                sections = extract_relevant_sections(source_content, keywords, target_file)
                
                # Append the extracted content to the target file
                if sections and sections not in target_content:
                    target_content += '\n\n' + sections
            
            # Write the updated content back to the target file
            with open(target_path, 'w') as f:
                f.write(target_content)
            
        except Exception as e:
            print(f'Error processing content mapping for {target_file}: {e}')
    
    print('Content mappings processed.')


def update_section_indexes():
    """Updates section index files with links to their content"""
    print('Updating section index files...')
    
    for dir_name, content in STRUCTURE.items():
        if dir_name != 'index.md' and isinstance(content, dict):
            index_path = TARGET_DIR / dir_name / 'index.md'
            index_content = f"# {content['index.md']['title']}\n\n"
            
            # Add description based on the section
            if dir_name == 'getting-started':
                index_content += 'This section helps you get up and running with KNIRVORACLE quickly.\n\n'
            elif dir_name == 'core-concepts':
                index_content += 'Learn about the fundamental concepts that power KNIRVORACLE.\n\n'
            elif dir_name == 'protocols':
                index_content += 'Detailed documentation of the protocols used in KNIRVORACLE.\n\n'
            elif dir_name == 'api-reference':
                index_content += 'Complete reference for all KNIRVORACLE APIs.\n\n'
            elif dir_name == 'components':
                index_content += 'Information about the various components that make up the KNIRVORACLE ecosystem.\n\n'
            elif dir_name == 'guides':
                index_content += 'Step-by-step guides for common KNIRVORACLE tasks.\n\n'
            elif dir_name == 'troubleshooting':
                index_content += 'Solutions to common problems and troubleshooting tips.\n\n'
            elif dir_name == 'sdk':
                index_content += 'Documentation for the KNIRVORACLE SDK.\n\n'
            
            # Add links to all files in this section
            index_content += '## Contents\n\n'
            
            for file_name, file_content in content.items():
                if file_name != 'index.md':
                    index_content += f"- [{file_content['title']}](./{file_name})\n"
            
            # Write the updated index
            with open(index_path, 'w') as f:
                f.write(index_content)
    
    print('Section index files updated.')


def create_sidebar():
    """Creates a navigation sidebar file"""
    print('Creating sidebar navigation...')
    
    sidebar_path = TARGET_DIR / '_sidebar.md'
    sidebar_content = '# KNIRVORACLE Docs\n\n'
    
    # Add link to home
    sidebar_content += '- [Home](./)\n'
    
    # Add links to main sections
    for dir_name, content in STRUCTURE.items():
        if dir_name != 'index.md' and isinstance(content, dict):
            sidebar_content += f"- [{content['index.md']['title']}](./{dir_name}/)\n"
            
            # Add subsections
            for file_name, file_content in content.items():
                if file_name != 'index.md':
                    sidebar_content += f"  - [{file_content['title']}](./{dir_name}/{file_name})\n"
    
    # Write the sidebar file
    with open(sidebar_path, 'w') as f:
        f.write(sidebar_content)
    
    print('Sidebar navigation created.')


def create_docsify_index():
    """Creates the index.html file for Docsify with search enabled"""
    print('Creating Docsify index.html with search...')
    
    index_html_content = """<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>KNIRVORACLE Documentation</title>
  <meta http-equiv="X-UA-Compatible" content="IE=edge,chrome=1" />
  <meta name="description" content="KNIRVORACLE Documentation">
  <meta name="viewport" content="width=device-width, initial-scale=1.0, minimum-scale=1.0">
  <link rel="stylesheet" href="//cdn.jsdelivr.net/npm/docsify@4/lib/themes/vue.css">
</head>
<body>
  <div id="app"></div>
  <script>
    window.$docsify = {
      name: 'KNIRVORACLE',
      repo: 'https://github.com/gperry/KNIRVORACLE',
      loadSidebar: true,
      subMaxLevel: 3,
      search: 'auto', // Enables the search plugin
    }
  </script>
  <!-- Docsify Core -->
  <script src="//cdn.jsdelivr.net/npm/docsify@4"></script>
  <!-- Docsify Search Plugin -->
  <script src="//cdn.jsdelivr.net/npm/docsify/lib/plugins/search.min.js"></script>
</body>
</html>"""
    
    index_path = TARGET_DIR / 'index.html'
    with open(index_path, 'w') as f:
        f.write(index_html_content)
    
    print('Docsify index.html created successfully.')


def clean_target_directory():
    """Cleans the target directory to ensure idempotency"""
    print('Cleaning target directory to ensure idempotency...')
    
    # Don't delete the entire directory as it might contain custom files
    # Instead, only delete files that we know we'll regenerate
    
    # Process each section in our structure
    for dir_name, content in STRUCTURE.items():
        if dir_name != 'index.md' and isinstance(content, dict):
            dir_path = TARGET_DIR / dir_name
            
            # Skip if directory doesn't exist
            if not os.path.exists(dir_path):
                continue
            
            # Clean index file
            index_path = dir_path / 'index.md'
            if os.path.exists(index_path):
                with open(index_path, 'w') as f:
                    f.write(f"# {content['index.md']['title']}\n\n{content['index.md']['content']}")
            
            # Clean other files defined in our structure
            for file_name, file_content in content.items():
                if file_name != 'index.md':
                    file_path = dir_path / file_name
                    if os.path.exists(file_path):
                        with open(file_path, 'w') as f:
                            f.write(f"# {file_content['title']}\n\n{file_content['content']}")
    
    # Clean main index and sidebar
    index_path = TARGET_DIR / 'index.md'
    if os.path.exists(index_path):
        with open(index_path, 'w') as f:
            f.write(f"# {STRUCTURE['index.md']['title']}\n\n{STRUCTURE['index.md']['content']}")
    
    sidebar_path = TARGET_DIR / '_sidebar.md'
    if os.path.exists(sidebar_path):
        with open(sidebar_path, 'w') as f:
            f.write('# KNIRVORACLE Docs\n\n')
    
    print('Target directory cleaned.')


def main():
    """Main function to run the documentation organization process"""
    try:
        print('Starting documentation organization process...')
        
        # Create the directory structure
        create_directory_structure()
        
        # Clean target directory to ensure idempotency
        clean_target_directory()
        
        # Process the README for the main index
        process_readme()
        
        # Process protocol documentation
        process_protocols()
        
        # Process category mappings
        process_category_mappings()
        
        # Process content mappings
        process_content_mappings()
        
        # Update section index files
        update_section_indexes()
        
        # Create sidebar navigation
        create_sidebar()
        
        # Create Docsify index.html for search functionality
        create_docsify_index()
        
        print('Documentation organization complete! The new documentation is available in the "documentation/docsify" directory.')
        
    except Exception as e:
        print(f'Error organizing documentation: {e}')


if __name__ == '__main__':
    main()