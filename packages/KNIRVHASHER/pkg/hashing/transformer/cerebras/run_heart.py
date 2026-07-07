#!/usr/bin/env cs_python
"""
HEART Transformer Runner for Cerebras WSE
Handles compilation, weight loading, and execution of the HEART model
"""

import argparse
import json
import numpy as np
import sys
import os
from pathlib import Path

from cerebras.sdk.runtime.sdkruntimepybind import SdkRuntime
from cerebras.sdk.runtime.sdkruntimepybind import MemcpyDataType
from cerebras.sdk.runtime.sdkruntimepybind import MemcpyOrder

class HEARTRunner:
    """
    Manages the HEART transformer execution on Cerebras WSE
    """

    def __init__(self, program_dir, cmaddr=None):
        """
        Initialize the HEART runner

        Args:
            program_dir: Directory containing compiled CSL program
            cmaddr: IP:port for Cerebras system (None for simulator)
        """
        self.program_dir = program_dir
        self.cmaddr = cmaddr
        self.runner = None
        self.params = None

        # Load compile metadata
        self._load_metadata()

        # Initialize runtime
        self.runner = SdkRuntime(program_dir, cmaddr=cmaddr)

    def _load_metadata(self):
        """Load parameters from compilation metadata"""
        metadata_path = Path(self.program_dir) / "out.json"
        with open(metadata_path, 'r', encoding='utf-8') as f:
            compile_data = json.load(f)

        self.params = {
            'GRID_WIDTH': int(compile_data['params']['GRID_WIDTH']),
            'GRID_HEIGHT': int(compile_data['params']['GRID_HEIGHT']),
            'SEQUENCE_LENGTH': int(compile_data['params']['SEQUENCE_LENGTH']),
            'NUM_NODES': int(compile_data['params']['NUM_NODES']),
            'EMBEDDING_DIM': int(compile_data['params']['EMBEDDING_DIM']),
            'NUM_HEADS': int(compile_data['params']['NUM_HEADS']),
            'NUM_LAYERS': int(compile_data['params']['NUM_LAYERS']),
            'MEASURE_VEC_SIZE': int(compile_data['params']['MEASURE_VEC_SIZE']),
            'CONTROL_VEC_SIZE': int(compile_data['params']['CONTROL_VEC_SIZE']),
            'SEQ_PER_PE': int(compile_data['params']['SEQ_PER_PE']),
            'EMBED_PER_PE': int(compile_data['params']['EMBED_PER_PE']),
        }

        print(f"Loaded HEART configuration:")
        for key, value in self.params.items():
            print(f"  {key}: {value}")

    def load_weights(self, weights_path):
        """
        Load model weights from file and transfer to WSE

        Args:
            weights_path: Path to weights file (NPZ format)
        """
        print(f"Loading weights from {weights_path}...")

        # Load weights from file
        weights_data = np.load(weights_path)

        # Get symbol IDs
        sym_embedding = self.runner.get_id("embedding_weights")
        sym_positional = self.runner.get_id("positional_encoding")

        # Load weights to device
        self.runner.load()

        # Transfer embedding weights
        embedding_weights = weights_data['embedding_weights'].astype(np.float32)
        print(f"  Embedding weights shape: {embedding_weights.shape}")
        self._transfer_weights_h2d(sym_embedding, embedding_weights)

        # Transfer positional encoding
        positional_encoding = weights_data['positional_encoding'].astype(np.float32)
        print(f"  Positional encoding shape: {positional_encoding.shape}")
        self._transfer_weights_h2d(sym_positional, positional_encoding)

        # Load layer weights (attention and FFN)
        # These are loaded per PE based on the decomposition
        # For now, we'll load pre-distributed weights
        if 'layer_weights' in weights_data:
            sym_layer_weights = self.runner.get_id("layer_weights")
            layer_weights = weights_data['layer_weights'].astype(np.float32)
            print(f"  Layer weights shape: {layer_weights.shape}")
            self._transfer_weights_h2d(sym_layer_weights, layer_weights)

        # Load output projection weights
        if 'output_weights' in weights_data:
            sym_output = self.runner.get_id("output_weights")
            output_weights = weights_data['output_weights'].astype(np.float32)
            print(f"  Output weights shape: {output_weights.shape}")
            self._transfer_weights_h2d(sym_output, output_weights)

        print("Weights loaded successfully!")

    def _transfer_weights_h2d(self, symbol, weights):
        """Transfer weights from host to device"""
        w = self.params['GRID_WIDTH']
        h = self.params['GRID_HEIGHT']

        # Flatten if needed
        weights_flat = weights.ravel()

        # Transfer to all PEs (broadcast)
        self.runner.memcpy_h2d(
            symbol,
            weights_flat,
            0, 0, w, h, len(weights_flat),
            streaming=False,
            data_type=MemcpyDataType.MEMCPY_32BIT,
            order=MemcpyOrder.ROW_MAJOR,
            nonblock=False
        )

    def run_inference(self, measurement_data, node_ids, time_slices):
        """
        Run transformer inference on input data

        Args:
            measurement_data: Network measurements (N x MEASURE_VEC_SIZE)
            node_ids: Node identifiers (N,)
            time_slices: Time slice indices (N,)

        Returns:
            heuristic_commands: Output control vectors (N x CONTROL_VEC_SIZE)
            output_node_ids: Node IDs from output
            output_time_slices: Time slices from output
        """
        print(f"Running inference on {len(measurement_data)} samples...")

        # Get symbol IDs for I/O
        sym_measurement_in = self.runner.get_id("measurement_vec_in")
        sym_node_id_in = self.runner.get_id("node_id_in")
        sym_t_slice_in = self.runner.get_id("t_slice_in")
        sym_heuristic_out = self.runner.get_id("heuristic_commands_out")
        sym_node_id_out = self.runner.get_id("node_id_out")
        sym_t_slice_out = self.runner.get_id("t_slice_out")

        w = self.params['GRID_WIDTH']
        h = self.params['GRID_HEIGHT']
        seq_per_pe = self.params['SEQ_PER_PE']
        measure_vec_size = self.params['MEASURE_VEC_SIZE']
        control_vec_size = self.params['CONTROL_VEC_SIZE']

        # Ensure data is correct shape
        num_samples = len(measurement_data)
        assert num_samples == self.params['SEQUENCE_LENGTH'] * self.params['NUM_NODES'], \
            f"Expected {self.params['SEQUENCE_LENGTH'] * self.params['NUM_NODES']} samples"

        # Prepare data distribution across PEs
        # Each PE handles seq_per_pe sequence positions
        measurement_data = measurement_data.astype(np.float32)
        node_ids = node_ids.astype(np.uint32)
        time_slices = time_slices.astype(np.uint32)

        # Distribute data across PEs in row-major order
        # Shape: (h, w, seq_per_pe, measure_vec_size)
        measurement_distributed = self._distribute_data(
            measurement_data.reshape(self.params['SEQUENCE_LENGTH'], -1),
            w, h, seq_per_pe, measure_vec_size
        )
        node_id_distributed = self._distribute_data(
            node_ids.reshape(self.params['SEQUENCE_LENGTH'], 1),
            w, h, seq_per_pe, 1
        )
        time_slice_distributed = self._distribute_data(
            time_slices.reshape(self.params['SEQUENCE_LENGTH'], 1),
            w, h, seq_per_pe, 1
        )

        # Transfer input data to device
        self.runner.memcpy_h2d(
            sym_measurement_in,
            measurement_distributed.ravel(),
            0, 0, w, h, seq_per_pe * measure_vec_size,
            streaming=False,
            data_type=MemcpyDataType.MEMCPY_32BIT,
            order=MemcpyOrder.ROW_MAJOR,
            nonblock=True
        )

        self.runner.memcpy_h2d(
            sym_node_id_in,
            node_id_distributed.ravel(),
            0, 0, w, h, seq_per_pe,
            streaming=False,
            data_type=MemcpyDataType.MEMCPY_32BIT,
            order=MemcpyOrder.ROW_MAJOR,
            nonblock=True
        )

        self.runner.memcpy_h2d(
            sym_t_slice_in,
            time_slice_distributed.ravel(),
            0, 0, w, h, seq_per_pe,
            streaming=False,
            data_type=MemcpyDataType.MEMCPY_32BIT,
            order=MemcpyOrder.ROW_MAJOR,
            nonblock=True
        )

        # Launch computation
        print("Launching transformer computation...")
        self.runner.launch("main", nonblock=False)

        # Retrieve output
        output_data = np.zeros(h * w * seq_per_pe * control_vec_size, dtype=np.uint32)
        output_node_ids = np.zeros(h * w * seq_per_pe, dtype=np.uint32)
        output_time_slices = np.zeros(h * w * seq_per_pe, dtype=np.uint32)

        self.runner.memcpy_d2h(
            output_data,
            sym_heuristic_out,
            0, 0, w, h, seq_per_pe * control_vec_size,
            streaming=False,
            data_type=MemcpyDataType.MEMCPY_32BIT,
            order=MemcpyOrder.ROW_MAJOR,
            nonblock=False
        )

        self.runner.memcpy_d2h(
            output_node_ids,
            sym_node_id_out,
            0, 0, w, h, seq_per_pe,
            streaming=False,
            data_type=MemcpyDataType.MEMCPY_32BIT,
            order=MemcpyOrder.ROW_MAJOR,
            nonblock=False
        )

        self.runner.memcpy_d2h(
            output_time_slices,
            sym_t_slice_out,
            0, 0, w, h, seq_per_pe,
            streaming=False,
            data_type=MemcpyDataType.MEMCPY_32BIT,
            order=MemcpyOrder.ROW_MAJOR,
            nonblock=False
        )

        # Reconstruct output from distributed format
        heuristic_commands = self._reconstruct_data(
            output_data.view(np.float32).reshape(h, w, seq_per_pe, control_vec_size),
            w, h, seq_per_pe, control_vec_size
        )
        output_node_ids = self._reconstruct_data(
            output_node_ids.reshape(h, w, seq_per_pe, 1),
            w, h, seq_per_pe, 1
        ).squeeze()
        output_time_slices = self._reconstruct_data(
            output_time_slices.reshape(h, w, seq_per_pe, 1),
            w, h, seq_per_pe, 1
        ).squeeze()

        print("Inference complete!")
        return heuristic_commands, output_node_ids, output_time_slices

    def _distribute_data(self, data, w, h, seq_per_pe, vec_size):
        """Distribute data across PE grid"""
        # Data shape: (sequence_length, vec_size)
        # Output shape: (h, w, seq_per_pe, vec_size)
        sequence_length = data.shape[0]
        distributed = np.zeros((h, w, seq_per_pe, vec_size), dtype=data.dtype)

        for px in range(w):
            for py in range(h):
                start_idx = px * seq_per_pe
                end_idx = min(start_idx + seq_per_pe, sequence_length)
                actual_len = end_idx - start_idx

                if actual_len > 0:
                    distributed[py, px, :actual_len, :] = data[start_idx:end_idx, :]

        return distributed

    def _reconstruct_data(self, distributed_data, w, h, seq_per_pe, vec_size):
        """Reconstruct data from PE grid distribution"""
        # Input shape: (h, w, seq_per_pe, vec_size)
        # Output shape: (sequence_length, vec_size)
        sequence_length = w * seq_per_pe
        reconstructed = np.zeros((sequence_length, vec_size), dtype=distributed_data.dtype)

        for px in range(w):
            start_idx = px * seq_per_pe
            end_idx = start_idx + seq_per_pe
            reconstructed[start_idx:end_idx, :] = distributed_data[0, px, :, :]

        return reconstructed

    def stop(self):
        """Stop the runner and cleanup"""
        if self.runner:
            self.runner.stop()
            print("Runner stopped")


