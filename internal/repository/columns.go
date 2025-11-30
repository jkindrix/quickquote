// Package repository provides SQL column definitions for database operations.
// This file centralizes column lists to prevent drift between queries and ensure consistency.
package repository

import (
	"strings"
)

// Table column definitions - these must match the database schema exactly.
// When adding/removing columns, update these definitions and all relevant queries will use them.

// CallColumns defines the columns for the calls table.
var CallColumns = TableColumns{
	TableName: "calls",
	Columns: []string{
		"id",
		"provider_call_id",
		"provider",
		"phone_number",
		"from_number",
		"caller_name",
		"status",
		"started_at",
		"ended_at",
		"duration_seconds",
		"transcript",
		"transcript_json",
		"recording_url",
		"quote_summary",
		"extracted_data",
		"error_message",
		"created_at",
		"updated_at",
		"deleted_at",
	},
}

// UserColumns defines the columns for the users table.
var UserColumns = TableColumns{
	TableName: "users",
	Columns: []string{
		"id",
		"email",
		"password_hash",
		"created_at",
		"updated_at",
		"deleted_at",
	},
}

// SessionColumns defines the columns for the sessions table.
var SessionColumns = TableColumns{
	TableName: "sessions",
	Columns: []string{
		"id",
		"user_id",
		"token",
		"expires_at",
		"created_at",
		"last_active_at",
		"ip_address",
		"user_agent",
		"previous_token",
		"rotated_at",
	},
}

// PromptColumns defines the columns for the prompts table.
var PromptColumns = TableColumns{
	TableName: "prompts",
	Columns: []string{
		"id",
		"name",
		"description",
		"task",
		"voice",
		"language",
		"model",
		"temperature",
		"interruption_threshold",
		"max_duration",
		"first_sentence",
		"wait_for_greeting",
		"transfer_phone_number",
		"transfer_list",
		"voicemail_action",
		"voicemail_message",
		"record",
		"background_track",
		"noise_cancellation",
		"knowledge_base_ids",
		"custom_tool_ids",
		"summary_prompt",
		"dispositions",
		"analysis_schema",
		"keywords",
		"is_default",
		"is_active",
		"created_at",
		"updated_at",
		"deleted_at",
	},
}

// PersonaColumns defines the columns for the personas table.
var PersonaColumns = TableColumns{
	TableName: "personas",
	Columns: []string{
		"id",
		"bland_id",
		"name",
		"description",
		"voice",
		"language",
		"voice_settings",
		"personality",
		"background_story",
		"system_prompt",
		"behavior",
		"knowledge_bases",
		"tools",
		"status",
		"is_default",
		"last_synced_at",
		"sync_error",
		"created_at",
		"updated_at",
	},
}

// KnowledgeBaseColumns defines the columns for the knowledge_bases table.
var KnowledgeBaseColumns = TableColumns{
	TableName: "knowledge_bases",
	Columns: []string{
		"id",
		"bland_id",
		"name",
		"description",
		"vector_db_id",
		"status",
		"document_count",
		"last_synced_at",
		"sync_error",
		"metadata",
		"created_at",
		"updated_at",
	},
}

// KnowledgeBaseDocumentColumns defines the columns for the knowledge_base_documents table.
var KnowledgeBaseDocumentColumns = TableColumns{
	TableName: "knowledge_base_documents",
	Columns: []string{
		"id",
		"knowledge_base_id",
		"bland_doc_id",
		"name",
		"content_type",
		"content_hash",
		"size_bytes",
		"chunk_count",
		"status",
		"error_message",
		"created_at",
		"updated_at",
	},
}

// PathwayColumns defines the columns for the pathways table.
var PathwayColumns = TableColumns{
	TableName: "pathways",
	Columns: []string{
		"id",
		"bland_id",
		"name",
		"description",
		"version",
		"status",
		"nodes",
		"edges",
		"start_node_id",
		"last_synced_at",
		"sync_error",
		"is_published",
		"published_at",
		"created_at",
		"updated_at",
	},
}

// PathwayVersionColumns defines the columns for the pathway_versions table.
var PathwayVersionColumns = TableColumns{
	TableName: "pathway_versions",
	Columns: []string{
		"id",
		"pathway_id",
		"version",
		"nodes",
		"edges",
		"change_notes",
		"created_at",
		"created_by",
	},
}

// SettingsColumns defines the columns for the settings table.
var SettingsColumns = TableColumns{
	TableName: "settings",
	Columns: []string{
		"id",
		"key",
		"value",
		"value_type",
		"category",
		"description",
		"created_at",
		"updated_at",
	},
}

