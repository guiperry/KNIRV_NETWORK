# Guide: Deploying and Running a WSE Simulation

This guide provides a walkthrough on how to deploy and run a simulation on the Cerebras Wafer-Scale Engine (WSE) using the Cerebras SDK. The SDK includes the Cerebras Fabric Simulator, which allows you to run and debug your CSL (Cerebras Software Language) programs on a local CPU without needing access to a physical CS system.

## Prerequisites

Before running a simulation, ensure you have the following:

1.  The Cerebras SDK is installed and accessible.
2.  Your CSL program, consisting of at least a layout file (e.g., `layout.csl`) and a program for the processing elements (e.g., `pe_program.csl`).

## Step 1: Compiling CSL Code

The first step is to compile your CSL source files into ELF binaries that the simulator can execute. This is done using the `cslc` compiler.

The `cslc` tool takes your layout file as input and various parameters to configure the compilation.

### Example Compilation Command:

```bash
# Path to the cslc compiler
CSLC_COMPILER="<path_to_your_sdk>/bin/cslc"

# Output directory for the compiled files
OUTPUT_DIR="my_program_elfs"

${CSLC_COMPILER} layout.csl \
    --fabric-dims=20,20 \
    --fabric-offsets=1,1 \
    --params=width:10,height:10 \
    -o=${OUTPUT_DIR} \
    --arch=wse2 \
    --memcpy
```

### Key Compiler Flags:

*   `--fabric-dims`: Specifies the dimensions of the fabric (the grid of PEs).
*   `--fabric-offsets`: Sets the offset of your application on the fabric.
*   `--params`: A list of key-value pairs to parameterize your CSL code.
*   `-o`: Specifies the output directory for the compiled ELF files and metadata.
*   `--arch`: The target architecture (e.g., `wse2` for Wafer-Scale Engine 2).
*   `--memcpy`: Enables the `memcpy` infrastructure for data transfer.

After a successful compilation, the output directory will contain the ELF files (e.g., `bin/out_*.elf`) and a JSON metadata file (`out.json`).

## Step 2: Running the Simulation with `SdkRuntime`

With the compiled ELF files, you can now run the simulation using a Python script and the `SdkRuntime` class.

### Example Python script (`run_simulation.py`):

```python
import numpy as np
from cerebras.sdk.runtime.sdkruntimepybind import SdkRuntime, MemcpyDataType, MemcpyOrder

# The directory containing the compiled ELF files
COMPILED_DIR = "my_program_elfs"

# Initialize the SdkRuntime for simulation
# When cmaddr is not provided, it defaults to simulation mode.
runner = SdkRuntime(COMPILED_DIR)

# Get symbols for variables in your CSL code
sym_input_data = runner.get_id("input_data")
sym_output_data = runner.get_id("output_data")

# Load the program onto the simulator
runner.load()

# Start the simulation
runner.run()

# --- Interact with the simulation ---

# Create some input data
input_array = np.ones(100, dtype=np.float32)

# Copy data from Host to Device (H2D)
runner.memcpy_h2d(sym_input_data, input_array, 0, 0, 10, 10, 1,
                  streaming=False, data_type=MemcpyDataType.MEMCPY_32BIT,
                  order=MemcpyOrder.ROW_MAJOR, nonblock=False)

# Launch a function on the device
runner.launch("my_csl_function", nonblock=False)

# Prepare an array to receive the results
output_array = np.zeros(100, dtype=np.float32)

# Copy data from Device to Host (D2H)
runner.memcpy_d2h(output_array, sym_output_data, 0, 0, 10, 10, 1,
                  streaming=False, data_type=MemcpyDataType.MEMCPY_32BIT,
                  order=MemcpyOrder.ROW_MAJOR, nonblock=False)


# Stop the simulation
runner.stop()

print("Simulation finished. Output data:")
print(output_array)

```

### Running the script:

To execute the simulation, simply run the Python script:

```bash
cs_python run_simulation.py
```

## Simulation Artifacts

When the simulation completes, it generates several files in the directory where you ran the script:

*   `sim.log`: A detailed log of the simulation, including PE activity and memory accesses. This is very useful for debugging.
*   `simfab_traces/`: A directory containing detailed trace files of the fabric activity.

This guide provides a basic overview of the simulation deployment process. For more advanced features and detailed API documentation, please refer to the official Cerebras SDK documentation and the examples provided in the `HEART/CERERAS_SDK/examples` directory.
