package crypto

import (
    "encoding/json"
    "fmt"
)

// StripKeysFromJSON parses a JSON object and removes the specified top-level keys.
// It returns a generic map[string]interface{} suitable for canonicalization.
func StripKeysFromJSON(raw []byte, removeKeys []string) (interface{}, error) {
    if len(raw) == 0 {
        return nil, fmt.Errorf("empty input")
    }
    var obj map[string]interface{}
    if err := json.Unmarshal(raw, &obj); err != nil {
        return nil, fmt.Errorf("invalid json: %w", err)
    }
    for _, k := range removeKeys {
        delete(obj, k)
    }
    return obj, nil
}
