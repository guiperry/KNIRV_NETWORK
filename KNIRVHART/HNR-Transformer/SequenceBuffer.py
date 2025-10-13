import csl_sdk.csl as csl
import csl_sdk.csl.mem as mem # For explicit on-chip memory management
import csl_sdk.csl.nn as nn

# Define constants (from Phase 2)
SEQUENCE_LENGTH = 64
EMBEDDING_DIM = 256
HEAD_DIM = EMBEDDING_DIM // 8 # 32

@csl.unit
def sequence_buffer_unit(
    # Input: Embedded vector for the current timestep (t)
    embedded_in: csl.Port(csl.Read, csl.VecF32(EMBEDDING_DIM)), 
    t_slice_in: csl.Port(csl.Read, csl.U32), 
    
    # Output: The full sequence buffered and ready for Attention (t-63 to t)
    # This output port will be triggered only when the buffer is full/updated.
    sequence_out: csl.Port(csl.Write, csl.VecF32(EMBEDDING_DIM * SEQUENCE_LENGTH)), 
    
    # Weights for Q, K, V projections (read-only inputs)
    W_Q: csl.Port(csl.Read, csl.MatF32(EMBEDDING_DIM, HEAD_DIM)),
    W_K: csl.Port(csl.Read, csl.MatF32(EMBEDDING_DIM, HEAD_DIM)),
    W_V: csl.Port(csl.Read, csl.MatF32(EMBEDDING_DIM, HEAD_DIM)),
):
    # Internal state: On-chip memory to hold the sliding window sequence
    # This memory holds the embedded input vectors for the last 64 timesteps.
    sequence_memory = mem.create_buffer(csl.VecF32(EMBEDDING_DIM), SEQUENCE_LENGTH)
    
    # Index pointer for the circular buffer (0 to 63)
    write_ptr = csl.State(csl.U32, initial_value=0)
    
    # ------------------- DATA PROCESSING LOGIC -------------------
    
    # 1. Update the Buffer (Stateful Write)
    csl.write_to_buffer(sequence_memory, embedded_in, write_ptr)
    
    # 2. Increment and Wrap the Pointer
    next_ptr = (write_ptr + 1) % SEQUENCE_LENGTH
    write_ptr.set(next_ptr)
    
    # 3. Compute Q, K, V Projections for ALL 64 vectors
    # This is the key parallel step on the WSE.
    Q_sequence = csl.VecF32(HEAD_DIM * SEQUENCE_LENGTH)
    K_sequence = csl.VecF32(HEAD_DIM * SEQUENCE_LENGTH)
    V_sequence = csl.VecF32(HEAD_DIM * SEQUENCE_LENGTH)
    
    # Read all 64 vectors from memory and apply the projection weights in parallel
    for i in range(SEQUENCE_LENGTH):
        # Read the i-th embedded vector from the circular buffer
        embedded_i = csl.read_from_buffer(sequence_memory, i) 
        
        # Compute projections (Matrix Multiply)
        Q_i = nn.matmul(embedded_i, W_Q)
        K_i = nn.matmul(embedded_i, W_K)
        V_i = nn.matmul(embedded_i, W_V)
        
        # Concatenate into the final Q, K, V sequences
        csl.set_slice(Q_sequence, i * HEAD_DIM, Q_i)
        csl.set_slice(K_sequence, i * HEAD_DIM, K_i)
        csl.set_slice(V_sequence, i * HEAD_DIM, V_i)

    # 4. Trigger Attention Calculation Unit
    # The actual Attention logic is too complex for a simple 'for' loop and
    # requires a dedicated, custom matrix unit, but we pass the data here.
    csl.write(sequence_out, Q_sequence, K_sequence, V_sequence) 

    # Note: The actual Attention Unit (matmul, softmax, matmul) will be a separate,
    # highly optimized CSL unit that takes these three 64-element sequences as input.
