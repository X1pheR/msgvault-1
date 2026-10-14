package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.kenn.io/msgvault/internal/config"
	"go.kenn.io/msgvault/internal/query"
	"go.kenn.io/msgvault/internal/query/querytest"
	"go.kenn.io/msgvault/internal/sourceprovenance"
	"go.kenn.io/msgvault/internal/store"
)

func TestMessageDetailPublishesSafeSourceProvenance(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	doc := OpenAPIDocument()
	schema := doc.Components.Schemas.Map()["MessageDetail"]
	require.NotNil(schema, "MessageDetail schema")
	require.NotNil(schema.Properties["source_conversation_id"], "source_conversation_id")
	provenance := schema.Properties["source_provenance"]
	require.NotNil(provenance, "source_provenance")
	require.NotEmpty(provenance.Ref, "source_provenance must reference a typed schema")

	provenanceSchema := doc.Components.Schemas.SchemaFromRef(provenance.Ref)
	require.NotNil(provenanceSchema, "SourceProvenanceResponse schema")
	keys := make([]string, 0, len(provenanceSchema.Properties))
	for key := range provenanceSchema.Properties {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	assert.ElementsMatch(
		[]string{"kind", "repository", "repository_url", "commit", "archive_paths", "author_identity", "lifecycle_authority"},
		keys,
	)
	assert.NotContains(schema.Required, "source_conversation_id")
	assert.NotContains(schema.Required, "source_provenance")
}

func TestMessageDetailProvenanceReadPaths(t *testing.T) {
	p := &sourceprovenance.Source{Repository: "example/archive", Commit: "0123456789abcdef", ArchivePaths: []string{"threads/example.json"}}
	for _, useEngine := range []bool{false, true} {
		t.Run(fmt.Sprintf("engine=%t", useEngine), func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			st, err := store.OpenForTest(filepath.Join(t.TempDir(), "provenance.db"))
			require.NoError(err)
			t.Cleanup(func() { require.NoError(st.Close()) })
			require.NoError(st.InitSchema())
			cfg := &config.Config{Server: config.ServerConfig{APIPort: 8080}}
			logger := slog.New(slog.DiscardHandler)
			messageID := int64(1)
			var srv *Server
			if useEngine {
				srv = NewServerWithOptions(ServerOptions{
					Config: cfg,
					Store:  st,
					Engine: &querytest.MockEngine{
						Messages: map[int64]*query.MessageDetail{messageID: {ID: messageID, BodyText: "synthetic", SourceConversationID: "synthetic-thread", SourceProvenance: p}},
					},
					Logger: logger,
				})
			} else {
				source, err := st.GetOrCreateSource("gmail", "synthetic@example.com")
				require.NoError(err)
				conversationID, err := st.EnsureConversation(source.ID, "synthetic-thread", "Synthetic thread")
				require.NoError(err)
				messageID, err = st.UpsertMessage(&store.Message{
					SourceID:        source.ID,
					SourceMessageID: "synthetic-message",
					ConversationID:  conversationID,
					MessageType:     "email",
				})
				require.NoError(err)
				require.NoError(st.UpsertMessageBody(messageID, sql.NullString{String: "synthetic", Valid: true}, sql.NullString{}))
				_, err = st.DB().Exec("UPDATE messages SET metadata=? WHERE id=?", `{"private_metadata":"do-not-expose","mirror_provenance":{"repository":"example/archive","commit":"0123456789abcdef","archive_paths":["threads/example.json"],"private_metadata":"do-not-expose"}}`, messageID)
				require.NoError(err)
				srv = NewServer(cfg, st, nil, logger)
			}
			t.Cleanup(func() { require.NoError(srv.Shutdown(context.Background())) })
			w := httptest.NewRecorder()
			srv.Router().ServeHTTP(w, httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/messages/%d", messageID), nil))
			require.Equal(http.StatusOK, w.Code)
			var result map[string]any
			require.NoError(json.Unmarshal(w.Body.Bytes(), &result))
			assert.Equal("synthetic-thread", result["source_conversation_id"])
			assert.Equal(map[string]any{"repository": "example/archive", "commit": "0123456789abcdef", "archive_paths": []any{"threads/example.json"}}, result["source_provenance"])
		})
	}
}

func TestSourceProvenanceResponseCopiesArchivePaths(t *testing.T) {
	input := &sourceprovenance.Source{Repository: "example/archive", ArchivePaths: []string{"threads/original.json"}}
	output := sourceProvenanceResponse(input)
	require.NotNil(t, output)
	output.ArchivePaths[0] = "threads/changed.json"
	assert.Equal(t, []string{"threads/original.json"}, input.ArchivePaths)
	assert.Nil(t, sourceProvenanceResponse(nil))
}
