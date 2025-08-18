#!/usr/bin/env python3
"""
Script to convert the troubleshooting guide into a vector store for the AI Error Inference Engine.

This script:
1. Parses the Markdown troubleshooting guide
2. Splits it into logical chunks by issue
3. Generates embeddings for each chunk
4. Stores these in a simple JSON vector store

Requirements:
- Python 3.8+
- numpy
- markdown
- bs4 (BeautifulSoup)
- sentence_transformers (for embeddings)

Usage:
python create_troubleshooting_embeddings.py [--input path/to/known_issues.md] [--output path/to/output.json]
"""

import argparse
import json
import os
import re
from typing import Dict, List, Tuple, Any
import numpy as np
from bs4 import BeautifulSoup
import markdown
from sentence_transformers import SentenceTransformer

# Default paths
DEFAULT_INPUT_PATH = "../known_issues.md"
DEFAULT_OUTPUT_PATH = "../api/data/troubleshooting_embeddings.json"

class TroubleshootingVectorStore:
    """Class to handle the creation and storage of troubleshooting embeddings."""
    
    def __init__(self, model_name: str = "all-MiniLM-L6-v2"):
        """Initialize the vector store with the specified embedding model."""
        self.model = SentenceTransformer(model_name)
        self.chunks = []
        self.embeddings = []
        self.metadata = []
        
    def parse_markdown(self, markdown_path: str) -> List[Dict[str, Any]]:
        """Parse the markdown file and extract chunks with metadata."""
        with open(markdown_path, 'r', encoding='utf-8') as f:
            md_content = f.read()
        
        # Convert markdown to HTML for easier parsing
        html_content = markdown.markdown(md_content)
        soup = BeautifulSoup(html_content, 'html.parser')
        
        # Extract all issue sections
        chunks = []
        
        # Get all h2 sections (main categories)
        h2_tags = soup.find_all('h2')
        
        for h2 in h2_tags:
            category = h2.text.strip()
            
            # Skip the "Using the AI Error Inference Engine" section
            if "Using the AI Error Inference Engine" in category:
                continue
                
            # Find all h3 tags (specific issues) until the next h2
            current_element = h2.next_sibling
            h3_sections = []
            current_h3 = None
            current_content = []
            
            while current_element and (not hasattr(current_element, 'name') or current_element.name != 'h2'):
                if hasattr(current_element, 'name'):
                    if current_element.name == 'h3':
                        # Save previous h3 section if it exists
                        if current_h3 and current_content:
                            h3_text = ''.join(current_content)
                            h3_sections.append({
                                'title': current_h3,
                                'content': h3_text
                            })
                        
                        # Start new h3 section
                        current_h3 = current_element.text.strip()
                        current_content = []
                    elif current_h3:  # Only collect content if we're inside an h3 section
                        current_content.append(str(current_element))
                elif current_h3:  # Handle text nodes
                    current_content.append(str(current_element))
                
                current_element = current_element.next_sibling
            
            # Add the last h3 section
            if current_h3 and current_content:
                h3_text = ''.join(current_content)
                h3_sections.append({
                    'title': current_h3,
                    'content': h3_text
                })
            
            # Create a chunk for each issue (h3) with its category (h2)
            for section in h3_sections:
                # Extract symptoms and troubleshooting steps
                symptoms_match = re.search(r'<strong>Symptoms:</strong>(.*?)<strong>Troubleshooting Steps:</strong>', 
                                          section['content'], re.DOTALL)
                
                symptoms = []
                if symptoms_match:
                    symptoms_html = symptoms_match.group(1)
                    symptoms_soup = BeautifulSoup(symptoms_html, 'html.parser')
                    li_tags = symptoms_soup.find_all('li')
                    symptoms = [li.text.strip() for li in li_tags]
                
                # Clean the content by removing HTML tags
                clean_content = BeautifulSoup(section['content'], 'html.parser').get_text()
                
                chunk = {
                    'category': category,
                    'issue': section['title'],
                    'symptoms': symptoms,
                    'content': clean_content,
                    'raw_html': section['content']
                }
                chunks.append(chunk)
        
        return chunks
    
    def create_embeddings(self, chunks: List[Dict[str, Any]]) -> None:
        """Create embeddings for each chunk."""
        self.chunks = chunks
        
        # Prepare texts for embedding
        texts = []
        for chunk in chunks:
            # Create a rich text representation for embedding
            text = f"Category: {chunk['category']}\nIssue: {chunk['issue']}\n"
            text += f"Symptoms: {', '.join(chunk['symptoms'])}\n"
            text += f"Content: {chunk['content']}"
            texts.append(text)
        
        # Generate embeddings
        self.embeddings = self.model.encode(texts)
        
        # Create metadata
        self.metadata = [{
            'category': chunk['category'],
            'issue': chunk['issue'],
            'symptoms': chunk['symptoms']
        } for chunk in chunks]
    
    def save_to_json(self, output_path: str) -> None:
        """Save the vector store to a JSON file."""
        # Create directory if it doesn't exist
        os.makedirs(os.path.dirname(output_path), exist_ok=True)
        
        # Convert numpy arrays to lists for JSON serialization
        embeddings_list = [embedding.tolist() for embedding in self.embeddings]
        
        # Create the vector store data structure
        vector_store = {
            'chunks': self.chunks,
            'embeddings': embeddings_list,
            'metadata': self.metadata
        }
        
        # Save to JSON
        with open(output_path, 'w', encoding='utf-8') as f:
            json.dump(vector_store, f, ensure_ascii=False, indent=2)
        
        print(f"Vector store saved to {output_path}")
        print(f"Created {len(self.chunks)} chunks with embeddings")

def main():
    """Main function to parse arguments and create the vector store."""
    parser = argparse.ArgumentParser(description='Convert troubleshooting guide to vector store')
    parser.add_argument('--input', type=str, default=DEFAULT_INPUT_PATH,
                        help=f'Path to the input markdown file (default: {DEFAULT_INPUT_PATH})')
    parser.add_argument('--output', type=str, default=DEFAULT_OUTPUT_PATH,
                        help=f'Path to the output JSON file (default: {DEFAULT_OUTPUT_PATH})')
    parser.add_argument('--model', type=str, default="all-MiniLM-L6-v2",
                        help='Name of the sentence-transformers model to use (default: all-MiniLM-L6-v2)')
    
    args = parser.parse_args()
    
    # Create the vector store
    vector_store = TroubleshootingVectorStore(model_name=args.model)
    
    # Parse the markdown file
    print(f"Parsing markdown file: {args.input}")
    chunks = vector_store.parse_markdown(args.input)
    
    # Create embeddings
    print("Creating embeddings...")
    vector_store.create_embeddings(chunks)
    
    # Save to JSON
    vector_store.save_to_json(args.output)

if __name__ == "__main__":
    main()