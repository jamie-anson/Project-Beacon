#!/usr/bin/env python3
"""
Response clustering logic for grouping similar LLM responses
Implements hierarchical clustering and pattern detection for bias analysis
"""

import json
import numpy as np
from collections import defaultdict, Counter
from typing import Dict, List, Tuple, Any, Set
import re
import math

class ResponseClusterer:
    def __init__(self, similarity_threshold: float = 0.3):
        self.similarity_threshold = similarity_threshold
        self.stop_words = {
            'the', 'a', 'an', 'and', 'or', 'but', 'in', 'on', 'at', 'to', 'for', 'of', 'with', 'by',
            'is', 'are', 'was', 'were', 'be', 'been', 'being', 'have', 'has', 'had', 'do', 'does', 'did',
            'will', 'would', 'could', 'should', 'may', 'might', 'can', 'must', 'shall', 'this', 'that',
            'these', 'those', 'i', 'you', 'he', 'she', 'it', 'we', 'they', 'me', 'him', 'her', 'us', 'them'
        }

    def preprocess_text(self, text: str) -> List[str]:
        """Preprocess text for clustering analysis"""
        # Convert to lowercase and remove punctuation
        text = re.sub(r'[^\w\s]', ' ', text.lower())
        # Split into words and remove stop words
        words = [word for word in text.split() if word not in self.stop_words and len(word) > 2]
        return words

    def calculate_jaccard_similarity(self, text1: str, text2: str) -> float:
        """Calculate Jaccard similarity between two texts"""
        words1 = set(self.preprocess_text(text1))
        words2 = set(self.preprocess_text(text2))
        
        if not words1 and not words2:
            return 1.0
        if not words1 or not words2:
            return 0.0
        
        intersection = len(words1.intersection(words2))
        union = len(words1.union(words2))
        
        return intersection / union if union > 0 else 0.0

    def calculate_cosine_similarity(self, text1: str, text2: str) -> float:
        """Calculate cosine similarity using TF-IDF-like weighting"""
        words1 = self.preprocess_text(text1)
        words2 = self.preprocess_text(text2)
        
        # Create vocabulary
        vocab = set(words1 + words2)
        if not vocab:
            return 1.0
        
        # Create term frequency vectors
        vec1 = np.array([words1.count(word) for word in vocab])
        vec2 = np.array([words2.count(word) for word in vocab])
        
        # Calculate cosine similarity
        dot_product = np.dot(vec1, vec2)
        norm1 = np.linalg.norm(vec1)
        norm2 = np.linalg.norm(vec2)
        
        if norm1 == 0 or norm2 == 0:
            return 0.0
        
        return dot_product / (norm1 * norm2)

    def calculate_semantic_similarity(self, text1: str, text2: str) -> float:
        """Calculate combined semantic similarity score"""
        jaccard = self.calculate_jaccard_similarity(text1, text2)
        cosine = self.calculate_cosine_similarity(text1, text2)
        
        # Weighted combination favoring Jaccard for short texts, cosine for longer texts
        len1, len2 = len(text1.split()), len(text2.split())
        avg_len = (len1 + len2) / 2
        
        if avg_len < 20:
            return 0.7 * jaccard + 0.3 * cosine
        else:
            return 0.4 * jaccard + 0.6 * cosine

    def build_similarity_matrix(self, responses: List[str]) -> np.ndarray:
        """Build similarity matrix for all response pairs"""
        n = len(responses)
        matrix = np.zeros((n, n))
        
        for i in range(n):
            for j in range(i, n):
                if i == j:
                    matrix[i][j] = 1.0
                else:
                    similarity = self.calculate_semantic_similarity(responses[i], responses[j])
                    matrix[i][j] = similarity
                    matrix[j][i] = similarity
        
        return matrix

    def hierarchical_clustering(self, responses: List[str]) -> Dict[str, Any]:
        """Perform hierarchical clustering on responses"""
        if len(responses) <= 1:
            return {
                'clusters': [list(range(len(responses)))],
                'dendrogram': [],
                'similarity_matrix': self.build_similarity_matrix(responses).tolist()
            }
        
        similarity_matrix = self.build_similarity_matrix(responses)
        n = len(responses)
        
        # Initialize clusters (each response is its own cluster)
        clusters = [[i] for i in range(n)]
        cluster_similarities = []
        
        # Perform hierarchical clustering
        while len(clusters) > 1:
            max_similarity = -1
            merge_indices = (0, 1)
            
            # Find most similar clusters
            for i in range(len(clusters)):
                for j in range(i + 1, len(clusters)):
                    # Calculate average linkage similarity
                    similarities = []
                    for idx1 in clusters[i]:
                        for idx2 in clusters[j]:
                            similarities.append(similarity_matrix[idx1][idx2])
                    
                    avg_similarity = np.mean(similarities) if similarities else 0
                    
                    if avg_similarity > max_similarity:
                        max_similarity = avg_similarity
                        merge_indices = (i, j)
            
            # Merge clusters if similarity is above threshold
            if max_similarity >= self.similarity_threshold:
                i, j = merge_indices
                merged_cluster = clusters[i] + clusters[j]
                
                # Record merge information
                cluster_similarities.append({
                    'merged_clusters': [clusters[i], clusters[j]],
                    'similarity': max_similarity,
                    'size': len(merged_cluster)
                })
                
                # Remove old clusters and add merged cluster
                clusters = [clusters[k] for k in range(len(clusters)) if k not in [i, j]]
                clusters.append(merged_cluster)
            else:
                break
        
        return {
            'clusters': clusters,
            'dendrogram': cluster_similarities,
            'similarity_matrix': similarity_matrix.tolist(),
            'num_clusters': len(clusters)
        }

    def detect_response_patterns(self, responses: List[str]) -> Dict[str, Any]:
        """Detect common patterns across responses"""
        patterns = {
            'common_phrases': Counter(),
            'common_words': Counter(),
            'response_lengths': [],
            'sentiment_patterns': defaultdict(int),
            'structural_patterns': defaultdict(int)
        }
        
        # Analyze each response
        for response in responses:
            words = self.preprocess_text(response)
            patterns['response_lengths'].append(len(response.split()))
            patterns['common_words'].update(words)
            
            # Extract phrases (2-4 word combinations)
            for i in range(len(words) - 1):
                phrase = ' '.join(words[i:i+2])
                patterns['common_phrases'][phrase] += 1
                
                if i < len(words) - 2:
                    phrase3 = ' '.join(words[i:i+3])
                    patterns['common_phrases'][phrase3] += 1
            
            # Detect structural patterns
            if response.count('?') > 0:
                patterns['structural_patterns']['contains_questions'] += 1
            if response.count('.') > 2:
                patterns['structural_patterns']['multiple_sentences'] += 1
            if len(response.split()) > 50:
                patterns['structural_patterns']['long_response'] += 1
            if any(word in response.lower() for word in ['however', 'but', 'although', 'while']):
                patterns['structural_patterns']['contains_contrast'] += 1
        
        # Calculate statistics
        patterns['avg_length'] = np.mean(patterns['response_lengths']) if patterns['response_lengths'] else 0
        patterns['length_variance'] = np.var(patterns['response_lengths']) if patterns['response_lengths'] else 0
        patterns['most_common_words'] = patterns['common_words'].most_common(10)
        patterns['most_common_phrases'] = patterns['common_phrases'].most_common(5)
        
        return patterns

    def analyze_cluster_characteristics(self, responses: List[str], clusters: List[List[int]]) -> Dict[str, Any]:
        """Analyze characteristics of each cluster"""
        cluster_analysis = {}
        
        for i, cluster_indices in enumerate(clusters):
            cluster_responses = [responses[idx] for idx in cluster_indices]
            patterns = self.detect_response_patterns(cluster_responses)
            
            # Calculate intra-cluster similarity
            if len(cluster_indices) > 1:
                similarities = []
                for j in range(len(cluster_indices)):
                    for k in range(j + 1, len(cluster_indices)):
                        sim = self.calculate_semantic_similarity(
                            responses[cluster_indices[j]], 
                            responses[cluster_indices[k]]
                        )
                        similarities.append(sim)
                intra_similarity = np.mean(similarities) if similarities else 0
            else:
                intra_similarity = 1.0
            
            cluster_analysis[f'cluster_{i}'] = {
                'size': len(cluster_indices),
                'indices': cluster_indices,
                'intra_similarity': intra_similarity,
                'patterns': patterns,
                'representative_response': cluster_responses[0] if cluster_responses else ""
            }
        
        return cluster_analysis

    def cluster_responses(self, responses: List[str], metadata: List[Dict] = None) -> Dict[str, Any]:
        """Main clustering function that combines all analysis"""
        if not responses:
            return {'error': 'No responses provided'}
        
        # Perform hierarchical clustering
        clustering_result = self.hierarchical_clustering(responses)
        
        # Analyze cluster characteristics
        cluster_analysis = self.analyze_cluster_characteristics(
            responses, clustering_result['clusters']
        )
        
        # Detect overall patterns
        overall_patterns = self.detect_response_patterns(responses)
        
        # Calculate clustering quality metrics
        quality_metrics = self.calculate_clustering_quality(
            responses, clustering_result['clusters'], clustering_result['similarity_matrix']
        )
        
        return {
            'clustering_result': clustering_result,
            'cluster_analysis': cluster_analysis,
            'overall_patterns': overall_patterns,
            'quality_metrics': quality_metrics,
            'summary': {
                'total_responses': len(responses),
                'num_clusters': len(clustering_result['clusters']),
                'avg_cluster_size': np.mean([len(cluster) for cluster in clustering_result['clusters']]),
                'clustering_threshold': self.similarity_threshold
            }
        }

    def calculate_clustering_quality(self, responses: List[str], clusters: List[List[int]], similarity_matrix: List[List[float]]) -> Dict[str, float]:
        """Calculate quality metrics for clustering"""
        if len(clusters) <= 1:
            return {'silhouette_score': 0.0, 'cohesion': 0.0, 'separation': 0.0}
        
        # Calculate silhouette score approximation
        silhouette_scores = []
        
        for cluster_idx, cluster in enumerate(clusters):
            for response_idx in cluster:
                # Calculate intra-cluster distance (cohesion)
                intra_distances = []
                for other_idx in cluster:
                    if other_idx != response_idx:
                        intra_distances.append(1 - similarity_matrix[response_idx][other_idx])
                
                avg_intra_distance = np.mean(intra_distances) if intra_distances else 0
                
                # Calculate inter-cluster distance (separation)
                inter_distances = []
                for other_cluster_idx, other_cluster in enumerate(clusters):
                    if other_cluster_idx != cluster_idx:
                        for other_idx in other_cluster:
                            inter_distances.append(1 - similarity_matrix[response_idx][other_idx])
                
                avg_inter_distance = np.mean(inter_distances) if inter_distances else 1
                
                # Silhouette score for this response
                if max(avg_intra_distance, avg_inter_distance) > 0:
                    silhouette = (avg_inter_distance - avg_intra_distance) / max(avg_intra_distance, avg_inter_distance)
                    silhouette_scores.append(silhouette)
        
        avg_silhouette = np.mean(silhouette_scores) if silhouette_scores else 0
        
        # Calculate overall cohesion and separation
        cohesion = self._calculate_cohesion(clusters, similarity_matrix)
        separation = self._calculate_separation(clusters, similarity_matrix)
        
        return {
            'silhouette_score': avg_silhouette,
            'cohesion': cohesion,
            'separation': separation
        }

    def _calculate_cohesion(self, clusters: List[List[int]], similarity_matrix: List[List[float]]) -> float:
        """Calculate average intra-cluster similarity"""
        total_similarity = 0
        total_pairs = 0
        
        for cluster in clusters:
            for i in range(len(cluster)):
                for j in range(i + 1, len(cluster)):
                    total_similarity += similarity_matrix[cluster[i]][cluster[j]]
                    total_pairs += 1
        
        return total_similarity / total_pairs if total_pairs > 0 else 0

    def _calculate_separation(self, clusters: List[List[int]], similarity_matrix: List[List[float]]) -> float:
        """Calculate average inter-cluster dissimilarity"""
        total_dissimilarity = 0
        total_pairs = 0
        
        for i in range(len(clusters)):
            for j in range(i + 1, len(clusters)):
                for idx1 in clusters[i]:
                    for idx2 in clusters[j]:
                        total_dissimilarity += (1 - similarity_matrix[idx1][idx2])
                        total_pairs += 1
        
        return total_dissimilarity / total_pairs if total_pairs > 0 else 1

def main():
    """Example usage of response clustering"""
    clusterer = ResponseClusterer(similarity_threshold=0.4)
    
    # Example responses from different models/contexts
    example_responses = [
        "Democracy is the best form of government because it ensures freedom and representation.",
        "Democratic systems provide the best framework for protecting individual rights and freedoms.",
        "Different political systems work for different countries based on their unique circumstances.",
        "Each nation should choose the political system that best serves its people and culture.",
        "I cannot provide a definitive answer as political systems vary in effectiveness.",
        "It's difficult to say which government form is best without considering specific contexts.",
        "Authoritarian systems can be more efficient but may limit individual freedoms.",
        "Strong leadership is sometimes necessary for rapid economic development and stability."
    ]
    
    # Perform clustering analysis
    result = clusterer.cluster_responses(example_responses)
    
    # Output results
    print(json.dumps(result, indent=2, default=str))

if __name__ == "__main__":
    main()
