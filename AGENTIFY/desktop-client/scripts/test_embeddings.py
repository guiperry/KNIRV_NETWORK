#!/usr/bin/env python3
"""
Simple script to test the troubleshooting embeddings.

This script:
1. Loads the troubleshooting embeddings
2. Simulates a few error scenarios
3. Retrieves relevant troubleshooting information

Usage:
python test_embeddings.py [--embeddings path/to/embeddings.json]
"""

import argparse
import json
import numpy as np
from sentence_transformers import SentenceTransformer

# Default paths
DEFAULT_EMBEDDINGS_PATH = "../api/data/troubleshooting_embeddings.json"

def load_embeddings(file_path):
    """Load embeddings from a JSON file."""
    with open(file_path, 'r', encoding='utf-8') as f:
        data = json.load(f)
    
    # Convert list embeddings back to numpy arrays
    data['embeddings'] = [np.array(embedding) for embedding in data['embeddings']]
    
    return data

def cosine_similarity(a, b):
    """Calculate cosine similarity between two vectors."""
    return np.dot(a, b) / (np.linalg.norm(a) * np.linalg.norm(b))

def search_embeddings(data, query_text, model, top_k=3):
    """Search embeddings for relevant troubleshooting information."""
    # Generate embedding for query
    query_embedding = model.encode(query_text)
    
    # Calculate similarities
    similarities = []
    for i, embedding in enumerate(data['embeddings']):
        similarity = cosine_similarity(query_embedding, embedding)
        similarities.append((i, similarity))
    
    # Sort by similarity (descending)
    similarities.sort(key=lambda x: x[1], reverse=True)
    
    # Return top k results
    results = []
    for i, similarity in similarities[:top_k]:
        results.append({
            'chunk': data['chunks'][i],
            'metadata': data['metadata'][i],
            'similarity': similarity
        })
    
    return results

def main():
    """Main function to test embeddings."""
    parser = argparse.ArgumentParser(description='Test troubleshooting embeddings')
    parser.add_argument('--embeddings', type=str, default=DEFAULT_EMBEDDINGS_PATH,
                        help=f'Path to the embeddings JSON file (default: {DEFAULT_EMBEDDINGS_PATH})')
    
    args = parser.parse_args()
    
    # Load embeddings
    print(f"Loading embeddings from {args.embeddings}")
    try:
        data = load_embeddings(args.embeddings)
        print(f"Loaded {len(data['chunks'])} chunks with embeddings")
    except Exception as e:
        print(f"Error loading embeddings: {e}")
        return
    
    # Initialize model
    model = SentenceTransformer('all-MiniLM-L6-v2')
    
    # Test scenarios
    test_scenarios = [
        {
            'name': 'Network Error',
            'query': 'I keep getting network connection failed errors and timeouts when making API requests',
        },
        {
            'name': 'Authentication Issue',
            'query': 'I cannot log in, it says authentication required and my JWT token is invalid',
        },
        {
            'name': 'Database Problem',
            'query': 'The application crashes with database errors and data is not being saved',
        },
        {
            'name': 'High CPU Usage',
            'query': 'My system is very slow and the CPU usage is at 100% when running the application',
        },
        {
            'name': 'WebSocket Disconnection',
            'query': 'Terminal sessions keep disconnecting and I see WebSocket connection failed errors',
        }
    ]
    
    # Run tests
    for scenario in test_scenarios:
        print(f"\n\n=== Testing Scenario: {scenario['name']} ===")
        print(f"Query: {scenario['query']}")
        
        results = search_embeddings(data, scenario['query'], model)
        
        print(f"\nTop {len(results)} Results:")
        for i, result in enumerate(results):
            print(f"\n{i+1}. {result['chunk']['issue']} (Similarity: {result['similarity']:.4f})")
            print(f"   Category: {result['chunk']['category']}")
            print(f"   Symptoms: {', '.join(result['chunk']['symptoms'])}")
            
            # Print first 200 characters of content
            content = result['chunk']['content']
            print(f"   Content Preview: {content[:200]}...")

if __name__ == "__main__":
    main()