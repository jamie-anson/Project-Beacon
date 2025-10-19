#!/usr/bin/env python3
"""
Analyze Modal inference metrics to understand performance characteristics
Usage: python scripts/analyze-modal-metrics.py logs.txt
"""

import json
import sys
from collections import defaultdict
from pathlib import Path

def parse_metrics(log_file):
    """Extract [METRICS] lines from Modal logs"""
    metrics = []
    
    with open(log_file, 'r') as f:
        for line in f:
            if '[METRICS]' in line:
                try:
                    # Extract JSON after [METRICS] marker
                    json_str = line.split('[METRICS]')[1].strip()
                    data = json.loads(json_str)
                    metrics.append(data)
                except (json.JSONDecodeError, IndexError) as e:
                    print(f"Warning: Failed to parse line: {e}", file=sys.stderr)
                    continue
    
    return metrics

def analyze_metrics(metrics):
    """Generate performance analysis report"""
    
    if not metrics:
        print("❌ No metrics found in log file")
        return
    
    print(f"\n{'='*60}")
    print(f"📊 MODAL INFERENCE PERFORMANCE ANALYSIS")
    print(f"{'='*60}\n")
    
    print(f"Total Inferences: {len(metrics)}\n")
    
    # Group by region
    by_region = defaultdict(list)
    for m in metrics:
        by_region[m['region']].append(m)
    
    # Analyze each region
    for region in sorted(by_region.keys()):
        region_metrics = by_region[region]
        
        print(f"\n{'─'*60}")
        print(f"🌍 Region: {region.upper()}")
        print(f"{'─'*60}")
        
        # Warm vs Cold
        warm_count = sum(1 for m in region_metrics if m.get('container_status') == 'warm')
        cold_count = sum(1 for m in region_metrics if m.get('container_status') == 'cold')
        total = len(region_metrics)
        
        warm_pct = (warm_count / total * 100) if total > 0 else 0
        cold_pct = (cold_count / total * 100) if total > 0 else 0
        
        print(f"\n📦 Container Status:")
        print(f"  Warm Starts: {warm_count:3d} ({warm_pct:5.1f}%)")
        print(f"  Cold Starts: {cold_count:3d} ({cold_pct:5.1f}%)")
        
        # Status indicator
        if cold_pct > 60:
            status = "🔴 HIGH - Consider keep_warm=1"
        elif cold_pct > 40:
            status = "🟡 MEDIUM - Try 30-min scaledown"
        elif cold_pct > 20:
            status = "🟢 ACCEPTABLE - Current setup OK"
        else:
            status = "✅ EXCELLENT - Well optimized"
        print(f"  Status: {status}")
        
        # Inference times
        warm_times = [m['inference_time_ms'] for m in region_metrics 
                      if m.get('container_status') == 'warm' and m.get('status') == 'success']
        cold_times = [m['inference_time_ms'] for m in region_metrics 
                      if m.get('container_status') == 'cold' and m.get('status') == 'success']
        
        if warm_times:
            avg_warm = sum(warm_times) / len(warm_times)
            p50_warm = sorted(warm_times)[len(warm_times)//2]
            p95_warm = sorted(warm_times)[int(len(warm_times)*0.95)] if len(warm_times) > 1 else warm_times[0]
            
            print(f"\n⚡ Warm Inference Times:")
            print(f"  Average: {avg_warm:7.1f}ms")
            print(f"  P50:     {p50_warm:7.1f}ms")
            print(f"  P95:     {p95_warm:7.1f}ms")
        
        if cold_times:
            avg_cold = sum(cold_times) / len(cold_times)
            p50_cold = sorted(cold_times)[len(cold_times)//2]
            p95_cold = sorted(cold_times)[int(len(cold_times)*0.95)] if len(cold_times) > 1 else cold_times[0]
            
            print(f"\n🧊 Cold Inference Times:")
            print(f"  Average: {avg_cold:7.1f}ms")
            print(f"  P50:     {p50_cold:7.1f}ms")
            print(f"  P95:     {p95_cold:7.1f}ms")
            
            if warm_times:
                slowdown = (avg_cold / avg_warm) if avg_warm > 0 else 0
                print(f"  Slowdown: {slowdown:.1f}x slower than warm")
        
        # Model breakdown
        by_model = defaultdict(list)
        for m in region_metrics:
            by_model[m['model']].append(m)
        
        print(f"\n🤖 By Model:")
        for model in sorted(by_model.keys()):
            model_metrics = by_model[model]
            model_warm = sum(1 for m in model_metrics if m.get('container_status') == 'warm')
            model_total = len(model_metrics)
            model_warm_pct = (model_warm / model_total * 100) if model_total > 0 else 0
            
            model_times = [m['inference_time_ms'] for m in model_metrics if m.get('status') == 'success']
            avg_time = sum(model_times) / len(model_times) if model_times else 0
            
            print(f"  {model:20s}: {model_total:3d} inferences, {model_warm_pct:5.1f}% warm, {avg_time:6.1f}ms avg")
        
        # GPU Memory
        gpu_memory = [m.get('gpu_memory_mb', 0) for m in region_metrics if m.get('gpu_memory_mb')]
        if gpu_memory:
            avg_gpu = sum(gpu_memory) / len(gpu_memory)
            max_gpu = max(gpu_memory)
            print(f"\n💾 GPU Memory:")
            print(f"  Average: {avg_gpu:7.1f}MB")
            print(f"  Peak:    {max_gpu:7.1f}MB")
    
    # Cross-region comparison
    print(f"\n{'='*60}")
    print(f"🌐 CROSS-REGION COMPARISON")
    print(f"{'='*60}\n")
    
    comparison = []
    for region in sorted(by_region.keys()):
        region_metrics = by_region[region]
        warm_count = sum(1 for m in region_metrics if m.get('container_status') == 'warm')
        total = len(region_metrics)
        warm_pct = (warm_count / total * 100) if total > 0 else 0
        
        warm_times = [m['inference_time_ms'] for m in region_metrics 
                      if m.get('container_status') == 'warm' and m.get('status') == 'success']
        avg_warm = sum(warm_times) / len(warm_times) if warm_times else 0
        
        comparison.append((region, warm_pct, avg_warm, total))
    
    print(f"{'Region':<15} {'Warm %':>10} {'Avg Time':>12} {'Count':>8}")
    print(f"{'-'*50}")
    for region, warm_pct, avg_time, count in comparison:
        print(f"{region:<15} {warm_pct:9.1f}% {avg_time:10.1f}ms {count:7d}")
    
    # Recommendations
    print(f"\n{'='*60}")
    print(f"💡 RECOMMENDATIONS")
    print(f"{'='*60}\n")
    
    for region, warm_pct, avg_time, count in comparison:
        if warm_pct < 60:
            print(f"🔴 {region.upper()}:")
            print(f"   - Cold start rate too high ({100-warm_pct:.0f}%)")
            print(f"   - Recommendation: Enable keep_warm=1 during business hours ($150/month)")
        elif warm_pct < 80:
            print(f"🟡 {region.upper()}:")
            print(f"   - Moderate cold start rate ({100-warm_pct:.0f}%)")
            print(f"   - Recommendation: Increase scaledown_window to 1800s (30 min) - FREE")
        else:
            print(f"✅ {region.upper()}:")
            print(f"   - Good warm coverage ({warm_pct:.0f}%)")
            print(f"   - Recommendation: Current setup is working well")
    
    print(f"\n{'='*60}\n")

def main():
    if len(sys.argv) != 2:
        print("Usage: python analyze-modal-metrics.py <log_file>")
        print("\nExample:")
        print("  modal logs project-beacon-hf > logs.txt")
        print("  python scripts/analyze-modal-metrics.py logs.txt")
        sys.exit(1)
    
    log_file = sys.argv[1]
    
    if not Path(log_file).exists():
        print(f"❌ Error: Log file not found: {log_file}")
        sys.exit(1)
    
    print(f"📖 Reading logs from: {log_file}")
    metrics = parse_metrics(log_file)
    
    if not metrics:
        print("\n❌ No [METRICS] entries found in log file")
        print("\nMake sure you've:")
        print("  1. Deployed the updated modal_hf_multiregion.py with metrics logging")
        print("  2. Run some test jobs to generate metrics")
        print("  3. Captured Modal logs with: modal logs project-beacon-hf > logs.txt")
        sys.exit(1)
    
    analyze_metrics(metrics)

if __name__ == "__main__":
    main()
