#!/bin/bash
# Script to update and test troubleshooting embeddings
# This script:
# 1. Sets up a Python virtual environment if needed
# 2. Installs required dependencies
# 3. Generates embeddings from the troubleshooting guide
# 4. Tests the generated embeddings
# 5. Provides clear output and error handling

set -e  # Exit immediately if a command exits with a non-zero status

# Configuration
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
VENV_DIR="$PROJECT_ROOT/.venv"
REQUIREMENTS_FILE="$SCRIPT_DIR/embedding_requirements.txt"
INPUT_FILE="$PROJECT_ROOT/docs/known_issues.md"
OUTPUT_DIR="$PROJECT_ROOT/api/data"
OUTPUT_FILE="$OUTPUT_DIR/troubleshooting_embeddings.json"

# ANSI color codes for output formatting
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print header
echo -e "${BLUE}======================================================${NC}"
echo -e "${BLUE}   KNIRVENGINE Troubleshooting Embeddings Update   ${NC}"
echo -e "${BLUE}======================================================${NC}"
echo

# Create requirements file if it doesn't exist
if [ ! -f "$REQUIREMENTS_FILE" ]; then
    echo -e "${YELLOW}Creating requirements file...${NC}"
    cat > "$REQUIREMENTS_FILE" << EOF
numpy>=1.20.0
markdown>=3.3.0
beautifulsoup4>=4.9.0
sentence-transformers>=2.2.0
EOF
    echo -e "${GREEN}Requirements file created at:${NC} $REQUIREMENTS_FILE"
fi

# Create output directory if it doesn't exist
if [ ! -d "$OUTPUT_DIR" ]; then
    echo -e "${YELLOW}Creating output directory...${NC}"
    mkdir -p "$OUTPUT_DIR"
    echo -e "${GREEN}Output directory created at:${NC} $OUTPUT_DIR"
fi

# Check if input file exists
if [ ! -f "$INPUT_FILE" ]; then
    echo -e "${RED}Error: Input file not found:${NC} $INPUT_FILE"
    echo "Please make sure the troubleshooting guide exists at this location."
    exit 1
fi

# Setup virtual environment
echo -e "${YELLOW}Setting up Python virtual environment...${NC}"

# Check if virtual environment exists
if [ ! -d "$VENV_DIR" ]; then
    echo "Creating new virtual environment..."
    python3 -m venv "$VENV_DIR"
    echo -e "${GREEN}Virtual environment created at:${NC} $VENV_DIR"
else
    echo -e "${GREEN}Using existing virtual environment at:${NC} $VENV_DIR"
fi

# Activate virtual environment
if [[ "$OSTYPE" == "msys" || "$OSTYPE" == "win32" ]]; then
    # Windows
    source "$VENV_DIR/Scripts/activate"
else
    # Linux/Mac
    source "$VENV_DIR/bin/activate"
fi

# Install dependencies
echo -e "${YELLOW}Installing required dependencies...${NC}"
pip install --upgrade pip
pip install -r "$REQUIREMENTS_FILE"
echo -e "${GREEN}Dependencies installed successfully.${NC}"

# Generate embeddings
echo
echo -e "${YELLOW}Generating embeddings from troubleshooting guide...${NC}"
echo "Input file: $INPUT_FILE"
echo "Output file: $OUTPUT_FILE"
echo

# Run the embedding creation script
python "$SCRIPT_DIR/create_troubleshooting_embeddings.py" --input "$INPUT_FILE" --output "$OUTPUT_FILE"

# Check if embedding creation was successful
if [ $? -eq 0 ] && [ -f "$OUTPUT_FILE" ]; then
    echo
    echo -e "${GREEN}Embeddings generated successfully!${NC}"
    echo -e "File size: $(du -h "$OUTPUT_FILE" | cut -f1)"
    echo
else
    echo
    echo -e "${RED}Error: Failed to generate embeddings.${NC}"
    echo "Please check the error messages above."
    exit 1
fi

# Test embeddings
echo -e "${YELLOW}Testing generated embeddings...${NC}"
echo

# Run the embedding test script
python "$SCRIPT_DIR/test_embeddings.py" --embeddings "$OUTPUT_FILE"

# Check if embedding test was successful
if [ $? -eq 0 ]; then
    echo
    echo -e "${GREEN}Embedding tests completed successfully!${NC}"
    echo
else
    echo
    echo -e "${RED}Warning: Embedding tests encountered issues.${NC}"
    echo "Please review the test results above."
fi

# Deactivate virtual environment
deactivate

echo -e "${BLUE}======================================================${NC}"
echo -e "${GREEN}Troubleshooting embeddings update complete!${NC}"
echo -e "${BLUE}======================================================${NC}"
echo
echo "The embeddings have been generated and tested successfully."
echo "They are now ready to be used by the AI Error Inference Engine."
echo
echo "File location: $OUTPUT_FILE"
echo
echo "If you made changes to the troubleshooting guide, remember to commit"
echo "both the guide and the generated embeddings file to the repository."
echo

exit 0