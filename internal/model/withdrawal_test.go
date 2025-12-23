package model

import (
	"encoding/json"
	"testing"
	"time"
)

func TestWithdrawal_ScanFieldsPointers(t *testing.T) {
	var w Withdrawal

	fields := w.ScanFields()
	if len(fields) != 4 {
		t.Fatalf("len=%d want=4", len(fields))
	}

	userIDPtr, ok := fields[0].(*int)
	if !ok {
		t.Fatalf("field[0] type=%T want *int", fields[0])
	}
	numberPtr, ok := fields[1].(*string)
	if !ok {
		t.Fatalf("field[1] type=%T want *string", fields[1])
	}
	sumPtr, ok := fields[2].(*int)
	if !ok {
		t.Fatalf("field[2] type=%T want *int", fields[2])
	}
	timePtr, ok := fields[3].(*time.Time)
	if !ok {
		t.Fatalf("field[3] type=%T want *time.Time", fields[3])
	}

	*userIDPtr = 5
	*numberPtr = "ORD-1"
	*sumPtr = 300
	now := time.Date(2025, 12, 21, 12, 0, 0, 0, time.UTC)
	*timePtr = now

	if w.UserID != 5 || w.Number != "ORD-1" || w.Sum != 300 || !w.ProcessedAt.Equal(now) {
		t.Fatalf("withdrawal not populated via ScanFields: %+v", w)
	}
}

func TestWithdrawal_MarshalJSON(t *testing.T) {
	w := Withdrawal{
		UserID:      99,
		Number:      "ORD-42",
		Sum:         700,
		ProcessedAt: time.Date(2025, 12, 21, 13, 0, 0, 0, time.UTC),
	}

	b, err := json.Marshal(w)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal: %v; json=%s", err, string(b))
	}

	if m["number"] != "ORD-42" {
		t.Fatalf("number=%v want=ORD-42", m["number"])
	}
	if int(m["sum"].(float64)) != 700 {
		t.Fatalf("sum=%v want=700", m["sum"])
	}
	if m["processed_at"] != w.ProcessedAt.Format(time.RFC3339) {
		t.Fatalf("processed_at=%v want=%v", m["processed_at"], w.ProcessedAt.Format(time.RFC3339))
	}
	if _, ok := m["user_id"]; ok {
		t.Fatalf("user_id must not be present in MarshalJSON, json=%s", string(b))
	}
}