def main():
    """Main entry point for HEART runner"""
    parser = argparse.ArgumentParser(description="Run HEART transformer on Cerebras WSE")
    parser.add_argument("--program", required=True, help="Compiled program directory")
    parser.add_argument("--cmaddr", help="IP:port for Cerebras system (omit for simulator)")
    parser.add_argument("--weights", required=True, help="Path to weights file (NPZ)")
    parser.add_argument("--input", help="Path to input data (NPZ)")
    parser.add_argument("--output", help="Path to save output (NPZ)")
    parser.add_argument("--test", action="store_true", help="Run with synthetic test data")

    args = parser.parse_args()

    # Initialize runner
    runner = HEARTRunner(args.program, cmaddr=args.cmaddr)

    # Load weights
    runner.load_weights(args.weights)

    # Prepare input data
    if args.test:
        print("Generating synthetic test data...")
        seq_len = runner.params['SEQUENCE_LENGTH']
        num_nodes = runner.params['NUM_NODES']
        measure_size = runner.params['MEASURE_VEC_SIZE']

        # Create dummy input
        measurement_data = np.random.randn(seq_len * num_nodes, measure_size).astype(np.float32)
        node_ids = np.tile(np.arange(num_nodes, dtype=np.uint32), seq_len)
        time_slices = np.repeat(np.arange(seq_len, dtype=np.uint32), num_nodes)
    elif args.input:
        print(f"Loading input data from {args.input}...")
        input_data = np.load(args.input)
        measurement_data = input_data['measurements']
        node_ids = input_data['node_ids']
        time_slices = input_data['time_slices']
    else:
        print("Error: Must specify either --input or --test")
        sys.exit(1)

    # Run inference
    heuristic_commands, output_node_ids, output_time_slices = runner.run_inference(
        measurement_data, node_ids, time_slices
    )

    # Save or display output
    if args.output:
        print(f"Saving output to {args.output}...")
        np.savez(
            args.output,
            heuristic_commands=heuristic_commands,
            node_ids=output_node_ids,
            time_slices=output_time_slices
        )
    else:
        print("\nOutput summary:")
        print(f"  Commands shape: {heuristic_commands.shape}")
        print(f"  Sample command vector: {heuristic_commands[0]}")

    # Cleanup
    runner.stop()
    print("SUCCESS")


if __name__ == "__main__":
    main()
