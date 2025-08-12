---
id: receipt
title: Receipt Schema
---

Receipt JSON Schema and example from MVP.

import Tabs from '@theme/Tabs';
import TabItem from '@theme/TabItem';

<Tabs>
  <TabItem value="schema" label="Schema" default>

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://projectbeacon.example/schemas/receipt.json",
  "title": "Receipt",
  "type": "object",
  "properties": {
    "jobspec_id": {"type": "string"},
    "runner": {"type": "string"},
    "timestamp": {"type": "string", "format": "date-time"},
    "signature": {"type": "string"},
    "output": {"type": "object"}
  },
  "required": ["jobspec_id", "runner", "timestamp", "signature"],
  "additionalProperties": true
}
```

  </TabItem>
  <TabItem value="example" label="MVP Example">

```json
{
  "jobspec_id": "who-are-you-v1",
  "runner": "golem-node-eu-west-2",
  "timestamp": "2025-08-12T12:00:00Z",
  "signature": "base64:...",
  "output": {
    "text": "I am an AI model."
  }
}
```

  </TabItem>
</Tabs>
