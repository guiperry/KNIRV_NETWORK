#!/bin/bash

# KNIRV Client Database Diagnostic Script

echo "╔════════════════════════════════════════════════════════════════╗"
echo "║         KNIRV Client Database Diagnostic Tool                  ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo ""

# Get the application data directory
APP_DATA_DIR="$HOME/.local/share/knirvclient"
CHROMEM_DB_DIR="$APP_DATA_DIR/chromem-db"
CHROMEM_MANAGER_DIR="$APP_DATA_DIR/chromem-manager-db"

echo "📁 Checking database directories..."
echo ""

# Check if app data directory exists
if [ -d "$APP_DATA_DIR" ]; then
    echo "✅ Application data directory exists: $APP_DATA_DIR"
    echo "   Size: $(du -sh "$APP_DATA_DIR" | cut -f1)"
else
    echo "❌ Application data directory NOT found: $APP_DATA_DIR"
    echo "   This means the application hasn't been run yet or data was deleted."
    exit 1
fi

echo ""

# Check chromem-db directory
if [ -d "$CHROMEM_DB_DIR" ]; then
    echo "✅ Chromem-DB directory exists: $CHROMEM_DB_DIR"
    echo "   Size: $(du -sh "$CHROMEM_DB_DIR" | cut -f1)"
    echo "   Files:"
    ls -lh "$CHROMEM_DB_DIR" | tail -n +2 | awk '{print "     " $9 " (" $5 ")"}'
else
    echo "❌ Chromem-DB directory NOT found: $CHROMEM_DB_DIR"
fi

echo ""

# Check chromem-manager-db directory
if [ -d "$CHROMEM_MANAGER_DIR" ]; then
    echo "✅ Chromem-Manager-DB directory exists: $CHROMEM_MANAGER_DIR"
    echo "   Size: $(du -sh "$CHROMEM_MANAGER_DIR" | cut -f1)"
    echo "   Files:"
    ls -lh "$CHROMEM_MANAGER_DIR" | tail -n +2 | awk '{print "     " $9 " (" $5 ")"}'
else
    echo "❌ Chromem-Manager-DB directory NOT found: $CHROMEM_MANAGER_DIR"
fi

echo ""
echo "════════════════════════════════════════════════════════════════"
echo ""

# Check for project collection files
echo "🔍 Searching for project-related files..."
echo ""

if [ -d "$CHROMEM_DB_DIR" ]; then
    PROJECT_FILES=$(find "$CHROMEM_DB_DIR" -type f -name "*project*" 2>/dev/null)
    if [ -n "$PROJECT_FILES" ]; then
        echo "✅ Found project-related files:"
        echo "$PROJECT_FILES" | while read -r file; do
            echo "   - $file ($(du -h "$file" | cut -f1))"
        done
    else
        echo "⚠️  No project-related files found in chromem-db"
        echo "   This could mean:"
        echo "   - No projects have been created yet"
        echo "   - Projects are stored with different naming"
        echo "   - Database is using a different structure"
    fi
else
    echo "⚠️  Cannot search - chromem-db directory doesn't exist"
fi

echo ""
echo "════════════════════════════════════════════════════════════════"
echo ""

# Check recent modifications
echo "📅 Recent database activity..."
echo ""

if [ -d "$APP_DATA_DIR" ]; then
    echo "Most recently modified files (last 10):"
    find "$APP_DATA_DIR" -type f -printf '%T@ %p\n' 2>/dev/null | sort -rn | head -10 | while read -r timestamp file; do
        date_str=$(date -d "@${timestamp%.*}" "+%Y-%m-%d %H:%M:%S" 2>/dev/null || echo "Unknown date")
        echo "   $date_str - $(basename "$file")"
    done
else
    echo "⚠️  Cannot check - application data directory doesn't exist"
fi

echo ""
echo "════════════════════════════════════════════════════════════════"
echo ""

# Summary
echo "📊 Summary:"
echo ""

if [ -d "$APP_DATA_DIR" ]; then
    FILE_COUNT=$(find "$APP_DATA_DIR" -type f 2>/dev/null | wc -l)
    TOTAL_SIZE=$(du -sh "$APP_DATA_DIR" 2>/dev/null | cut -f1)
    echo "   Total files: $FILE_COUNT"
    echo "   Total size: $TOTAL_SIZE"
    echo ""
    
    if [ "$FILE_COUNT" -eq 0 ]; then
        echo "⚠️  WARNING: No files found in database directory!"
        echo "   The application may not have successfully saved any data yet."
    elif [ "$FILE_COUNT" -lt 5 ]; then
        echo "⚠️  WARNING: Very few files found ($FILE_COUNT)"
        echo "   This might indicate the database is not fully initialized."
    else
        echo "✅ Database appears to be initialized with $FILE_COUNT files"
    fi
else
    echo "❌ Application data directory doesn't exist"
    echo "   Run the application at least once to initialize the database."
fi

echo ""
echo "════════════════════════════════════════════════════════════════"
echo ""
echo "💡 Tips:"
echo "   - If no database exists, run: ./knirvclient"
echo "   - To reset database, delete: $APP_DATA_DIR"
echo "   - Check application logs for database errors"
echo "   - Look for 'Project persisted to database' messages in logs"
echo ""