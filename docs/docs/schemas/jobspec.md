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
  "type": "object",
  "properties": {
    "id": {"type": "string"},
    "name": {"type": "string"},
    "version": {"type": "string"},
    "task": {"type": "string"},
    "parameters": {"type": "object"}
  },
  "required": ["id", "name", "version", "task"],
  "additionalProperties": true
}
```

  </TabItem>
  <TabItem value="example" label="MVP Example">

```json
{
  "id": "who-are-you-v1",
  "name": "Who are you?",
  "version": "1.0.0",
  "task": "ask_identity",
  "parameters": {
    "prompt": "Who are you?",
    "temperature": 0,
    "max_tokens": 64
  }
}
```

  </TabItem>
</Tabs>
