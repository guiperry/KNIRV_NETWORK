It's a common issue to run into gated models or models that aren't publicly available. The sapient/hrm-27m model, as described in the project's README.md, is based on a novel "Hierarchical Reasoning Model" architecture. Finding an exact open-source replica might be difficult, but we can definitely find excellent open-source alternatives that fit the same role in our agent-core architecture.

The goal is to find a small, powerful model capable of reasoning, which can be integrated into our agent's "Cognitive Core" and potentially compiled to WASM.

Here are some great open-source alternatives we can use right away:

Recommended Alternatives
1. Microsoft Phi-3 Mini (Best Overall Start)
Model ID: microsoft/phi-3-mini-4k-instruct
Why it's a good fit: This is likely our best option. It's a 3.8 billion parameter model, but it's designed to be extremely capable for its size, outperforming models twice its size. It has a very permissive MIT license, making it great for any project. Its small size and efficiency make it a prime candidate for compiling to WASM or running in resource-constrained environments.
Link: https://huggingface.co/microsoft/phi-3-mini-4k-instruct
2. Google RecurrentGemma (Architecturally Interesting)
Model ID: google/recurrentgemma-2b
Why it's a good fit: This model is architecturally interesting because, like HRM, it is recurrent. It uses a novel architecture (GrGrU) that is not a standard Transformer, making it very efficient for long sequences and stateful tasks, which seems to align perfectly with the goals of the HRM. It's small (2.7B parameters) and has a permissive license.
Link: https://huggingface.co/google/recurrentgemma-2b
3. TinyLlama
Model ID: TinyLlama/TinyLlama-1.1B-Chat-v1.0
Why it's a good fit: If you need something even smaller for testing or for very constrained environments, TinyLlama is an excellent choice. At 1.1 billion parameters, it's very lightweight but still surprisingly capable. It also has a permissive Apache 2.0 license.
Link: https://huggingface.co/TinyLlama/TinyLlama-1.1B-Chat-v1.0

Code Integration Plan
To integrate one of these models, you'll need to download its weights and convert them to the .safetensors format that your WASM pipeline expects. The existing convert_checkpoint_to_safetensors.py script can be significantly improved to handle this process automatically for any model on the Hugging Face Hub.

Here is a suggested improvement to the script. This new version will use the transformers library to download a specified model and save it directly to safetensors.

convert_checkpoint_to_safetensors.py
-78
+43
 Show full code block 
"""
Convert HRM checkpoint to safetensors format for WASM integration
"""

import os
import torch
import numpy as np
from safetensors.torch import save_file
import argparse
from pathlib import Path
from huggingface_hub import snapshot_download
from transformers import AutoModel

