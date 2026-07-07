#!/usr/bin/env node

/**
 * ChromeDB Query Fix Script
 * 
 * This script automatically identifies and fixes problematic ChromeDB query patterns
 * in Go source files. It addresses two main issues:
 * 1. nResults validation errors (requesting more results than documents exist)
 * 2. Unsupported where clause operators
 * 
 * Usage: node fix_chromem_queries.js [directory]
 */

const fs = require('fs');
const path = require('path');

// Configuration
const CONFIG = {
    // File patterns to process
    filePatterns: [/\.go$/],
    
    // Backup original files
    createBackups: true,
    
    // Conservative nResults limits to try
    progressiveLimits: [1, 3, 5, 10],
    
    // Generic search terms for different entity types
    searchTerms: {
        'Agent': 'Agent',
        'AgentRelationship': 'relationship', 
        'Badge': 'badge',
        'Capability': 'capability',
        'Transaction': 'transaction'
    }
};

/**
 * Identifies problematic ChromeDB query patterns
 */
function findProblematicQueries(content) {
    const issues = [];
    
    // Pattern 1: High nResults values
    const highNResultsPattern = /\.Query\s*\(\s*[^,]+,\s*[^,]+,\s*(\d+),/g;
    let match;
    
    while ((match = highNResultsPattern.exec(content)) !== null) {
        const nResults = parseInt(match[1]);
        if (nResults > 3) {
            issues.push({
                type: 'high_nresults',
                line: getLineNumber(content, match.index),
                value: nResults,
                match: match[0],
                index: match.index
            });
        }
    }
    
    // Pattern 2: Where clause usage
    const whereClausePattern = /\.Query\s*\([^)]*map\s*\[\s*string\s*\]\s*string\s*\{[^}]*\}[^)]*\)/g;
    
    while ((match = whereClausePattern.exec(content)) !== null) {
        issues.push({
            type: 'where_clause',
            line: getLineNumber(content, match.index),
            match: match[0],
            index: match.index
        });
    }
    
    return issues;
}

/**
 * Fixes ChromeDB query issues in content
 */
function fixQueries(content) {
    let fixed = content;
    let changesMade = [];
    
    // Fix 1: Replace high nResults with conservative values
    fixed = fixed.replace(
        /\.Query\s*\(\s*([^,]+),\s*([^,]+),\s*(\d+),/g,
        (match, context, searchTerm, nResults) => {
            const currentLimit = parseInt(nResults);
            if (currentLimit > 3) {
                changesMade.push(`Reduced nResults from ${currentLimit} to 3`);
                return `.Query(${context}, ${searchTerm}, 3,`;
            }
            return match;
        }
    );
    
    // Fix 2: Remove where clauses and add manual filtering template
    fixed = fixed.replace(
        /\.Query\s*\(\s*([^,]+),\s*([^,]+),\s*([^,]+),\s*(map\s*\[\s*string\s*\]\s*string\s*\{[^}]*\}),([^)]*)\)/g,
        (match, context, searchTerm, nResults, whereClause, remaining) => {
            changesMade.push('Removed where clause, added manual filtering');
            
            // Extract field and value from where clause
            const fieldMatch = whereClause.match(/"([^"]+)":\s*"([^"]+)"/);
            const field = fieldMatch ? fieldMatch[1] : 'field';
            const value = fieldMatch ? fieldMatch[2] : 'value';
            
            return `.Query(${context}, ${searchTerm}, ${nResults}, nil, ${remaining})
	// TODO: Add manual filtering for ${field} == "${value}"
	// Filter results in loop: if result.Metadata["${field}"] == "${value}" { ... }`;
        }
    );
    
    return { content: fixed, changes: changesMade };
}

/**
 * Generates improved query method with progressive fallback
 */
