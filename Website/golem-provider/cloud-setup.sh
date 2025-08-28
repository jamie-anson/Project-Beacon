#!/bin/bash

# Golem Provider Cloud Setup Script
# For x86_64 cloud servers (DigitalOcean, Linode, AWS, etc.)

set -e

echo "=== Golem Provider Cloud Setup ==="

# Update system
sudo apt-get update
sudo apt-get upgrade -y

# Install Docker
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg
echo "deb [arch=amd64 signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
sudo apt-get update
sudo apt-get install -y docker-ce docker-ce-cli containerd.io

# Add user to docker group
sudo usermod -aG docker $USER

# Install Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/download/v2.20.0/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

# Install Yagna directly (faster than Docker for cloud)
curl -sSf https://join.golem.network/as-provider | bash

# Add to PATH
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc

echo "=== Setup Complete ==="
echo "Next steps:"
echo "1. Start Yagna: yagna service run --api-allow-origin='*' &"
echo "2. Create app key: yagna app-key create requestor"
echo "3. Get node ID: yagna id show"
echo "4. Get testnet GLM: https://faucet.testnet.golem.network/"
echo "5. Configure provider: yagna provider preset create --preset-name beacon-provider --exe-unit wasmtime --pricing linear --price-duration 0.1 --price-cpu 0.1 --price-initial 0.0"
echo "6. Start provider: yagna provider run"