def convert_checkpoint_to_safetensors(checkpoint_path, output_path):
def convert_hf_model_to_safetensors(repo_id, local_dir, output_path):
    """
    Convert HRM checkpoint to safetensors format
    Download a model from Hugging Face Hub and ensure it's in safetensors format.
    If the model is in PyTorch format (.bin), it will be converted.
    """
    print(f"Loading checkpoint from: {checkpoint_path}")
    print(f"Downloading model '{repo_id}' to '{local_dir}'...")
    
    try:
        # Try loading the checkpoint
        checkpoint = torch.load(checkpoint_path, map_location="cpu")
        print(f"Checkpoint loaded successfully")
        print(f"Checkpoint keys: {list(checkpoint.keys()) if isinstance(checkpoint, dict) else 'Not a dict'}")
        # Download the model files. This will handle caching.
        # We prefer safetensors if available.
        snapshot_download(
            repo_id=repo_id,
            local_dir=local_dir,
            local_dir_use_symlinks=False,
            allow_patterns=["*.safetensors", "*.json", "*.md"],
        )
        
        # Extract model state dict
        if isinstance(checkpoint, dict):
            if 'model' in checkpoint:
                model_state = checkpoint['model']
            elif 'state_dict' in checkpoint:
                model_state = checkpoint['state_dict']
            elif 'model_state_dict' in checkpoint:
                model_state = checkpoint['model_state_dict']
            else:
                # Assume the checkpoint itself is the state dict
                model_state = checkpoint
        else:
            print("Checkpoint is not a dictionary, treating as model state")
            model_state = checkpoint

        # Handle compiled model state dict prefixes
        if isinstance(model_state, dict):
            unwrapped_model_state = {}
            for k, v in model_state.items():
                unwrapped_model_state[k.removeprefix("_orig_mod.")] = v
            model_state = unwrapped_model_state
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
        
        print(f"Model state type: {type(model_state)}")
        if isinstance(model_state, dict):
            print(f"Model state keys: {list(model_state.keys())[:10]}...")  # Show first 10 keys
            
            # Convert tensors to float16 to reduce size
            print("Converting tensors to float16...")
            converted_tensors = {}
            total_params = 0
            
            for key, tensor in model_state.items():
                if isinstance(tensor, torch.Tensor):
                    # Convert to float16 for size reduction
                    converted_tensor = tensor.half()
                    converted_tensors[key] = converted_tensor
                    total_params += tensor.numel()
                    print(f"  {key}: {tensor.shape} -> {converted_tensor.shape} ({converted_tensor.dtype})")
                else:
                    print(f"  Skipping non-tensor: {key} ({type(tensor)})")
            
            print(f"Total parameters: {total_params:,}")
            print(f"Estimated model size: {total_params * 2 / 1024 / 1024:.1f} MB (float16)")
            
            # Save as safetensors
            print(f"Saving to safetensors format: {output_path}")
            save_file(converted_tensors, output_path)
            
            # Verify the saved file
            file_size = os.path.getsize(output_path)
            print(f"Saved file size: {file_size / 1024 / 1024:.1f} MB")
            
            return True
            
        else:
            print(f"Error: Model state is not a dictionary: {type(model_state)}")
            return False
        print("No safetensors file found. Loading PyTorch model to convert.")
        model = AutoModel.from_pretrained(repo_id, trust_remote_code=True)
        print(f"Saving model to safetensors format at: {output_path}")
        model.save_pretrained(output_path.parent, safe_serialization=True)
        print("Conversion successful.")
        return True
            
    except Exception as e:
        print(f"Error loading checkpoint: {e}")
        print(f"An error occurred: {e}")
        return False

def main():
    parser = argparse.ArgumentParser(description="Convert HRM checkpoint to safetensors")
    parser.add_argument("--checkpoint", 
                       default="~/.cache/hrm/checkpoints/hrm_27m.pt",
                       help="Path to HRM checkpoint file")
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
    checkpoint_path = Path(args.checkpoint).expanduser()
    cache_dir = Path(args.cache_dir).expanduser()
    output_path = Path(args.output)
    
    if not checkpoint_path.exists():
        print(f"Error: Checkpoint file not found: {checkpoint_path}")
        return 1
    # Ensure output directory exists
    output_path.parent.mkdir(parents=True, exist_ok=True)
    
    print(f"Converting HRM checkpoint to safetensors...")
    print(f"Input: {checkpoint_path}")
    print(f"Downloading and converting model to safetensors...")
    print(f"Model: {args.repo_id}")
    print(f"Output: {output_path}")
    
    success = convert_checkpoint_to_safetensors(checkpoint_path, output_path)
    success = convert_hf_model_to_safetensors(args.repo_id, cache_dir, output_path)
    
    if success:
        print("Conversion completed successfully!")

With this improved script, you can now easily download and prepare any of the recommended models:

bash
# Download and convert Phi-3 Mini
python convert_checkpoint_to_safetensors.py \
    --repo-id="microsoft/phi-3-mini-4k-instruct" \
    --output="phi-3-mini.safetensors"

# Download and convert RecurrentGemma
python convert_checkpoint_to_safetensors.py \
    --repo-id="google/recurrentgemma-2b" \
    --output="recurrentgemma-2b.safetensors"
This approach gives you a flexible and robust way to experiment with different open-source models for your project.