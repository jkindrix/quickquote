package repository

import (
	"strings"
	"testing"
)

func TestTableColumns_Select(t *testing.T) {
	tc := TableColumns{
		TableName: "users",
		Columns:   []string{"id", "name", "email", "created_at"},
	}

	result := tc.Select()
	expected := "id, name, email, created_at"

	if result != expected {
		t.Errorf("Select() = %q, want %q", result, expected)
	}
}

func TestTableColumns_SelectPrefixed(t *testing.T) {
	tc := TableColumns{
		TableName: "users",
		Columns:   []string{"id", "name", "email"},
	}

	result := tc.SelectPrefixed()
	expected := "users.id, users.name, users.email"

	if result != expected {
		t.Errorf("SelectPrefixed() = %q, want %q", result, expected)
	}
}

func TestTableColumns_SelectAliased(t *testing.T) {
	tc := TableColumns{
		TableName: "users",
		Columns:   []string{"id", "name"},
	}

	result := tc.SelectAliased("u")
	expected := "users.id AS u_id, users.name AS u_name"

	if result != expected {
		t.Errorf("SelectAliased() = %q, want %q", result, expected)
	}
}

func TestTableColumns_Placeholders(t *testing.T) {
	tc := TableColumns{
		TableName: "users",
		Columns:   []string{"id", "name", "email", "created_at"},
	}

	result := tc.Placeholders()
	expected := "$1, $2, $3, $4"

	if result != expected {
		t.Errorf("Placeholders() = %q, want %q", result, expected)
	}
}

func TestTableColumns_PlaceholdersFrom(t *testing.T) {
	tc := TableColumns{
		TableName: "users",
		Columns:   []string{"id", "name", "email"},
	}

	result := tc.PlaceholdersFrom(5)
	expected := "$5, $6, $7"

	if result != expected {
		t.Errorf("PlaceholdersFrom(5) = %q, want %q", result, expected)
	}
}

func TestTableColumns_UpdateSet(t *testing.T) {
	tc := TableColumns{
		TableName: "users",
		Columns:   []string{"id", "name", "email", "updated_at"},
	}

	result := tc.UpdateSet()
	expected := "name = $2, email = $3, updated_at = $4"

	if result != expected {
		t.Errorf("UpdateSet() = %q, want %q", result, expected)
	}
}

func TestTableColumns_UpdateSetFrom(t *testing.T) {
	tc := TableColumns{
		TableName: "users",
		Columns:   []string{"id", "name", "email"},
	}

	result := tc.UpdateSetFrom(10)
	expected := "name = $10, email = $11"

	if result != expected {
		t.Errorf("UpdateSetFrom(10) = %q, want %q", result, expected)
	}
}

func TestTableColumns_InsertColumns(t *testing.T) {
	tc := TableColumns{
		TableName: "users",
		Columns:   []string{"id", "name", "email"},
	}

	// InsertColumns is same as Select
	if tc.InsertColumns() != tc.Select() {
		t.Error("InsertColumns() should equal Select()")
	}
}

func TestTableColumns_Count(t *testing.T) {
	tc := TableColumns{
		TableName: "users",
		Columns:   []string{"id", "name", "email", "created_at"},
	}

	if tc.Count() != 4 {
		t.Errorf("Count() = %d, want 4", tc.Count())
	}
}

func TestTableColumns_Without(t *testing.T) {
	tc := TableColumns{
		TableName: "users",
		Columns:   []string{"id", "name", "email", "password_hash", "created_at"},
	}

	filtered := tc.Without("password_hash", "created_at")

	if len(filtered.Columns) != 3 {
		t.Errorf("Without() resulted in %d columns, want 3", len(filtered.Columns))
	}

	expected := "id, name, email"
	if filtered.Select() != expected {
		t.Errorf("Without().Select() = %q, want %q", filtered.Select(), expected)
	}

	// Verify table name is preserved
	if filtered.TableName != "users" {
		t.Errorf("Without() changed TableName to %q", filtered.TableName)
	}
}

func TestTableColumns_Only(t *testing.T) {
	tc := TableColumns{
		TableName: "users",
		Columns:   []string{"id", "name", "email", "password_hash", "created_at"},
	}

	filtered := tc.Only("id", "email")

	if len(filtered.Columns) != 2 {
		t.Errorf("Only() resulted in %d columns, want 2", len(filtered.Columns))
	}

	expected := "id, email"
	if filtered.Select() != expected {
		t.Errorf("Only().Select() = %q, want %q", filtered.Select(), expected)
	}
}

func TestTableColumns_UpdateSet_SingleColumn(t *testing.T) {
	tc := TableColumns{
		TableName: "users",
		Columns:   []string{"id"},
	}

	result := tc.UpdateSet()
	if result != "" {
		t.Errorf("UpdateSet() with only id should be empty, got %q", result)
	}
}

