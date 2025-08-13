---
id: jobspec
title: JobSpec Schema
---

Below is the JSON Schema for `JobSpec` and an MVP example.

import Tabs from '@theme/Tabs';
import TabItem from '@theme/TabItem';

<Tabs>
  <TabItem value="schema" label="Schema" default>

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://projectbeacon.example/schemas/jobspec.json",
  "title": "JobSpec",
  "description": "Specification for a benchmarking job to be executed by Project Beacon",
  "type": "object",
  "properties": {
    "id": {
      "type": "string",
      "description": "Unique identifier for this job specification",
      "pattern": "^[a-z0-9-]+$",
      "minLength": 3,
      "maxLength": 100
    },
    "name": {
      "type": "string",
      "description": "Human-readable name for the job",
      "minLength": 3,
      "maxLength": 200
    },
    "version": {
      "type": "string",
      "description": "Version of the job specification",
      "pattern": "^\\d+\\.\\d+\\.\\d+$"
    },
    "task": {
      "type": "string",
      "description": "Type of task to be performed",
      "enum": ["text_generation", "question_answering", "summarization", "translation"]
    },
    "parameters": {
      "type": "object",
      "description": "Task-specific parameters",
      "properties": {
        "prompt": {
          "type": "string",
          "description": "Input prompt for the task",
          "minLength": 1
        },
        "max_tokens": {
          "type": "integer",
          "description": "Maximum number of tokens to generate",
          "minimum": 1,
          "maximum": 4000,
          "default": 100
        },
        "temperature": {
          "type": "number",
          "description": "Sampling temperature (0-2)",
          "minimum": 0,
          "maximum": 2,
          "default": 0.7
        },
        "top_p": {
          "type": "number",
          "description": "Nucleus sampling parameter",
          "minimum": 0,
          "maximum": 1,
          "default": 1.0
        }
      },
      "required": ["prompt"],
      "additionalProperties": false
    },
    "metadata": {
      "type": "object",
      "description": "Additional metadata about the job",
      "properties": {
        "author": {
          "type": "string",
          "description": "Name or identifier of the job author"
        },
        "created_at": {
          "type": "string",
          "description": "ISO 8601 timestamp of job creation",
          "format": "date-time"
        },
        "tags": {
          "type": "array",
          "description": "Tags for categorizing the job",
          "items": {
            "type": "string"
          },
          "uniqueItems": true
        }
      }
    }
  },
  "required": ["id", "name", "version", "task", "parameters"],
  "additionalProperties": false
}
```

  </TabItem>
  <TabItem value="example" label="MVP Example">

```json
{
  "id": "summarization-benchmark-v1",
  "name": "Document Summarization Benchmark",
  "version": "1.0.0",
  "task": "summarization",
  "parameters": {
    "prompt": "Summarize the following document in 3-5 sentences: The quick brown fox jumps over the lazy dog. This is a test document that needs to be summarized for our benchmark.",
    "temperature": 0.7,
    "max_tokens": 256,
    "top_p": 0.9
  },
  "metadata": {
    "author": "benchmark-team@example.com",
    "created_at": "2025-01-15T09:30:00Z",
    "tags": ["summarization", "benchmark", "v1.0"]
  }
}
```

  </TabItem>
</Tabs>
