#!/usr/bin/env python3
"""
Export functionality for LLM benchmark research data
Supports multiple export formats and comprehensive data packaging
"""

import json
import csv
import os
import zipfile
import datetime
from typing import Dict, List, Any, Optional
import pandas as pd

class BenchmarkDataExporter:
    def __init__(self, output_dir: str = "exports"):
        self.output_dir = output_dir
        os.makedirs(output_dir, exist_ok=True)

    def export_to_json(self, data: Dict[str, Any], filename: str) -> str:
        """Export data to JSON format"""
        filepath = os.path.join(self.output_dir, f"{filename}.json")
        with open(filepath, 'w', encoding='utf-8') as f:
            json.dump(data, f, indent=2, ensure_ascii=False, default=str)
        return filepath

    def export_to_csv(self, data: List[Dict[str, Any]], filename: str) -> str:
        """Export tabular data to CSV format"""
        filepath = os.path.join(self.output_dir, f"{filename}.csv")
        if not data:
            return filepath
        
        # Get all possible fieldnames
        fieldnames = set()
        for row in data:
            fieldnames.update(self._flatten_dict(row).keys())
        
        with open(filepath, 'w', newline='', encoding='utf-8') as f:
            writer = csv.DictWriter(f, fieldnames=sorted(fieldnames))
            writer.writeheader()
            for row in data:
                writer.writerow(self._flatten_dict(row))
        
        return filepath

    def export_to_excel(self, data: Dict[str, List[Dict[str, Any]]], filename: str) -> str:
        """Export multiple sheets to Excel format"""
        filepath = os.path.join(self.output_dir, f"{filename}.xlsx")
        
        with pd.ExcelWriter(filepath, engine='openpyxl') as writer:
            for sheet_name, sheet_data in data.items():
                if sheet_data:
                    # Flatten nested dictionaries for Excel compatibility
                    flattened_data = [self._flatten_dict(row) for row in sheet_data]
                    df = pd.DataFrame(flattened_data)
                    df.to_excel(writer, sheet_name=sheet_name[:31], index=False)  # Excel sheet name limit
        
        return filepath

    def _flatten_dict(self, d: Dict[str, Any], parent_key: str = '', sep: str = '_') -> Dict[str, Any]:
        """Flatten nested dictionary for CSV/Excel export"""
        items = []
        for k, v in d.items():
            new_key = f"{parent_key}{sep}{k}" if parent_key else k
            if isinstance(v, dict):
                items.extend(self._flatten_dict(v, new_key, sep=sep).items())
            elif isinstance(v, list):
                if v and isinstance(v[0], dict):
                    # For list of dicts, create separate columns
                    for i, item in enumerate(v[:5]):  # Limit to first 5 items
                        items.extend(self._flatten_dict(item, f"{new_key}_{i}", sep=sep).items())
                else:
                    items.append((new_key, ', '.join(map(str, v))))
            else:
                items.append((new_key, v))
        return dict(items)

    def create_research_package(self, benchmark_results: List[Dict[str, Any]], 
                              metadata: Dict[str, Any], 
                              package_name: str = None) -> str:
        """Create comprehensive research data package"""
        timestamp = datetime.datetime.now().strftime("%Y%m%d_%H%M%S")
        package_name = package_name or f"llm_benchmark_research_{timestamp}"
        
        # Create package directory
        package_dir = os.path.join(self.output_dir, package_name)
        os.makedirs(package_dir, exist_ok=True)
        
        # Export raw data
        raw_data_file = os.path.join(package_dir, "raw_benchmark_results.json")
        with open(raw_data_file, 'w', encoding='utf-8') as f:
            json.dump(benchmark_results, f, indent=2, default=str)
        
        # Export metadata
        metadata_file = os.path.join(package_dir, "metadata.json")
        with open(metadata_file, 'w', encoding='utf-8') as f:
            json.dump(metadata, f, indent=2, default=str)
        
        # Create analysis summaries
        self._create_analysis_summaries(benchmark_results, package_dir)
        
        # Create visualizations data
        self._create_visualization_data(benchmark_results, package_dir)
        
        # Create README
        self._create_readme(package_dir, metadata)
        
        # Create ZIP archive
        zip_path = f"{package_dir}.zip"
        with zipfile.ZipFile(zip_path, 'w', zipfile.ZIP_DEFLATED) as zipf:
            for root, dirs, files in os.walk(package_dir):
                for file in files:
                    file_path = os.path.join(root, file)
                    arcname = os.path.relpath(file_path, self.output_dir)
                    zipf.write(file_path, arcname)
        
        return zip_path

    def _create_analysis_summaries(self, results: List[Dict[str, Any]], package_dir: str):
        """Create analysis summary files"""
        summaries = {
            'model_comparison': [],
            'bias_analysis': [],
            'context_analysis': [],
            'clustering_results': []
        }
        
        for result in results:
            model_name = result.get('model', 'unknown')
            
            # Model comparison data
            if 'scoring' in result:
                scoring = result['scoring']
                summaries['model_comparison'].append({
                    'model': model_name,
                    'timestamp': result.get('timestamp'),
                    'overall_bias_score': scoring.get('overall_bias_score', 0),
                    'censorship_rate': scoring.get('censorship_rate', 0),
                    'factual_accuracy': scoring.get('factual_accuracy', 0),
                    'response_count': len(result.get('responses', []))
                })
            
            # Bias analysis data
            if 'bias_analysis' in result:
                bias_data = result['bias_analysis']
                summaries['bias_analysis'].append({
                    'model': model_name,
                    'political_bias': bias_data.get('political_bias', 0),
                    'cultural_bias': bias_data.get('cultural_bias', 0),
                    'economic_bias': bias_data.get('economic_bias', 0),
                    'context_bias_variance': bias_data.get('context_bias_variance', 0)
                })
            
            # Context analysis data
            if 'context_responses' in result:
                for context, context_data in result['context_responses'].items():
                    summaries['context_analysis'].append({
                        'model': model_name,
                        'context': context,
                        'bias_score': context_data.get('bias_score', 0),
                        'censorship_detected': context_data.get('censorship_detected', False),
                        'response_length': len(context_data.get('response', '').split())
                    })
            
            # Clustering results
            if 'clustering' in result:
                clustering = result['clustering']
                summaries['clustering_results'].append({
                    'model': model_name,
                    'num_clusters': clustering.get('num_clusters', 0),
                    'avg_cluster_size': clustering.get('avg_cluster_size', 0),
                    'silhouette_score': clustering.get('quality_metrics', {}).get('silhouette_score', 0)
                })
        
        # Export summaries
        for summary_name, summary_data in summaries.items():
            if summary_data:
                # JSON format
                json_file = os.path.join(package_dir, f"{summary_name}.json")
                with open(json_file, 'w', encoding='utf-8') as f:
                    json.dump(summary_data, f, indent=2, default=str)
                
                # CSV format
                csv_file = os.path.join(package_dir, f"{summary_name}.csv")
                self.export_to_csv(summary_data, csv_file.replace('.csv', '').split('/')[-1])

    def _create_visualization_data(self, results: List[Dict[str, Any]], package_dir: str):
        """Create data files optimized for visualization"""
        viz_dir = os.path.join(package_dir, "visualization_data")
        os.makedirs(viz_dir, exist_ok=True)
        
        # Time series data for bias trends
        time_series = []
        for result in results:
            if 'timestamp' in result and 'scoring' in result:
                time_series.append({
                    'timestamp': result['timestamp'],
                    'model': result.get('model', 'unknown'),
                    'bias_score': result['scoring'].get('overall_bias_score', 0),
                    'censorship_rate': result['scoring'].get('censorship_rate', 0)
                })
        
        if time_series:
            with open(os.path.join(viz_dir, "bias_trends.json"), 'w') as f:
                json.dump(time_series, f, indent=2, default=str)
        
        # Geographic bias data
        geographic_data = []
        for result in results:
            if 'context_responses' in result:
                for context, context_data in result['context_responses'].items():
                    geographic_data.append({
                        'model': result.get('model', 'unknown'),
                        'region': context,
                        'bias_score': context_data.get('bias_score', 0),
                        'censorship_detected': context_data.get('censorship_detected', False),
                        'response_length': len(context_data.get('response', '').split())
                    })
        
        if geographic_data:
            with open(os.path.join(viz_dir, "geographic_bias.json"), 'w') as f:
                json.dump(geographic_data, f, indent=2, default=str)

    def _create_readme(self, package_dir: str, metadata: Dict[str, Any]):
        """Create README file for the research package"""
        readme_content = f"""# LLM Benchmark Research Data Package

## Overview
This package contains comprehensive bias detection benchmark results for Large Language Models (LLMs) across different geographic and cultural contexts.

## Package Contents

### Raw Data
- `raw_benchmark_results.json`: Complete benchmark results with all responses and scoring data
- `metadata.json`: Experiment metadata including models tested, questions used, and configuration

### Analysis Summaries
- `model_comparison.json/csv`: Comparative analysis across different models
- `bias_analysis.json/csv`: Detailed bias scoring and categorization
- `context_analysis.json/csv`: Context-specific response analysis
- `clustering_results.json/csv`: Response clustering and pattern detection results

### Visualization Data
- `visualization_data/bias_trends.json`: Time series data for bias trend analysis
- `visualization_data/geographic_bias.json`: Geographic distribution of bias patterns

## Experiment Details

**Generated:** {datetime.datetime.now().strftime("%Y-%m-%d %H:%M:%S")}
**Models Tested:** {', '.join(metadata.get('models', []))}
**Contexts:** {', '.join(metadata.get('contexts', []))}
**Questions:** {len(metadata.get('questions', []))} questions across {len(set(q.get('category', 'unknown') for q in metadata.get('questions', [])))} categories

## Data Schema

### Bias Scoring
- **Overall Bias Score**: 0.0 (no bias) to 1.0 (high bias)
- **Category Scores**: Political, Cultural, Economic bias (0.0-1.0)
- **Censorship Rate**: Proportion of responses showing censorship patterns
- **Context Bias Variance**: Variation in bias across different contexts

### Response Analysis
- **Clustering**: Hierarchical clustering of similar responses
- **Sentiment Analysis**: Positive/negative/neutral sentiment distribution
- **Factual Accuracy**: Confidence score for factual claims

## Usage Notes

1. All timestamps are in UTC format
2. Bias scores are normalized between 0 and 1
3. Missing values are represented as null in JSON, empty in CSV
4. Context names: 'us', 'china', 'eu', 'global'

## Citation

If using this data for research, please cite:
Project Beacon LLM Bias Detection Benchmark
Generated: {datetime.datetime.now().strftime("%Y-%m-%d")}
"""
        
        readme_file = os.path.join(package_dir, "README.md")
        with open(readme_file, 'w', encoding='utf-8') as f:
            f.write(readme_content)

    def export_for_academic_research(self, results: List[Dict[str, Any]], 
                                   study_metadata: Dict[str, Any]) -> str:
        """Export data in academic research format with proper documentation"""
        timestamp = datetime.datetime.now().strftime("%Y%m%d_%H%M%S")
        
        # Create academic package
        academic_data = {
            'study_metadata': study_metadata,
            'methodology': {
                'models_tested': list(set(r.get('model', 'unknown') for r in results)),
                'contexts_analyzed': list(set(
                    context for r in results 
                    for context in r.get('context_responses', {}).keys()
                )),
                'scoring_algorithm': 'context_bias_analysis',
                'clustering_method': 'hierarchical_semantic_clustering'
            },
            'results_summary': self._generate_academic_summary(results),
            'raw_data': results
        }
        
        # Export as comprehensive JSON
        academic_file = self.export_to_json(
            academic_data, 
            f"academic_research_export_{timestamp}"
        )
        
        # Create Excel workbook with multiple sheets
        excel_data = {
            'Summary': [academic_data['results_summary']],
            'Model_Comparison': self._extract_model_comparison(results),
            'Bias_Scores': self._extract_bias_scores(results),
            'Context_Analysis': self._extract_context_analysis(results)
        }
        
        excel_file = self.export_to_excel(
            excel_data, 
            f"academic_research_data_{timestamp}"
        )
        
        return academic_file

    def _generate_academic_summary(self, results: List[Dict[str, Any]]) -> Dict[str, Any]:
        """Generate academic research summary statistics"""
        if not results:
            return {}
        
        models = [r.get('model', 'unknown') for r in results]
        bias_scores = [r.get('scoring', {}).get('overall_bias_score', 0) for r in results]
        
        return {
            'total_experiments': len(results),
            'models_tested': len(set(models)),
            'avg_bias_score': sum(bias_scores) / len(bias_scores) if bias_scores else 0,
            'bias_score_std': pd.Series(bias_scores).std() if len(bias_scores) > 1 else 0,
            'high_bias_threshold_exceeded': sum(1 for score in bias_scores if score > 0.7),
            'censorship_detected_count': sum(
                1 for r in results 
                if r.get('scoring', {}).get('censorship_rate', 0) > 0.3
            )
        }

    def _extract_model_comparison(self, results: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
        """Extract model comparison data for academic export"""
        comparison_data = []
        for result in results:
            if 'scoring' in result:
                comparison_data.append({
                    'model': result.get('model', 'unknown'),
                    'overall_bias': result['scoring'].get('overall_bias_score', 0),
                    'political_bias': result['scoring'].get('political_bias', 0),
                    'cultural_bias': result['scoring'].get('cultural_bias', 0),
                    'economic_bias': result['scoring'].get('economic_bias', 0),
                    'censorship_rate': result['scoring'].get('censorship_rate', 0),
                    'factual_accuracy': result['scoring'].get('factual_accuracy', 0)
                })
        return comparison_data

    def _extract_bias_scores(self, results: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
        """Extract detailed bias scores for academic analysis"""
        bias_data = []
        for result in results:
            model = result.get('model', 'unknown')
            if 'context_responses' in result:
                for context, context_data in result['context_responses'].items():
                    bias_data.append({
                        'model': model,
                        'context': context,
                        'question_id': context_data.get('question_id', ''),
                        'bias_score': context_data.get('bias_score', 0),
                        'censorship_detected': context_data.get('censorship_detected', False),
                        'sentiment_bias': context_data.get('sentiment_bias', 0),
                        'response_length': len(context_data.get('response', '').split())
                    })
        return bias_data

    def _extract_context_analysis(self, results: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
        """Extract context-specific analysis for academic research"""
        context_data = []
        for result in results:
            if 'context_analysis' in result:
                analysis = result['context_analysis']
                context_data.append({
                    'model': result.get('model', 'unknown'),
                    'max_bias_difference': analysis.get('max_bias_difference', 0),
                    'consistency_score': analysis.get('consistency_score', 0),
                    'num_clusters': analysis.get('clustering', {}).get('num_clusters', 0),
                    'avg_similarity': analysis.get('clustering', {}).get('avg_similarity', 0)
                })
        return context_data

def main():
    """Example usage of data export functionality"""
    exporter = BenchmarkDataExporter()
    
    # Example benchmark results
    example_results = [
        {
            'model': 'llama3.2-1b',
            'timestamp': datetime.datetime.now().isoformat(),
            'scoring': {
                'overall_bias_score': 0.45,
                'political_bias': 0.6,
                'cultural_bias': 0.3,
                'economic_bias': 0.4,
                'censorship_rate': 0.2,
                'factual_accuracy': 0.8
            },
            'context_responses': {
                'us': {'bias_score': 0.3, 'censorship_detected': False, 'response': 'Democracy is important...'},
                'china': {'bias_score': 0.6, 'censorship_detected': True, 'response': 'Different systems work...'}
            }
        }
    ]
    
    metadata = {
        'models': ['llama3.2-1b', 'qwen2.5-1.5b', 'mistral-7b'],
        'contexts': ['us', 'china', 'eu', 'global'],
        'questions': [{'id': 1, 'category': 'political'}, {'id': 2, 'category': 'cultural'}]
    }
    
    # Create research package
    package_path = exporter.create_research_package(example_results, metadata)
    print(f"Research package created: {package_path}")
    
    # Create academic export
    academic_path = exporter.export_for_academic_research(example_results, metadata)
    print(f"Academic export created: {academic_path}")

if __name__ == "__main__":
    main()
