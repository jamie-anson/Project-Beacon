# Resource Requirements and Troubleshooting Guide

## Resource Requirements

### Docker Configuration
- **Memory**: Minimum 8GB, recommended 16GB
- **CPU Cores**: Minimum 2, recommended 4
- **GPU**: Optional, but recommended for faster processing

### Network
- Ensure stable internet connection for downloading models and dependencies

## Troubleshooting

### Insufficient Resources
- **Memory**: Increase Docker memory allocation via Docker settings
- **CPU**: Ensure at least 2 CPU cores are allocated to Docker
- **GPU**: Verify GPU access is enabled in Docker settings

### Docker Daemon Not Running
- Restart Docker from the system tray or command line
- Check Docker service status and logs for errors

### Network Connectivity Issues
- Verify internet connection
- Check firewall settings that might block Docker

### Common Errors
- **Timeouts**: Increase timeout settings in `benchmark.py` if necessary
- **Model Download Issues**: Prewarm models using provided Makefile commands

This guide should be included in the user onboarding process to assist with setup and troubleshooting.
