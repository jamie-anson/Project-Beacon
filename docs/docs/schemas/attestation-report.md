---
id: attestation-report
title: AttestationReport Schema
---

Schema and example attester confirmation.

import Tabs from '@theme/Tabs';
import TabItem from '@theme/TabItem';

<Tabs>
  <TabItem value="schema" label="Schema" default>

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "https://projectbeacon.example/schemas/attestation.json",
  "title": "AttestationReport",
  "type": "object",
  "properties": {
    "jobspec_id": {"type": "string"},
    "diff_cid": {"type": "string"},
    "decision": {"type": "string", "enum": ["confirm", "reject"]},
    "notes": {"type": "string"},
    "attester": {"type": "string"},
    "signature": {"type": "string"}
  },
  "required": ["jobspec_id", "diff_cid", "decision", "attester", "signature"],
  "additionalProperties": true
}
```

  </TabItem>
  <TabItem value="example" label="Example">

```json
{
  "jobspec_id": "who-are-you-v1",
  "diff_cid": "bafy...diffcid",
  "decision": "confirm",
  "notes": "Outputs match claim across regions.",
  "attester": "attester-123",
  "signature": "base64:..."
}
```

  </TabItem>
</Tabs>