func TestItoa(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{10, "10"},
		{99, "99"},
		{123, "123"},
		{1000, "1000"},
	}

	for _, tt := range tests {
		result := itoa(tt.input)
		if result != tt.expected {
			t.Errorf("itoa(%d) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// Verify actual column definitions match expected counts

func TestCallColumns(t *testing.T) {
	if CallColumns.TableName != "calls" {
		t.Errorf("CallColumns.TableName = %q, want %q", CallColumns.TableName, "calls")
	}

	// Should have at least the essential columns
	essentialColumns := []string{"id", "provider_call_id", "status", "created_at", "updated_at"}
	for _, col := range essentialColumns {
		found := false
		for _, c := range CallColumns.Columns {
			if c == col {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("CallColumns missing essential column %q", col)
		}
	}
}

func TestUserColumns(t *testing.T) {
	if UserColumns.TableName != "users" {
		t.Errorf("UserColumns.TableName = %q, want %q", UserColumns.TableName, "users")
	}

	essentialColumns := []string{"id", "email", "password_hash", "created_at", "updated_at"}
	for _, col := range essentialColumns {
		found := false
		for _, c := range UserColumns.Columns {
			if c == col {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("UserColumns missing essential column %q", col)
		}
	}
}

func TestPromptColumns(t *testing.T) {
	if PromptColumns.TableName != "prompts" {
		t.Errorf("PromptColumns.TableName = %q, want %q", PromptColumns.TableName, "prompts")
	}

	// Verify it has a good number of columns (prompts table is large)
	if len(PromptColumns.Columns) < 20 {
		t.Errorf("PromptColumns expected at least 20 columns, got %d", len(PromptColumns.Columns))
	}
}

func TestPersonaColumns(t *testing.T) {
	if PersonaColumns.TableName != "personas" {
		t.Errorf("PersonaColumns.TableName = %q, want %q", PersonaColumns.TableName, "personas")
	}

	essentialColumns := []string{"id", "bland_id", "name", "status", "created_at", "updated_at"}
	for _, col := range essentialColumns {
		found := false
		for _, c := range PersonaColumns.Columns {
			if c == col {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("PersonaColumns missing essential column %q", col)
		}
	}
}

func TestPathwayColumns(t *testing.T) {
	if PathwayColumns.TableName != "pathways" {
		t.Errorf("PathwayColumns.TableName = %q, want %q", PathwayColumns.TableName, "pathways")
	}

	essentialColumns := []string{"id", "bland_id", "name", "nodes", "edges", "status", "created_at", "updated_at"}
	for _, col := range essentialColumns {
		found := false
		for _, c := range PathwayColumns.Columns {
			if c == col {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("PathwayColumns missing essential column %q", col)
		}
	}
}

func TestKnowledgeBaseColumns(t *testing.T) {
	if KnowledgeBaseColumns.TableName != "knowledge_bases" {
		t.Errorf("KnowledgeBaseColumns.TableName = %q, want %q", KnowledgeBaseColumns.TableName, "knowledge_bases")
	}

	essentialColumns := []string{"id", "bland_id", "name", "status", "document_count", "created_at", "updated_at"}
	for _, col := range essentialColumns {
		found := false
		for _, c := range KnowledgeBaseColumns.Columns {
			if c == col {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("KnowledgeBaseColumns missing essential column %q", col)
		}
	}
}

func TestSettingsColumns(t *testing.T) {
	if SettingsColumns.TableName != "settings" {
		t.Errorf("SettingsColumns.TableName = %q, want %q", SettingsColumns.TableName, "settings")
	}

	essentialColumns := []string{"id", "key", "value", "value_type", "category"}
	for _, col := range essentialColumns {
		found := false
		for _, c := range SettingsColumns.Columns {
			if c == col {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("SettingsColumns missing essential column %q", col)
		}
	}
}

// Test that column lists don't have duplicates

func TestNoDuplicateColumns(t *testing.T) {
	allTables := []TableColumns{
		CallColumns,
		UserColumns,
		SessionColumns,
		PromptColumns,
		PersonaColumns,
		KnowledgeBaseColumns,
		KnowledgeBaseDocumentColumns,
		PathwayColumns,
		PathwayVersionColumns,
		SettingsColumns,
		QuoteJobColumns,
	}

	for _, tc := range allTables {
		seen := make(map[string]bool)
		for _, col := range tc.Columns {
			if seen[col] {
				t.Errorf("%s has duplicate column: %q", tc.TableName, col)
			}
			seen[col] = true
		}
	}
}

// Test that no column names have whitespace

func TestNoWhitespaceInColumns(t *testing.T) {
	allTables := []TableColumns{
		CallColumns,
		UserColumns,
		SessionColumns,
		PromptColumns,
		PersonaColumns,
		KnowledgeBaseColumns,
		KnowledgeBaseDocumentColumns,
		PathwayColumns,
		PathwayVersionColumns,
		SettingsColumns,
		QuoteJobColumns,
	}

	for _, tc := range allTables {
		for _, col := range tc.Columns {
			if strings.TrimSpace(col) != col {
				t.Errorf("%s has column with whitespace: %q", tc.TableName, col)
			}
			if col == "" {
				t.Errorf("%s has empty column name", tc.TableName)
			}
		}
	}
}
