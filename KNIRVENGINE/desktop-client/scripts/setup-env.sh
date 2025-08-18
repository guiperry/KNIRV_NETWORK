#!/bin/bash

# Script to set up the environment configuration for KNIRVENGINE

# Get the directory where the script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
# Navigate to the project root directory (parent of scripts)
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

# Change to the project root directory
cd "$PROJECT_ROOT"

# Define file paths relative to the project root
ENV_EXAMPLE=".env.example"
ENV_FILE=".env"
DEFAULT_ENV="default.env"

echo "Working in directory: $(pwd)"

# Function to create default.env from a source file
create_default_env() {
    local source_file=$1
    echo "Creating $DEFAULT_ENV from $source_file..."
    
    # Copy the source file to default.env
    cp "$source_file" "$DEFAULT_ENV"
    
    # Set a default JWT secret if not already set
    if ! grep -q "^JWT_SECRET=" "$DEFAULT_ENV" || grep -q "^JWT_SECRET=$" "$DEFAULT_ENV"; then
        sed -i 's/^# JWT_SECRET=.*$/JWT_SECRET=default_jwt_secret_please_change_in_production/' "$DEFAULT_ENV"
        # If the line doesn't exist or is commented, add it
        if ! grep -q "^JWT_SECRET=" "$DEFAULT_ENV"; then
            echo "JWT_SECRET=default_jwt_secret_please_change_in_production" >> "$DEFAULT_ENV"
        fi
    fi
    
    # Add a header comment
    sed -i '1i # KNIRVENGINE Default Environment Configuration\n# These are the default values used when no .env file is found\n' "$DEFAULT_ENV"
    
    echo "✅ Created $DEFAULT_ENV file with embedded defaults"
}

# Check if .env already exists
if [ -f "$ENV_FILE" ]; then
    echo "⚠️  $ENV_FILE file already exists."
    echo "Options:"
    echo "  1. Keep existing .env and use it as the basis for default.env (including any API keys)"
    echo "  2. Overwrite .env with .env.example and create a new default.env"
    echo "  3. Cancel operation"
    
    read -p "Enter your choice (1-3): " choice
    
    case $choice in
        1)
            echo "Using existing $ENV_FILE as the basis for $DEFAULT_ENV"
            echo "⚠️  Note: Any API keys in your .env will be embedded in the application binary."
            echo "    This is useful for distribution but may pose security concerns."
            read -p "Continue? (y/n): " confirm
            if [ "$confirm" != "y" ]; then
                echo "Operation cancelled."
                exit 0
            fi
            create_default_env "$ENV_FILE"
            ;;
        2)
            if [ -f "$ENV_EXAMPLE" ]; then
                cp "$ENV_EXAMPLE" "$ENV_FILE"
                echo "✅ Overwrote $ENV_FILE with $ENV_EXAMPLE"
                echo "📝 Please edit the $ENV_FILE to add your API keys and customize settings."
                create_default_env "$ENV_EXAMPLE"
            else
                echo "❌ Error: $ENV_EXAMPLE file not found."
                exit 1
            fi
            ;;
        3)
            echo "Operation cancelled. No files were modified."
            exit 0
            ;;
        *)
            echo "Invalid choice. Operation cancelled."
            exit 1
            ;;
    esac
else
    # No .env exists, create both files from .env.example
    if [ -f "$ENV_EXAMPLE" ]; then
        cp "$ENV_EXAMPLE" "$ENV_FILE"
        echo "✅ Created $ENV_FILE from $ENV_EXAMPLE"
        echo "📝 Please edit the $ENV_FILE to add your API keys and customize settings."
        create_default_env "$ENV_EXAMPLE"
    else
        echo "❌ Error: $ENV_EXAMPLE file not found."
        exit 1
    fi
fi

# Make the script executable (in case it was copied)
chmod +x "$SCRIPT_DIR/setup-env.sh"

echo "🚀 Environment setup complete!"
echo "You can now start the application with your configured environment."