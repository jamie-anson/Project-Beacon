# Golem Provider Node - x86_64 Docker Container
FROM --platform=linux/amd64 ubuntu:22.04

# Install dependencies
RUN apt-get update && apt-get install -y \
    curl \
    wget \
    gnupg \
    software-properties-common \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Install Docker (needed for Golem runtime)
RUN curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg
RUN echo "deb [arch=amd64 signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu jammy stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null
RUN apt-get update && apt-get install -y docker-ce docker-ce-cli containerd.io

# Create golem user
RUN useradd -m -s /bin/bash golem
USER golem
WORKDIR /home/golem

# Create directories and prepare for runtime installation
RUN mkdir -p /home/golem/.local/bin

# Add Yagna to PATH
ENV PATH="/home/golem/.local/bin:${PATH}"

# Create directories for data persistence
RUN mkdir -p /home/golem/.local/share/yagna
RUN mkdir -p /home/golem/golem-logs

# Expose Yagna ports
EXPOSE 7465 7464

# Copy startup script
COPY --chown=golem:golem startup.sh /home/golem/startup.sh
RUN chmod +x /home/golem/startup.sh

CMD ["/home/golem/startup.sh"]
