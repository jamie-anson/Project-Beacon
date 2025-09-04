# Golem Provider Node Setup

This directory contains the Docker-based setup for running a Golem Provider node on Apple Silicon Macs using x86_64 emulation.

## Quick Start

1. **Build and start the provider:**
   ```bash
   cd golem-provider
   docker-compose up --build
   ```

2. **Monitor provider status:**
   ```bash
   docker-compose logs -f golem-provider
   ```

3. **Get provider node ID:**
   ```bash
   docker-compose exec golem-provider yagna id show
   ```

## Provider Configuration

The provider is configured with:
- **Preset name**: `beacon-provider`
- **Runtime**: WebAssembly (wasmtime)
- **Pricing**: Linear model
  - Duration: 0.1 GLM/hour
  - CPU: 0.1 GLM/hour  
  - Initial: 0.0 GLM

## Testnet Setup

To connect to testnet and get GLM tokens:

1. **Get your node ID:**
   ```bash
   docker-compose exec golem-provider yagna id show
   ```

2. **Request testnet GLM tokens:**
   - Visit: https://faucet.testnet.golem.network/
   - Enter your node ID
   - Request tokens

3. **Check balance:**
   ```bash
   docker-compose exec golem-provider yagna payment status
   ```

## Monitoring

- **Logs**: `./logs/` directory
- **Provider status**: `docker-compose exec golem-provider yagna provider status`
- **Payment status**: `docker-compose exec golem-provider yagna payment status`
- **Active agreements**: `docker-compose exec golem-provider yagna market agreements list`

## Testing with Beacon Containers

Once the provider is running, test with our benchmark containers:

1. **From your runner app (Terminal C):**
   ```bash
   curl -X POST http://localhost:8090/api/v1/jobs \
     -H "Content-Type: application/json" \
     -d '{
       "id": "test-local-provider",
       "version": "1.0",
       "benchmark": {
         "name": "text-generation",
         "category": "llm"
       },
       "container": {
         "image": "your-benchmark-image",
         "tag": "latest"
       },
       "input": {
         "prompt": "Who are you?",
         "hash": "sha256:..."
       },
       "regions": ["local"],
       "constraints": {
         "max_duration": 300,
         "max_cost": 1.0
       }
     }'
   ```

## Troubleshooting

- **Provider not starting**: Check Docker daemon is running
- **No GLM tokens**: Visit testnet faucet
- **Port conflicts**: Ensure ports 7464/7465 are available
- **Performance issues**: Expected due to x86_64 emulation on ARM

## Cloud Alternative

For production use, deploy on x86_64 cloud server:
- DigitalOcean: 2-4 CPU, 4-8GB RAM droplet
- AWS: t3.medium instance
- Linode: 4GB shared CPU instance
