package query

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGetMessageProjectsMirrorProvenance(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	env := newTestEnv(t)
	_, err := env.DB.Exec(`UPDATE messages SET metadata=? WHERE id=1`, `{"provider_private":"do-not-expose","mirror_provenance":{"repository":"example/archive","commit":"0123456789abcdef","unknown":"do-not-expose"}}`)
	require.NoError(err)
	msg, err := env.Engine.GetMessage(env.Ctx, 1)
	require.NoError(err)
	require.NotNil(msg)
	require.NotNil(msg.SourceProvenance)
	assert.Equal("example/archive", msg.SourceProvenance.Repository)
	encoded, err := json.Marshal(msg.SourceProvenance)
	require.NoError(err)
	assert.NotContains(string(encoded), "do-not-expose")
	bySource, err := env.Engine.GetMessageBySourceID(env.Ctx, msg.SourceMessageID, []int64{msg.SourceID})
	require.NoError(err)
	require.NotNil(bySource)
	assert.Equal(msg.SourceProvenance, bySource.SourceProvenance)
	absent, err := env.Engine.GetMessageBySourceID(env.Ctx, msg.SourceMessageID, []int64{msg.SourceID + 1000})
	require.NoError(err)
	assert.Nil(absent)
}
