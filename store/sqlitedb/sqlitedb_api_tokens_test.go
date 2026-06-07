package sqlitedb

import (
	"testing"
	"time"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/DigitalTolk/wireguard-ui/model"
	"github.com/DigitalTolk/wireguard-ui/store"
	"github.com/DigitalTolk/wireguard-ui/util"
)

func newAPITokenForTest(t *testing.T, name string) (model.APIToken, string) {
	t.Helper()
	plain, err := util.GenerateAPIToken()
	require.NoError(t, err)
	return model.APIToken{
		ID:        xid.New().String(),
		Name:      name,
		CreatedBy: "admin",
		CreatedAt: time.Now().UTC(),
	}, plain
}

func TestCreateAndLookupAPIToken(t *testing.T) {
	db := initTestDB(t)
	tok, plain := newAPITokenForTest(t, "ci-runner")
	hash := util.HashAPIToken(plain)

	require.NoError(t, db.CreateAPIToken(tok, hash))

	got, err := db.GetAPITokenByHash(hash)
	require.NoError(t, err)
	assert.Equal(t, tok.ID, got.ID)
	assert.Equal(t, "ci-runner", got.Name)
	assert.Equal(t, "admin", got.CreatedBy)
	assert.Nil(t, got.LastUsedAt, "fresh token has no last_used_at")
	assert.Nil(t, got.RevokedAt)
}

func TestGetAPITokenByHash_NotFound(t *testing.T) {
	db := initTestDB(t)
	_, err := db.GetAPITokenByHash(util.HashAPIToken("wgui_nonexistent_lookalike_padding_padding_padding_padding"))
	assert.ErrorIs(t, err, store.ErrAPITokenNotFound)
}

func TestCreateAPIToken_DuplicateHashRejected(t *testing.T) {
	db := initTestDB(t)
	tok, plain := newAPITokenForTest(t, "first")
	hash := util.HashAPIToken(plain)
	require.NoError(t, db.CreateAPIToken(tok, hash))

	// New ID, same hash — must collide on UNIQUE(token_hash).
	dup := tok
	dup.ID = xid.New().String()
	dup.Name = "duplicate"
	err := db.CreateAPIToken(dup, hash)
	assert.Error(t, err, "two tokens with the same hash must not coexist")
}

func TestListAPITokens_NewestFirst_IncludesRevoked(t *testing.T) {
	db := initTestDB(t)

	older, plain1 := newAPITokenForTest(t, "older")
	older.CreatedAt = time.Now().Add(-1 * time.Hour).UTC()
	require.NoError(t, db.CreateAPIToken(older, util.HashAPIToken(plain1)))

	newer, plain2 := newAPITokenForTest(t, "newer")
	newer.CreatedAt = time.Now().UTC()
	require.NoError(t, db.CreateAPIToken(newer, util.HashAPIToken(plain2)))

	require.NoError(t, db.RevokeAPIToken(older.ID))

	list, err := db.ListAPITokens()
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, "newer", list[0].Name, "newest first")
	assert.Equal(t, "older", list[1].Name)
	assert.NotNil(t, list[1].RevokedAt, "revoked tokens must still appear in the list")
}

func TestRevokeAPIToken_IsIdempotent(t *testing.T) {
	db := initTestDB(t)
	tok, plain := newAPITokenForTest(t, "revoke-me")
	require.NoError(t, db.CreateAPIToken(tok, util.HashAPIToken(plain)))

	require.NoError(t, db.RevokeAPIToken(tok.ID))
	got1, err := db.GetAPITokenByHash(util.HashAPIToken(plain))
	require.NoError(t, err)
	require.NotNil(t, got1.RevokedAt)
	first := *got1.RevokedAt

	// Second revoke must not move the timestamp — once burned, the original
	// revocation moment is the audit-relevant one.
	require.NoError(t, db.RevokeAPIToken(tok.ID))
	got2, err := db.GetAPITokenByHash(util.HashAPIToken(plain))
	require.NoError(t, err)
	require.NotNil(t, got2.RevokedAt)
	assert.True(t, first.Equal(*got2.RevokedAt), "revoked_at must not be overwritten by subsequent revoke calls")
}

func TestRevokeAPIToken_UnknownID(t *testing.T) {
	db := initTestDB(t)
	err := db.RevokeAPIToken(xid.New().String())
	assert.ErrorIs(t, err, store.ErrAPITokenNotFound)
}

func TestTouchAPITokenLastUsed(t *testing.T) {
	db := initTestDB(t)
	tok, plain := newAPITokenForTest(t, "touchable")
	require.NoError(t, db.CreateAPIToken(tok, util.HashAPIToken(plain)))

	when := time.Now().Add(-5 * time.Minute).UTC().Truncate(time.Second)
	require.NoError(t, db.TouchAPITokenLastUsed(tok.ID, when))

	got, err := db.GetAPITokenByHash(util.HashAPIToken(plain))
	require.NoError(t, err)
	require.NotNil(t, got.LastUsedAt)
	assert.True(t, when.Equal(got.LastUsedAt.UTC()), "got %v want %v", got.LastUsedAt, when)
}
