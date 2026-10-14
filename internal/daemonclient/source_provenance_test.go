package daemonclient

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.kenn.io/msgvault/pkg/client/generated"
	"testing"
)

func TestGeneratedDetailPreservesSourceProvenance(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)
	assert.Nil(generatedDetailToAPIMessage(nil))
	assert.Nil(generatedDetailToAPIMessage(&generated.MessageDetail{}).SourceProvenance)
	kind, repository, url, commit := "git", "example/repository", "https://example.com/repository", "abc123"
	author, authority, conversation := "synthetic@example.com", "archive-owner", "conversation-a"
	source := &generated.SourceProvenanceResponse{
		Kind: &kind, Repository: &repository, RepositoryURL: &url, Commit: &commit,
		ArchivePaths: []string{"archive/message.json"}, AuthorIdentity: &author, LifecycleAuthority: &authority,
	}
	msg := generatedDetailToAPIMessage(&generated.MessageDetail{SourceConversationID: &conversation, SourceProvenance: source})
	require.NotNil(msg.SourceProvenance)
	assert.Equal(conversation, msg.SourceConversationID)
	assert.Equal(kind, msg.SourceProvenance.Kind)
	assert.Equal(repository, msg.SourceProvenance.Repository)
	assert.Equal(url, msg.SourceProvenance.RepositoryURL)
	assert.Equal(commit, msg.SourceProvenance.Commit)
	assert.Equal(author, msg.SourceProvenance.AuthorIdentity)
	assert.Equal(authority, msg.SourceProvenance.LifecycleAuthority)
	assert.Equal(source.ArchivePaths, msg.SourceProvenance.ArchivePaths)
	source.ArchivePaths[0] = "changed"
	assert.Equal([]string{"archive/message.json"}, msg.SourceProvenance.ArchivePaths)
}
