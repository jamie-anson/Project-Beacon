# LLM Benchmark helpers

COMPOSE=docker compose
VOLUME_NAME=ollama-cache
RESULTS_DIR=./results

# Allow overriding image registry and tag for all providers
REGISTRY ?= beacon
IMAGE_TAG ?= latest

.PHONY: help
help:
	@echo "Targets:";
	@echo "  volume              Create named volume '$(VOLUME_NAME)'";
	@echo "  build-all           Build all provider images ($(REGISTRY)/*:$(IMAGE_TAG))";
	@echo "  build-llama         Build llama image";
	@echo "  build-qwen          Build qwen image";
	@echo "  build-mistral       Build mistral image";
	@echo "  prewarm-all         Pull all models into the shared cache";
	@echo "  prewarm-llama       Pull llama3.2:1b into cache";
	@echo "  prewarm-qwen        Pull qwen2.5:1.5b into cache";
	@echo "  prewarm-mistral     Pull mistral:7b into cache";
	@echo "  run-llama           Build+prewarm then run llama benchmark with shared cache";
	@echo "  run-qwen            Build+prewarm then run qwen benchmark with shared cache";
	@echo "  run-mistral         Build+prewarm then run mistral benchmark with shared cache";
	@echo "  run-all             Build+prewarm then run all models (parallel)";

volume:
	docker volume create $(VOLUME_NAME)
	@mkdir -p $(RESULTS_DIR)

# Internal helper to pull a model by overriding entrypoint
# Usage: make _pull MODEL=llama3.2:1b IMAGE=beacon/llama-3.2-1b:latest
.PHONY: _pull
_pull:
	docker run --rm \
	  --entrypoint /bin/sh \
	  -v $(VOLUME_NAME):/root/.ollama \
	  $(IMAGE) -lc 'set -e; ollama serve & pid=$$!; sleep 6; ollama pull $(MODEL); kill $$pid || true'

prewarm-all: volume prewarm-llama prewarm-qwen prewarm-mistral

prewarm-llama: volume
	$(MAKE) _pull MODEL=llama3.2:1b IMAGE=$(REGISTRY)/llama-3.2-1b:$(IMAGE_TAG)

prewarm-qwen: volume
	$(MAKE) _pull MODEL=qwen2.5:1.5b IMAGE=$(REGISTRY)/qwen-2.5-1.5b:$(IMAGE_TAG)

prewarm-mistral: volume
	$(MAKE) _pull MODEL=mistral:7b IMAGE=$(REGISTRY)/mistral-7b:$(IMAGE_TAG)

# Build images
.PHONY: build-all build-llama build-qwen build-mistral
build-all: build-llama build-qwen build-mistral

build-llama:
	docker build -t $(REGISTRY)/llama-3.2-1b:$(IMAGE_TAG) llama-3.2-1b

build-qwen:
	docker build -t $(REGISTRY)/qwen-2.5-1.5b:$(IMAGE_TAG) qwen-2.5-1.5b

build-mistral:
	docker build -t $(REGISTRY)/mistral-7b:$(IMAGE_TAG) mistral-7b

run-llama: build-llama prewarm-llama volume
	$(COMPOSE) --profile llama run --rm llama

run-qwen: build-qwen prewarm-qwen volume
	$(COMPOSE) --profile qwen run --rm qwen

run-mistral: build-mistral prewarm-mistral volume
	$(COMPOSE) --profile mistral run --rm mistral

run-all: build-all prewarm-all volume
	$(MAKE) -j3 run-llama run-qwen run-mistral
