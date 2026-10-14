package store

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGetMessageProjectsExplicitMirrorProvenance(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	st := openTestStore(t)
	source, err := st.GetOrCreateSource("gmail", "synthetic@example.com")
	require.NoError(err)
	convID, err := st.EnsureConversation(source.ID, "synthetic-thread", "Synthetic thread")
	require.NoError(err)
	id := seedMessage(t, st, source.ID, convID, "synthetic-message", "subject", "snippet")
	_, err = st.DB().Exec("UPDATE messages SET metadata=? WHERE id=?", `{"provider_private":"do-not-expose","mirror_provenance":{"kind":"public_git_mirror","repository":"example/archive","commit":"0123456789abcdef","private_extra":"do-not-expose"}}`, id)
	require.NoError(err)
	msg, err := st.GetMessageContext(t.Context(), id)
	require.NoError(err)
	require.NotNil(msg.SourceProvenance)
	assert.Equal("example/archive", msg.SourceProvenance.Repository)
	encoded, err := json.Marshal(msg.SourceProvenance)
	require.NoError(err)
	assert.NotContains(string(encoded), "do-not-expose")
	assert.NotContains(string(encoded), "private_extra")
}
