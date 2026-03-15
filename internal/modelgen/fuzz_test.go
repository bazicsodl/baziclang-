package modelgen

import (
	"encoding/json"
	"testing"
)

func FuzzSchemaJSON(f *testing.F) {
	f.Add([]byte(`{"version":1,"database":{"driver":"sqlite","dsn":"app.db"},"models":[]}`))
	f.Add([]byte(`{"models":[{"name":"User","table":"users","fields":[{"name":"id","type":"int","pk":true}]}]}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		var s Schema
		if err := json.Unmarshal(data, &s); err != nil {
			return
		}
		normalizeSchema(&s)
		_ = ValidateSchema(s)
	})
}
