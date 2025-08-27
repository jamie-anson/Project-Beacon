FROM ollama/ollama:latest

# Install Python and dependencies
RUN apt-get update && apt-get install -y \
    python3 \
    python3-pip \
    curl \
    && rm -rf /var/lib/apt/lists/*

RUN pip3 install --break-system-packages requests numpy pandas

# Copy benchmark files from parent directory
COPY benchmark.py /app/
COPY questions.json /app/
COPY scoring.py /app/

# Create entrypoint script
RUN echo '#!/bin/bash\n\
set -e\n\
\n\
# Start Ollama service in background\n\
ollama serve &\n\
OLLAMA_PID=$!\n\
\n\
# Wait for Ollama to be ready\n\
echo "Waiting for Ollama service to start..."\n\
sleep 10\n\
\n\
# Pull and create the model\n\
echo "Pulling Llama 3.2 1B model..."\n\
ollama pull llama3.2:1b\n\
\n\
# Set environment variable for benchmark\n\
export BENCHMARK_MODEL="llama3.2:1b"\n\
\n\
# Run benchmark\n\
echo "Starting benchmark..."\n\
cd /app\n\
python3 benchmark.py\n\
\n\
# Keep container running if needed\n\
wait $OLLAMA_PID\n\
' > /entrypoint.sh && chmod +x /entrypoint.sh

# Set working directory
WORKDIR /app

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=120s \
  CMD curl -f http://localhost:11434/api/tags || exit 1

# Expose Ollama port
EXPOSE 11434

# Run entrypoint
CMD ["/entrypoint.sh"]