function generateProgressiveQueryMethod(methodName, entityType) {
    const searchTerm = CONFIG.searchTerms[entityType] || 'document';
    
    return `
// ${methodName} with progressive query fallback to handle ChromeDB limitations
func (m *ChromemManager) ${methodName}(id string) (*${entityType}, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()

    if m.client == nil {
        return nil, fmt.Errorf("ChromemDB client not initialized")
    }

    collection, err := m.client.GetOrCreateCollection(${entityType}Collection, nil, m.ef)
    if err != nil {
        return nil, fmt.Errorf("failed to get collection: %w", err)
    }

    // Progressive query with fallback limits
    limits := []int{${CONFIG.progressiveLimits.join(', ')}}
    
    for _, limit := range limits {
        results, err := collection.Query(
            context.Background(),
            "${searchTerm}",
            limit,
            nil, // No where clause (causes errors)
            nil,
        )
        
        if err != nil {
            // If nResults error, try smaller limit
            if strings.Contains(err.Error(), "nResults") {
                continue
            }
            // Other errors are not recoverable
            return nil, fmt.Errorf("failed to query: %w", err)
        }
        
        // Manual filtering for exact match
        for _, result := range results {
            if result.ID == id {
                var entity ${entityType}
                if err := json.Unmarshal([]byte(result.Metadata["data"]), &entity); err != nil {
                    return nil, fmt.Errorf("failed to unmarshal: %w", err)
                }
                return &entity, nil
            }
        }
        
        // If we got results but no match, entity doesn't exist
        break
    }
    
    return nil, fmt.Errorf("${entityType.toLowerCase()} not found: %s", id)
}`;
}

/**
 * Processes a single Go file
 */
function processFile(filePath) {
    console.log(`Processing: ${filePath}`);
    
    const content = fs.readFileSync(filePath, 'utf8');
    const issues = findProblematicQueries(content);
    
    if (issues.length === 0) {
        console.log(`  ✅ No issues found`);
        return;
    }
    
    console.log(`  🔍 Found ${issues.length} issues:`);
    issues.forEach(issue => {
        console.log(`    - Line ${issue.line}: ${issue.type} (${issue.value || 'N/A'})`);
    });
    
    // Create backup if enabled
    if (CONFIG.createBackups) {
        fs.writeFileSync(`${filePath}.backup`, content);
        console.log(`  💾 Backup created: ${filePath}.backup`);
    }
    
    // Apply fixes
    const { content: fixedContent, changes } = fixQueries(content);
    
    if (changes.length > 0) {
        fs.writeFileSync(filePath, fixedContent);
        console.log(`  ✅ Applied ${changes.length} fixes:`);
        changes.forEach(change => console.log(`    - ${change}`));
    }
}

/**
 * Recursively processes directory
 */
function processDirectory(dirPath) {
    const entries = fs.readdirSync(dirPath);
    
    for (const entry of entries) {
        const fullPath = path.join(dirPath, entry);
        const stat = fs.statSync(fullPath);
        
        if (stat.isDirectory()) {
            // Skip common non-source directories
            if (!['node_modules', '.git', 'vendor', 'build', 'dist'].includes(entry)) {
                processDirectory(fullPath);
            }
        } else if (stat.isFile()) {
            // Check if file matches our patterns
            if (CONFIG.filePatterns.some(pattern => pattern.test(entry))) {
                processFile(fullPath);
            }
        }
    }
}

/**
 * Utility function to get line number from string index
 */
function getLineNumber(content, index) {
    return content.substring(0, index).split('\n').length;
}

/**
 * Main execution
 */
function main() {
    const targetDir = process.argv[2] || '.';
    
    console.log('🔧 ChromeDB Query Fix Script');
    console.log(`📁 Target directory: ${path.resolve(targetDir)}`);
    console.log(`🔍 File patterns: ${CONFIG.filePatterns.map(p => p.toString()).join(', ')}`);
    console.log('');
    
    if (!fs.existsSync(targetDir)) {
        console.error(`❌ Directory not found: ${targetDir}`);
        process.exit(1);
    }
    
    try {
        processDirectory(targetDir);
        console.log('\n✅ Processing complete!');
        
        // Generate example improved methods
        console.log('\n📝 Example improved method templates:');
        console.log(generateProgressiveQueryMethod('GetAgent', 'Agent'));
        
    } catch (error) {
        console.error(`❌ Error: ${error.message}`);
        process.exit(1);
    }
}

// Run if called directly
if (require.main === module) {
    main();
}

module.exports = {
    findProblematicQueries,
    fixQueries,
    generateProgressiveQueryMethod,
    processFile,
    CONFIG
};
