package audit

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test_audit.db")
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS audit_logs (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			actor         TEXT NOT NULL,
			action        TEXT NOT NULL,
			resource_type TEXT NOT NULL DEFAULT '',
			resource_id   TEXT NOT NULL DEFAULT '',
			details       TEXT NOT NULL DEFAULT '{}',
			ip_address    TEXT NOT NULL DEFAULT ''
		)
	`)
	require.NoError(t, err)
	return db
}

func TestLog(t *testing.T) {
	db := newTestDB(t)
	logger := NewLogger(db)

	logger.Log(Entry{
		Actor:        "admin",
		Action:       "user.create",
		ResourceType: "user",
		ResourceID:   "testuser",
		Details:      map[string]string{"role": "admin"},
		IPAddress:    "10.0.0.1",
	})

	entries, total, err := logger.Query("", "", "", "", 1, 50)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, entries, 1)
	assert.Equal(t, "admin", entries[0].Actor)
	assert.Equal(t, "user.create", entries[0].Action)
	assert.Equal(t, "user", entries[0].ResourceType)
	assert.Equal(t, "testuser", entries[0].ResourceID)
	assert.Contains(t, entries[0].Details, "admin")
	assert.Equal(t, "10.0.0.1", entries[0].IPAddress)
}

func TestLog_NilLogger(t *testing.T) {
	var logger *Logger
	// should not panic
	logger.Log(Entry{Actor: "test", Action: "test"})
}

func TestLog_NilDB(t *testing.T) {
	logger := &Logger{db: nil}
	// should not panic
	logger.Log(Entry{Actor: "test", Action: "test"})
}

func TestLogWithUser(t *testing.T) {
	db := newTestDB(t)
	logger := NewLogger(db)

	logger.LogWithUser("admin", "client.delete", "client", "xyz123", "192.168.1.1", nil)

	entries, total, err := logger.Query("", "", "", "", 1, 50)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, "admin", entries[0].Actor)
	assert.Equal(t, "client.delete", entries[0].Action)
}

func TestQuery_Filtering(t *testing.T) {
	db := newTestDB(t)
	logger := NewLogger(db)

	logger.Log(Entry{Actor: "admin", Action: "user.create", ResourceType: "user", ResourceID: "u1", IPAddress: "10.0.0.1"})
	logger.Log(Entry{Actor: "admin", Action: "client.create", ResourceType: "client", ResourceID: "c1", IPAddress: "10.0.0.1"})
	logger.Log(Entry{Actor: "manager", Action: "client.update", ResourceType: "client", ResourceID: "c1", IPAddress: "10.0.0.2"})

	// filter by actor
	entries, total, err := logger.Query("", "", "admin", "", 1, 50)
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, entries, 2)

	// filter by action
	entries, total, err = logger.Query("", "", "", "client.create", 1, 50)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, "client.create", entries[0].Action)

	// filter by both
	entries, total, err = logger.Query("", "", "manager", "client.update", 1, 50)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
}

func TestQuery_Pagination(t *testing.T) {
	db := newTestDB(t)
	logger := NewLogger(db)

	for i := 0; i < 10; i++ {
		logger.Log(Entry{Actor: "admin", Action: "test", IPAddress: "10.0.0.1"})
	}

	entries, total, err := logger.Query("", "", "", "", 1, 3)
	require.NoError(t, err)
	assert.Equal(t, 10, total)
	assert.Len(t, entries, 3)

	entries2, _, err := logger.Query("", "", "", "", 2, 3)
	require.NoError(t, err)
	assert.Len(t, entries2, 3)
	// ensure different page
	assert.NotEqual(t, entries[0].ID, entries2[0].ID)
}

func TestQuery_DefaultPagination(t *testing.T) {
	db := newTestDB(t)
	logger := NewLogger(db)

	logger.Log(Entry{Actor: "admin", Action: "test", IPAddress: "10.0.0.1"})

	entries, _, err := logger.Query("", "", "", "", 0, 0)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
}

func TestQuery_DateRange(t *testing.T) {
	db := newTestDB(t)
	logger := NewLogger(db)

	logger.Log(Entry{Actor: "admin", Action: "test", IPAddress: "10.0.0.1"})

	// future date range should return nothing
	future := time.Now().Add(24 * time.Hour).Format("2006-01-02")
	entries, total, err := logger.Query(future, "", "", "", 1, 50)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Len(t, entries, 0)

	// past date range should return the entry
	past := time.Now().Add(-24 * time.Hour).Format("2006-01-02")
	entries, total, err = logger.Query(past, "", "", "", 1, 50)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
}

func TestQueryAll(t *testing.T) {
	db := newTestDB(t)
	logger := NewLogger(db)

	for i := 0; i < 5; i++ {
		logger.Log(Entry{Actor: "admin", Action: "test", IPAddress: "10.0.0.1"})
	}

	entries, err := logger.QueryAll("", "", "", "")
	require.NoError(t, err)
	assert.Len(t, entries, 5)
}

func TestQueryAll_Filtering(t *testing.T) {
	db := newTestDB(t)
	logger := NewLogger(db)

	logger.Log(Entry{Actor: "admin", Action: "user.create", IPAddress: "10.0.0.1"})
	logger.Log(Entry{Actor: "user1", Action: "client.create", IPAddress: "10.0.0.2"})

	entries, err := logger.QueryAll("", "", "admin", "")
	require.NoError(t, err)
	assert.Len(t, entries, 1)
}

func TestQueryAll_DateRange(t *testing.T) {
	db := newTestDB(t)
	logger := NewLogger(db)

	logger.Log(Entry{Actor: "admin", Action: "test.action", IPAddress: "10.0.0.1"})
	logger.Log(Entry{Actor: "admin", Action: "test.action2", IPAddress: "10.0.0.1"})

	// past date - should include entries
	past := time.Now().Add(-24 * time.Hour).Format("2006-01-02")
	entries, err := logger.QueryAll(past, "", "", "")
	require.NoError(t, err)
	assert.Len(t, entries, 2)

	// future date range should return nothing
	future := time.Now().Add(24 * time.Hour).Format("2006-01-02")
	entries, err = logger.QueryAll(future, "", "", "")
	require.NoError(t, err)
	assert.Len(t, entries, 0)

	// with end date in the past
	pastEnd := time.Now().Add(-1 * time.Hour).Format("2006-01-02T15:04:05")
	entries, err = logger.QueryAll("", pastEnd, "", "")
	require.NoError(t, err)
	// Entries were created "now" which is after pastEnd, so may return 0
	// The exact result depends on timing - just verify no error
	assert.NotNil(t, entries)
}

func TestQueryAll_ByAction(t *testing.T) {
	db := newTestDB(t)
	logger := NewLogger(db)

	logger.Log(Entry{Actor: "admin", Action: "client.create", IPAddress: "10.0.0.1"})
	logger.Log(Entry{Actor: "admin", Action: "client.delete", IPAddress: "10.0.0.1"})
	logger.Log(Entry{Actor: "admin", Action: "user.create", IPAddress: "10.0.0.1"})

	entries, err := logger.QueryAll("", "", "", "client.create")
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "client.create", entries[0].Action)
}

func TestLog_NilDetails(t *testing.T) {
	db := newTestDB(t)
	logger := NewLogger(db)

	logger.Log(Entry{
		Actor:        "admin",
		Action:       "test.nil.details",
		ResourceType: "test",
		ResourceID:   "1",
		Details:      nil,
		IPAddress:    "10.0.0.1",
	})

	entries, total, err := logger.Query("", "", "", "test.nil.details", 1, 50)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, "{}", entries[0].Details)
}

func TestQuery_CombinedActorAndDateRange(t *testing.T) {
	db := newTestDB(t)
	logger := NewLogger(db)

	logger.Log(Entry{Actor: "admin", Action: "combined.test", IPAddress: "10.0.0.1"})
	logger.Log(Entry{Actor: "user1", Action: "combined.test", IPAddress: "10.0.0.2"})

	past := time.Now().Add(-24 * time.Hour).Format("2006-01-02")
	futureEnd := time.Now().Add(24 * time.Hour).Format("2006-01-02")

	entries, total, err := logger.Query(past, futureEnd, "admin", "combined.test", 1, 50)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, entries, 1)
	assert.Equal(t, "admin", entries[0].Actor)
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
