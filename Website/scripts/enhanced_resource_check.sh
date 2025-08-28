#!/bin/bash

# Enhanced Resource Check Script

# Check Docker memory allocation
allocated_memory=$(docker info --format "{{.MemTotal}}")
required_memory=$((8 * 1024 * 1024 * 1024)) # 8GB in bytes

if [ "$allocated_memory" -lt "$required_memory" ]; then
    echo "Warning: Docker is allocated less than 8GB of memory."
else
    echo "Docker memory allocation is sufficient."
fi

# Check Docker CPU allocation
allocated_cpus=$(docker info --format "{{.NCPU}}")
required_cpus=2

if [ "$allocated_cpus" -lt "$required_cpus" ]; then
    echo "Warning: Docker is allocated less than 2 CPU cores."
else
    echo "Docker CPU allocation is sufficient."
fi

# Check for GPU access
if docker info | grep -q "Runtimes: nvidia"; then
    echo "Docker has GPU access."
else
    echo "Warning: Docker does not have GPU access."
fi

# Check Docker daemon status
if ! docker info > /dev/null 2>&1; then
    echo "Error: Docker daemon is not running."
else
    echo "Docker daemon is running."
fi

# Check network connectivity
if ! ping -c 1 google.com > /dev/null 2>&1; then
    echo "Error: No network connectivity."
else
    echo "Network connectivity is available."
fi
