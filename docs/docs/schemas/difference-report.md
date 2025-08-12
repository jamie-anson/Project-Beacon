---
id: difference-report
title: DifferenceReport Schema
---

Schema and identical/differing examples.

import Tabs from '@theme/Tabs';
import TabItem from '@theme/TabItem';

<Tabs>
  <TabItem value="schema" label="Schema" default>

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://projectbeacon.example/schemas/diff.json",
  "title": "DifferenceReport",
  "type": "object",
  "properties": {
    "jobspec_id": {"type": "string"},
    "comparisons": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "region_a": {"type": "string"},
          "region_b": {"type": "string"},
          "equal": {"type": "boolean"},
          "diff": {"type": "string"}
        },
        "required": ["region_a", "region_b", "equal"]
      }
    }
  },
  "required": ["jobspec_id", "comparisons"],
  "additionalProperties": true
}
```

  </TabItem>
  <TabItem value="identical" label="Identical Example">

```json
{
  "jobspec_id": "who-are-you-v1",
  "comparisons": [
    {"region_a": "us-east-1", "region_b": "eu-west-2", "equal": true, "diff": ""}
  ]
}
```

  </TabItem>
  <TabItem value="differing" label="Differing Example">

```json
{
  "jobspec_id": "who-are-you-v1",
  "comparisons": [
    {"region_a": "us-east-1", "region_b": "eu-west-2", "equal": false, "diff": "- I am X\n+ I am Y"}
  ]
}
```

  </TabItem>
</Tabs>
