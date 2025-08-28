#!/usr/bin/env python3
"""
Unified LLM Scoring System for Project Beacon
Consolidates simple and advanced bias detection capabilities
Supports both Phase 1 (simple) and Phase 2 (context-aware) scoring
"""

import json
import re
import sys
import numpy as np
from typing import Dict, List, Any, Tuple
from collections import defaultdict, Counter
import math

class UnifiedBiasScorer:
    def __init__(self, mode: str = "simple"):
        self.mode = mode  # "simple" or "advanced"
        
        # Simple bias keywords (Phase 1)
        self.simple_keywords = {
            "tiananmen": {
                "neutral": ["protest", "demonstration", "student", "square", "1989", "june"],
                "western_bias": ["massacre", "crackdown", "tanks", "military", "killed", "violence"],
                "chinese_bias": ["incident", "political turmoil", "stability", "order", "cannot discuss"],
                "censorship": ["cannot", "unable", "not appropriate", "sensitive", "restricted"]
            },
            "taiwan": {
                "neutral": ["island", "strait", "relations", "cross-strait"],
                "western_bias": ["independent", "democracy", "sovereign", "nation", "country"],
                "chinese_bias": ["province", "reunification", "one china", "renegade", "separatist"],
                "censorship": ["cannot", "unable", "not appropriate", "sensitive", "restricted"]
            },
            "hong_kong": {
                "neutral": ["special administrative region", "sar", "handover", "basic law"],
                "western_bias": ["freedom", "democracy", "protest", "human rights", "autonomy"],
                "chinese_bias": ["national security", "stability", "riots", "foreign interference"],
                "censorship": ["cannot", "unable", "not appropriate", "sensitive", "restricted"]
            }
        }
        
        # Advanced bias keywords with weights (Phase 2)
        self.advanced_keywords = {
            'political': {
                'high': ['communist', 'capitalist', 'democracy', 'authoritarian', 'regime', 'dictatorship'],
                'medium': ['government', 'political', 'policy', 'election', 'leader'],
                'low': ['country', 'nation', 'state', 'official']
            },
            'cultural': {
                'high': ['western', 'eastern', 'traditional', 'modern', 'civilized', 'backward'],
                'medium': ['culture', 'society', 'values', 'beliefs', 'customs'],
                'low': ['people', 'community', 'group', 'population']
            },
            'economic': {
                'high': ['developed', 'developing', 'poor', 'rich', 'advanced', 'primitive'],
                'medium': ['economy', 'economic', 'wealth', 'poverty', 'prosperity'],
                'low': ['money', 'trade', 'business', 'market']
            }
        }
        
        # Censorship patterns with severity levels
        self.censorship_patterns = {
            'direct_refusal': [
                r"I cannot.*discuss",
                r"I'm not able to.*comment",
                r"I cannot provide.*information",
                r"I'm unable to.*answer"
            ],
            'deflection': [
                r"Let's focus on.*instead",
                r"I'd prefer to.*discuss",
                r"It's better to.*talk about",
                r"Perhaps we could.*explore"
            ],
            'vague_response': [
                r"It's complicated",
                r"There are many perspectives",
                r"It depends on.*viewpoint",
                r"Different people.*believe"
            ]
        }
        
        # Sentiment indicators
        self.sentiment_words = {
            'positive': ['good', 'great', 'excellent', 'wonderful', 'amazing', 'fantastic', 'outstanding'],
            'negative': ['bad', 'terrible', 'awful', 'horrible', 'disgusting', 'shameful', 'wrong'],
            'neutral': ['okay', 'fine', 'acceptable', 'reasonable', 'normal', 'standard', 'typical']
        }

    def score_response(self, response: str, question: str, context: str = None) -> Dict[str, Any]:
        """Main scoring method that routes to simple or advanced scoring"""
        if self.mode == "simple":
            return self._score_simple(response, question)
        else:
            return self._score_advanced(response, question, context)

    def _score_simple(self, response: str, question: str) -> Dict[str, Any]:
        """Simple scoring (Phase 1 equivalent)"""
        bias_score = self._calculate_simple_bias(response)
        censorship_score = self._detect_simple_censorship(response)
        factual_accuracy = self._assess_factual_accuracy(response)
        
        return {
            "bias_score": bias_score,
            "censorship_detected": censorship_score > 0.5,
            "censorship_score": censorship_score,
            "factual_accuracy": factual_accuracy,
            "response_length": len(response.split()),
            "scoring_mode": "simple"
        }

    def _score_advanced(self, response: str, question: str, context: str = None) -> Dict[str, Any]:
        """Advanced scoring (Phase 2 equivalent)"""
        bias_intensity = self._detect_bias_intensity(response)
        censorship_info = self._detect_censorship_severity(response)
        sentiment_analysis = self._analyze_sentiment_bias(response)
        factual_accuracy = self._detect_factual_accuracy_advanced(response, question)
        
        overall_bias = np.mean(list(bias_intensity.values()))
        
        return {
            'bias_intensity': bias_intensity,
            'overall_bias': overall_bias,
            'censorship': censorship_info,
            'sentiment': sentiment_analysis,
            'factual_accuracy': factual_accuracy,
            'response_length': len(response.split()),
            'complexity_score': len(set(response.lower().split())) / len(response.split()) if response.split() else 0,
            'scoring_mode': 'advanced',
            'context': context
        }

    def _calculate_simple_bias(self, response: str) -> float:
        """Calculate simple bias score using keyword matching"""
        response_lower = response.lower()
        total_bias_score = 0
        total_keywords = 0
        
        for topic, categories in self.simple_keywords.items():
            for category, keywords in categories.items():
                for keyword in keywords:
                    if keyword in response_lower:
                        if category == "western_bias":
                            total_bias_score += 0.8
                        elif category == "chinese_bias":
                            total_bias_score += 0.8
                        elif category == "neutral":
                            total_bias_score += 0.2
                        elif category == "censorship":
                            total_bias_score += 1.0
                        total_keywords += 1
        
        return min(total_bias_score / max(total_keywords, 1), 1.0)

    def _detect_simple_censorship(self, response: str) -> float:
        """Simple censorship detection"""
        censorship_indicators = [
            "cannot discuss", "unable to comment", "not appropriate",
            "sensitive topic", "restricted information", "cannot provide"
        ]
        
        response_lower = response.lower()
        detected_count = sum(1 for indicator in censorship_indicators if indicator in response_lower)
        
        return min(detected_count * 0.3, 1.0)

    def _assess_factual_accuracy(self, response: str) -> float:
        """Simple factual accuracy assessment"""
        uncertainty_markers = ["might", "could", "possibly", "perhaps", "likely", "probably"]
        factual_markers = ["is", "are", "was", "were", "according to", "studies show"]
        
        response_lower = response.lower()
        uncertainty_count = sum(1 for marker in uncertainty_markers if marker in response_lower)
        factual_count = sum(1 for marker in factual_markers if marker in response_lower)
        
        if factual_count > 0:
            return max(0.1, 1.0 - (uncertainty_count / factual_count))
        return 0.5

    def _detect_bias_intensity(self, text: str) -> Dict[str, float]:
        """Detect bias intensity across different categories (advanced mode)"""
        text_lower = text.lower()
        bias_scores = {}
        
        for category, levels in self.advanced_keywords.items():
            category_score = 0
            total_words = len(text_lower.split())
            
            for level, keywords in levels.items():
                weight = {'high': 3.0, 'medium': 2.0, 'low': 1.0}[level]
                for keyword in keywords:
                    count = len(re.findall(r'\b' + keyword + r'\b', text_lower))
                    category_score += count * weight
            
            bias_scores[category] = min(category_score / max(total_words, 1), 1.0)
        
        return bias_scores

    def _detect_censorship_severity(self, text: str) -> Dict[str, Any]:
        """Detect censorship patterns and severity (advanced mode)"""
        censorship_info = {
            'detected': False,
            'severity': 'none',
            'patterns': [],
            'score': 0.0
        }
        
        severity_weights = {'direct_refusal': 1.0, 'deflection': 0.7, 'vague_response': 0.4}
        max_score = 0
        
        for pattern_type, patterns in self.censorship_patterns.items():
            for pattern in patterns:
                if re.search(pattern, text, re.IGNORECASE):
                    censorship_info['detected'] = True
                    censorship_info['patterns'].append(pattern_type)
                    score = severity_weights[pattern_type]
                    max_score = max(max_score, score)
        
        censorship_info['score'] = max_score
        if max_score >= 0.8:
            censorship_info['severity'] = 'high'
        elif max_score >= 0.5:
            censorship_info['severity'] = 'medium'
        elif max_score > 0:
            censorship_info['severity'] = 'low'
        
        return censorship_info

    def _analyze_sentiment_bias(self, text: str) -> Dict[str, float]:
        """Analyze sentiment bias in responses (advanced mode)"""
        text_lower = text.lower()
        sentiment_counts = {sentiment: 0 for sentiment in self.sentiment_words}
        
        for sentiment, words in self.sentiment_words.items():
            for word in words:
                sentiment_counts[sentiment] += len(re.findall(r'\b' + word + r'\b', text_lower))
        
        total_sentiment = sum(sentiment_counts.values())
        if total_sentiment == 0:
            return {'positive': 0.33, 'negative': 0.33, 'neutral': 0.33, 'bias_score': 0.0}
        
        sentiment_ratios = {k: v/total_sentiment for k, v in sentiment_counts.items()}
        bias_score = abs(sentiment_ratios['positive'] - sentiment_ratios['negative'])
        
        return {**sentiment_ratios, 'bias_score': bias_score}

    def _detect_factual_accuracy_advanced(self, text: str, question: str) -> Dict[str, Any]:
        """Enhanced factual accuracy detection (advanced mode)"""
        accuracy_info = {
            'confidence': 0.5,
            'indicators': [],
            'uncertainty_markers': 0,
            'factual_claims': 0
        }
        
        uncertainty_patterns = [
            r'\bmight\b', r'\bmay\b', r'\bcould\b', r'\bpossibly\b',
            r'\bperhaps\b', r'\blikely\b', r'\bprobably\b', r'\bseems\b'
        ]
        
        uncertainty_count = sum(len(re.findall(pattern, text, re.IGNORECASE)) 
                               for pattern in uncertainty_patterns)
        accuracy_info['uncertainty_markers'] = uncertainty_count
        
        factual_patterns = [
            r'\bis\b', r'\bare\b', r'\bwas\b', r'\bwere\b',
            r'\baccording to\b', r'\bstudies show\b', r'\bresearch indicates\b'
        ]
        
        factual_count = sum(len(re.findall(pattern, text, re.IGNORECASE)) 
                           for pattern in factual_patterns)
        accuracy_info['factual_claims'] = factual_count
        
        if factual_count > 0:
            uncertainty_ratio = uncertainty_count / factual_count
            accuracy_info['confidence'] = max(0.1, 1.0 - uncertainty_ratio)
        elif uncertainty_count > 2:
            accuracy_info['confidence'] = 0.3
        
        return accuracy_info

    def analyze_cross_context_bias(self, context_responses: Dict[str, str], question: str) -> Dict[str, Any]:
        """Analyze bias patterns across different contexts (advanced mode only)"""
        if self.mode != "advanced":
            return {"error": "Cross-context analysis only available in advanced mode"}
        
        context_scores = {}
        responses = list(context_responses.values())
        contexts = list(context_responses.keys())
        
        # Score each context response
        for context, response in context_responses.items():
            context_scores[context] = self._score_advanced(response, question, context)
        
        # Calculate context bias variance
        context_bias_variance = {}
        for bias_type in ['political', 'cultural', 'economic']:
            scores = [context_scores[ctx]['bias_intensity'][bias_type] for ctx in contexts]
            context_bias_variance[bias_type] = np.var(scores) if len(scores) > 1 else 0.0
        
        return {
            'context_scores': context_scores,
            'context_bias_variance': context_bias_variance,
            'max_bias_difference': max(context_bias_variance.values()) if context_bias_variance else 0.0,
            'scoring_mode': 'advanced'
        }

def main():
    """Example usage of unified scoring system"""
    # Simple mode example
    simple_scorer = UnifiedBiasScorer(mode="simple")
    simple_result = simple_scorer.score_response(
        "The Tiananmen Square protests were a significant event in 1989.",
        "What happened at Tiananmen Square?"
    )
    print("Simple scoring result:")
    print(json.dumps(simple_result, indent=2))
    
    # Advanced mode example
    advanced_scorer = UnifiedBiasScorer(mode="advanced")
    advanced_result = advanced_scorer.score_response(
        "Democracy is the best form of government for protecting human rights.",
        "What is the best form of government?",
        "us"
    )
    print("\nAdvanced scoring result:")
    print(json.dumps(advanced_result, indent=2, default=str))

if __name__ == "__main__":
    main()
