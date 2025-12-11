#!/usr/bin/env python3
"""
Convert HRM checkpoint to safetensors format for WASM integration
"""
import os
import argparse
from pathlib import Path
from huggingface_hub import snapshot_download
from transformers import AutoModel

def convert_hf_model_to_safetensors(repo_id, local_dir, output_path):
    """
    Download a model from Hugging Face Hub and ensure it's in safetensors format.
    If the model is in PyTorch format (.bin), it will be converted.
    """
    print(f"Downloading model '{repo_id}' to '{local_dir}'...")
    
    try:
        # Download the model files. This will handle caching.
        # We prefer safetensors if available.
        snapshot_download(
            repo_id=repo_id,
            local_dir=local_dir,
            local_dir_use_symlinks=False,
            allow_patterns=["*.safetensors", "*.json", "*.md"],
        )
        
        # Check if safetensors files were downloaded
        safetensor_files = list(Path(local_dir).glob('*.safetensors'))
        if safetensor_files:
            print("Model already in safetensors format. Copying to output path.")
            # For simplicity, we'll just point to the main model file.
            # In a real scenario, you might need to handle multiple shards.
            main_model_file = safetensor_files[0]
            os.link(main_model_file, output_path) # Create a hard link to save space
            print(f"Model weights available at: {output_path}")
            return True
        
        print("No safetensors file found. Loading PyTorch model to convert.")
        model = AutoModel.from_pretrained(repo_id, trust_remote_code=True)
        print(f"Saving model to safetensors format at: {output_path}")
        model.save_pretrained(output_path.parent, safe_serialization=True)
        print("Conversion successful.")
        return True
            
    except Exception as e:
        print(f"An error occurred: {e}")
        return False

def main():
    parser = argparse.ArgumentParser(description="Convert HRM checkpoint to safetensors")
    parser.add_argument("--repo-id", 
                       default="microsoft/phi-3-mini-4k-instruct",
                       help="Hugging Face repository ID of the model to download.")
    parser.add_argument("--cache-dir",
                       default="~/.cache/knirv-models",
                       help="Directory to cache the downloaded model.")
    parser.add_argument("--output",
                       default="weights.safetensors", 
                       help="Output safetensors file path")
    
    args = parser.parse_args()
    
    # Expand user path
    cache_dir = Path(args.cache_dir).expanduser()
    output_path = Path(args.output)
    
    # Ensure output directory exists
    output_path.parent.mkdir(parents=True, exist_ok=True)
    
    print(f"Downloading and converting model to safetensors...")
    print(f"Model: {args.repo_id}")
    print(f"Output: {output_path}")
    
    success = convert_hf_model_to_safetensors(args.repo_id, cache_dir, output_path)
    
    if success:
        print("Conversion completed successfully!")
        return 0
    else:
        print("Conversion failed!")
        return 1

if __name__ == "__main__":
    exit(main())
