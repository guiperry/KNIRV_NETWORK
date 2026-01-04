# Copyright 2026 KNIRV-NEXUS
# SPDX-License-Identifier: GPL-3.0-or-later

# eBPF programs Makefile

# Compiler settings
CC ?= clang
CFLAGS ?= -O2 -g -target bpf -D__TARGET_ARCH_x86
INCLUDES ?= -I/usr/include/bpf

# Output directory
OUTPUT_DIR ?= ../../programs

# Source files
SRC_FILES := $(wildcard *.c)
OBJ_FILES := $(SRC_FILES:.c=.o)

# Generate vmlinux.h from BTF
vmlinux.h:
	bpftool btf dump file /sys/kernel/btf/vmlinux format c > vmlinux.h

# Build all eBPF programs
all: vmlinux.h $(OBJ_FILES)

# Build individual .o files
%.o: %.c vmlinux.h
	$(CC) $(CFLAGS) $(INCLUDES) -c $< -o $@

# Clean build artifacts
clean:
	rm -f *.o *.sha256 vmlinux.h

# Install programs to output directory
install: all
	mkdir -p $(OUTPUT_DIR)
	cp *.o $(OUTPUT_DIR)

# Generate checksums for verification
gen-checksums: all
	for obj in *.o; do \
		sha256sum "$$obj" > "$$obj.sha256"; \
		echo "Generated checksum for $$obj"; \
	done

.PHONY: all clean install gen-checksums