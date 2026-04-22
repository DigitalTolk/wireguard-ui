package audit

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/labstack/gommon/log"
)

// Logger handles audit log operations
type Logger struct {
	db *sql.DB
}

// Entry represents a single audit log entry
type Entry struct {
	Actor        string
	Action       string
	ResourceType string
	ResourceID   string
	Details      interface{}
	IPAddress    string
}

// LogEntry represents a stored audit log entry
type LogEntry struct {
	ID           int       `json:"id"`
	Timestamp    time.Time `json:"timestamp"`
	Actor        string    `json:"actor"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resource_type"`
	ResourceID   string    `json:"resource_id"`
	Details      string    `json:"details"`
	IPAddress    string    `json:"ip_address"`
}

// NewLogger creates a new audit logger
func NewLogger(db *sql.DB) *Logger {
	return &Logger{db: db}
}

// Log records an audit log entry
func (l *Logger) Log(entry Entry) {
	if l == nil || l.db == nil {
		return
	}

	details := "{}"
	if entry.Details != nil {
		if b, err := json.Marshal(entry.Details); err == nil {
			details = string(b)
		}
	}

	_, err := l.db.Exec(
		`INSERT INTO audit_logs (actor, action, resource_type, resource_id, details, ip_address)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		entry.Actor, entry.Action, entry.ResourceType, entry.ResourceID, details, entry.IPAddress,
	)
	if err != nil {
		log.Errorf("Failed to write audit log: %v", err)
	}
}

// LogWithUser records an audit log entry with explicit username
func (l *Logger) LogWithUser(username, action, resourceType, resourceID, ipAddress string, details interface{}) {
	l.Log(Entry{
		Actor:        username,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Details:      details,
		IPAddress:    ipAddress,
	})
}

// Query returns audit log entries with optional filtering
const maxPerPage = 200
const maxExportRows = 100000

func buildWhereClause(from, to, actor, action string) (string, []interface{}) {
	where := "WHERE 1=1"
	args := make([]interface{}, 0)
	if from != "" {
		where += " AND timestamp >= ?"
		args = append(args, from)
	}
	if to != "" {
		where += " AND timestamp <= ?"
		args = append(args, to)
	}
	if actor != "" {
		where += " AND actor = ?"
		args = append(args, actor)
	}
	if action != "" {
		where += " AND action = ?"
		args = append(args, action)
	}
	return where, args
}

func scanLogEntries(rows *sql.Rows) ([]LogEntry, error) {
	var entries []LogEntry
	for rows.Next() {
		var e LogEntry
		if err := rows.Scan(&e.ID, &e.Timestamp, &e.Actor, &e.Action, &e.ResourceType, &e.ResourceID, &e.Details, &e.IPAddress); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// Query returns audit log entries with optional filtering
func (l *Logger) Query(from, to string, actor, action string, page, perPage int) ([]LogEntry, int, error) {
	if perPage <= 0 {
		perPage = 50
	}
	if perPage > maxPerPage {
		perPage = maxPerPage
	}
	if page <= 0 {
		page = 1
	}

	where, args := buildWhereClause(from, to, actor, action)

	var total int
	if err := l.db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM audit_logs %s", where), args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * perPage
	query := fmt.Sprintf("SELECT id, timestamp, actor, action, resource_type, resource_id, details, ip_address FROM audit_logs %s ORDER BY timestamp DESC LIMIT ? OFFSET ?", where)
	args = append(args, perPage, offset)

	rows, err := l.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	entries, err := scanLogEntries(rows)
	return entries, total, err
}

// QueryAll returns audit log entries matching filters (for export), capped at maxExportRows
func (l *Logger) QueryAll(from, to string, actor, action string) ([]LogEntry, error) {
	where, args := buildWhereClause(from, to, actor, action)

	query := fmt.Sprintf("SELECT id, timestamp, actor, action, resource_type, resource_id, details, ip_address FROM audit_logs %s ORDER BY timestamp DESC LIMIT ?", where)
	args = append(args, maxExportRows)

	rows, err := l.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanLogEntries(rows)
}
