package mcp

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.kenn.io/msgvault/internal/query"
	"go.kenn.io/msgvault/internal/query/querytest"
	"go.kenn.io/msgvault/internal/sourceprovenance"
)

func TestGetMessageIncludesOnlyExplicitProvenance(t *testing.T) {
	for _, p := range []*sourceprovenance.Source{nil, {Repository: "example/archive", Commit: "0123456789abcdef"}} {
		name := "without-provenance"
		if p != nil {
			name = "with-provenance"
		}
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			h := newTestHandlers(&querytest.MockEngine{Messages: map[int64]*query.MessageDetail{42: {ID: 42, SourceID: 7, SourceProvenance: p}}})
			response := callToolDirect(t, "get_message", h.getMessage, map[string]any{"id": float64(42)})
			require.False(response.isError, "unexpected error: %s", resultText(t, response))
			decoder := json.NewDecoder(strings.NewReader(resultText(t, response)))
			decoder.UseNumber()
			var result map[string]any
			require.NoError(decoder.Decode(&result), "decode failed")
			if p == nil {
				assert.NotContains(result, "source_provenance")
			} else {
				assert.Equal(map[string]any{"repository": "example/archive", "commit": "0123456789abcdef"}, result["source_provenance"])
				assert.Equal(json.Number("7"), result["source_id"])
			}
		})
	}
}