// QuoteJobColumns defines the columns for the quote_jobs table.
var QuoteJobColumns = TableColumns{
	TableName: "quote_jobs",
	Columns: []string{
		"id",
		"call_id",
		"status",
		"retry_count",
		"last_error",
		"created_at",
		"updated_at",
		"started_at",
		"completed_at",
	},
}

// TableColumns provides helper methods for generating SQL fragments.
type TableColumns struct {
	TableName string
	Columns   []string
}

// Select returns a comma-separated list of columns for SELECT queries.
// Example: "id, name, email, created_at"
func (tc TableColumns) Select() string {
	return strings.Join(tc.Columns, ", ")
}

// SelectPrefixed returns columns prefixed with table name for joins.
// Example: "users.id, users.name, users.email"
func (tc TableColumns) SelectPrefixed() string {
	prefixed := make([]string, len(tc.Columns))
	for i, col := range tc.Columns {
		prefixed[i] = tc.TableName + "." + col
	}
	return strings.Join(prefixed, ", ")
}

// SelectAliased returns columns with aliases for joins.
// Example: "users.id AS users_id, users.name AS users_name"
func (tc TableColumns) SelectAliased(alias string) string {
	aliased := make([]string, len(tc.Columns))
	for i, col := range tc.Columns {
		aliased[i] = tc.TableName + "." + col + " AS " + alias + "_" + col
	}
	return strings.Join(aliased, ", ")
}

// Placeholders returns numbered placeholders for the columns.
// Example: "$1, $2, $3, $4" for 4 columns
func (tc TableColumns) Placeholders() string {
	placeholders := make([]string, len(tc.Columns))
	for i := range tc.Columns {
		placeholders[i] = "$" + itoa(i+1)
	}
	return strings.Join(placeholders, ", ")
}

// PlaceholdersFrom returns numbered placeholders starting from a given number.
// Example: PlaceholdersFrom(3) for 4 columns returns "$3, $4, $5, $6"
func (tc TableColumns) PlaceholdersFrom(start int) string {
	placeholders := make([]string, len(tc.Columns))
	for i := range tc.Columns {
		placeholders[i] = "$" + itoa(start+i)
	}
	return strings.Join(placeholders, ", ")
}

// UpdateSet returns the SET clause for UPDATE queries (excluding first column, assumed to be id).
// Example: "name = $2, email = $3, updated_at = $4"
func (tc TableColumns) UpdateSet() string {
	if len(tc.Columns) <= 1 {
		return ""
	}
	parts := make([]string, len(tc.Columns)-1)
	for i := 1; i < len(tc.Columns); i++ {
		parts[i-1] = tc.Columns[i] + " = $" + itoa(i+1)
	}
	return strings.Join(parts, ", ")
}

// UpdateSetFrom returns the SET clause starting from a given placeholder number.
func (tc TableColumns) UpdateSetFrom(start int) string {
	if len(tc.Columns) <= 1 {
		return ""
	}
	parts := make([]string, len(tc.Columns)-1)
	for i := 1; i < len(tc.Columns); i++ {
		parts[i-1] = tc.Columns[i] + " = $" + itoa(start+i-1)
	}
	return strings.Join(parts, ", ")
}

// InsertColumns returns a comma-separated list of columns for INSERT queries.
// Same as Select() but explicitly named for clarity.
func (tc TableColumns) InsertColumns() string {
	return tc.Select()
}

// Count returns the number of columns.
func (tc TableColumns) Count() int {
	return len(tc.Columns)
}

// Without returns a new TableColumns excluding the specified columns.
func (tc TableColumns) Without(exclude ...string) TableColumns {
	excludeMap := make(map[string]bool, len(exclude))
	for _, col := range exclude {
		excludeMap[col] = true
	}

	filtered := make([]string, 0, len(tc.Columns))
	for _, col := range tc.Columns {
		if !excludeMap[col] {
			filtered = append(filtered, col)
		}
	}

	return TableColumns{
		TableName: tc.TableName,
		Columns:   filtered,
	}
}

// Only returns a new TableColumns containing only the specified columns.
func (tc TableColumns) Only(include ...string) TableColumns {
	includeMap := make(map[string]bool, len(include))
	for _, col := range include {
		includeMap[col] = true
	}

	filtered := make([]string, 0, len(include))
	for _, col := range tc.Columns {
		if includeMap[col] {
			filtered = append(filtered, col)
		}
	}

	return TableColumns{
		TableName: tc.TableName,
		Columns:   filtered,
	}
}

// itoa converts an integer to a string without importing strconv.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	if i < 0 {
		return "-" + itoa(-i)
	}

	var result []byte
	for i > 0 {
		result = append([]byte{byte('0' + i%10)}, result...)
		i /= 10
	}
	return string(result)
}
