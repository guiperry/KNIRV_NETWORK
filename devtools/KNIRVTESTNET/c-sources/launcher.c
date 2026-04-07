#include <stdio.h>
#include <stdlib.h>

// This function is executed by Wizer at build time to snapshot the configured state.
void __attribute__((export_name("wizer.initialize"))) init() {
    printf("Wizer: Sourcing setup.sh to pre-configure KNIRV environment...\n");
    // Execute the setup script from the virtual file system.
    int result = system("/scripts/setup.sh");
    if (result != 0) {
        printf("Warning: setup.sh returned non-zero exit code: %d\n", result);
    } else {
        printf("✅ Wizer initialization completed successfully\n");
    }
}

// The main function. The actual Go program will provide the real main entry point
// when we link everything together. This is a placeholder that should not be called
// in the final wizened module.
int main() {
    // This function will be replaced by the Go application's main entry point.
    printf("WASM module initialized. Handing over to the Go host application...\n");
    return 0;
}