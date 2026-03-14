#!/bin/python
import time
import sys
import os

print("🐍 KNIRV Python Data Processor running inside VFS!")
print(f"Arguments: {sys.argv}")
print(f"Environment: KNIRV_ENV={os.getenv('KNIRV_ENV', 'not set')}")
print(f"Python path: {os.getenv('PYTHON_EXECUTABLE', 'not set')}")
print(f"Working directory: {os.getcwd()}")

# Test VFS access
try:
    print("📁 VFS /bin directory contents:")
    for item in os.listdir("/bin"):
        print(f"  - {item}")
except Exception as e:
    print(f"⚠️ Could not list /bin: {e}")

for i in range(3):
    print(f"Processing KNIRV data... step {i+1}")
    time.sleep(0.5)  # Shorter sleep for faster execution
print("✅ KNIRV Python script finished successfully!")
