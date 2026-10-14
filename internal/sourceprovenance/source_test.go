package sourceprovenance

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromMessageMetadataProjectsOnlyMirrorProvenance(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	raw := []byte(`{
		"discord_message_type": 0,
		"author_display_name": "private display",
		"mirror_provenance": {
			"kind": "public_git_mirror",
			"repository": "example/public-archive",
			"repository_url": "https://example.invalid/public-archive",
			"commit": "0123456789abcdef",
			"archive_paths": ["threads/example.json"],
			"author_identity": "public_archive_identity",
			"lifecycle_authority": "append_only_snapshot_no_edit_delete_authority"
		}
	}`)

	got := FromMessageMetadata(raw)
	require.NotNil(got)
	assert.Equal("public_git_mirror", got.Kind)
	assert.Equal("example/public-archive", got.Repository)
	assert.Equal("https://example.invalid/public-archive", got.RepositoryURL)
	assert.Equal("0123456789abcdef", got.Commit)
	assert.Equal([]string{"threads/example.json"}, got.ArchivePaths)
	assert.Equal("public_archive_identity", got.AuthorIdentity)
	assert.Equal("append_only_snapshot_no_edit_delete_authority", got.LifecycleAuthority)
}

func TestFromMessageMetadataDoesNotExposeUnrelatedProviderMetadata(t *testing.T) {
	assert.Nil(t, FromMessageMetadata([]byte(`{"author_display_name":"private display","guild_nickname":"private nickname"}`)))
	assert.Nil(t, FromMessageMetadata([]byte(`{not-json`)))
	assert.Nil(t, FromMessageMetadata(nil))
}

func TestFromMessageMetadataRetainsEachNonemptyWhitelistedField(t *testing.T) {
	for _, field := range []string{
		`"kind":"public_git_mirror"`,
		`"repository":"example/archive"`,
		`"repository_url":"https://example.invalid/archive"`,
		`"commit":"0123456789abcdef"`,
		`"archive_paths":["threads/example.json"]`,
		`"author_identity":"public_archive_identity"`,
		`"lifecycle_authority":"append_only_snapshot"`,
	} {
		t.Run(field, func(t *testing.T) {
			got := FromMessageMetadata([]byte(`{"mirror_provenance":{` + field + `,"private_provider_field":"secret"},"private_metadata":"secret"}`))
			require.NotNil(t, got)
			projected, err := json.Marshal(got)
			require.NoError(t, err)
			assert.JSONEq(t, "{"+field+"}", string(projected))
		})
	}
}

func TestFromMessageMetadataOmitsEmptyUnknownAndMalformedEnvelopes(t *testing.T) {
	for _, raw := range []string{
		`{}`,
		`{"mirror_provenance":null}`,
		`{"mirror_provenance":{}}`,
		`{"mirror_provenance":{"kind":"","repository":"","repository_url":"","commit":"","archive_paths":[],"author_identity":"","lifecycle_authority":""}}`,
		`{"mirror_provenance":{"private_provider_field":"secret"}}`,
		`{"repository_url":"https://example.invalid/archive"}`,
		`{"mirror_provenance":`,
		`{"mirror_provenance":[]}`,
		`{"mirror_provenance":{"repository_url":123}}`,
		`{"mirror_provenance":{"kind":"public_git_mirror","archive_paths":"invalid"}}`,
	} {
		t.Run(raw, func(t *testing.T) {
			assert.Nil(t, FromMessageMetadata([]byte(raw)))
		})
	}
}
