#!/bin/bash
# Ping all models across regions to verify they're available

set -e

MODELS=("llama3.2-1b" "mistral-7b" "qwen2.5-1.5b")
REGIONS=(
  "US:https://jamie-anson--project-beacon-hf-us-ping-model.modal.run"
  "EU:https://jamie-anson--project-beacon-hf-eu-ping-model.modal.run"
)

echo "🔍 Pinging all models across regions..."
echo ""

for region_url in "${REGIONS[@]}"; do
  region="${region_url%%:*}"
  url="${region_url#*:}"
  
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo "📍 Region: $region"
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  
  for model in "${MODELS[@]}"; do
    echo -n "  🤖 $model: "
    
    response=$(curl -s "$url" \
      -X POST \
      -H "Content-Type: application/json" \
      -d "{\"model\":\"$model\"}" 2>/dev/null)
    
    success=$(echo "$response" | jq -r '.success // false')
    
    if [ "$success" = "true" ]; then
      hf_model=$(echo "$response" | jq -r '.hf_model')
      cached=$(echo "$response" | jq -r '.cached // false')
      gpu=$(echo "$response" | jq -r '.gpu // "N/A"')
      
      cache_status="❄️  cold"
      if [ "$cached" = "true" ]; then
        cache_status="🔥 cached"
      fi
      
      echo "✅ $hf_model ($cache_status, GPU: $gpu)"
    else
      error=$(echo "$response" | jq -r '.error // "Unknown error"')
      echo "❌ $error"
    fi
  done
  
  echo ""
done

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✨ Model ping complete!"
