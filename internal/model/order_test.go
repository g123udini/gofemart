package model

import (
	"encoding/json"
	"testing"
	"time"
)

func TestOrder_ScanFieldsPointers(t *testing.T) {
	var o Order

	fields := o.ScanFields()
	if len(fields) != 5 {
		t.Fatalf("len=%d want=5", len(fields))
	}

	numberPtr, ok := fields[0].(*string)
	if !ok {
		t.Fatalf("field[0] type=%T want *string", fields[0])
	}
	statusPtr, ok := fields[1].(*string)
	if !ok {
		t.Fatalf("field[1] type=%T want *string", fields[1])
	}
	accrualPtr, ok := fields[2].(*int)
	if !ok {
		t.Fatalf("field[2] type=%T want *int", fields[2])
	}
	uploadedPtr, ok := fields[3].(*time.Time)
	if !ok {
		t.Fatalf("field[3] type=%T want *time.Time", fields[3])
	}
	userIDPtr, ok := fields[4].(*int)
	if !ok {
		t.Fatalf("field[4] type=%T want *int", fields[4])
	}

	*numberPtr = "n1"
	*statusPtr = "NEW"
	*accrualPtr = 123
	now := time.Date(2025, 12, 21, 9, 8, 7, 0, time.UTC)
	*uploadedPtr = now
	*userIDPtr = 77

	if o.Number != "n1" || o.Status != "NEW" || o.Accrual != 123 || !o.UploadedAt.Equal(now) || o.UserID != 77 {
		t.Fatalf("order not populated via ScanFields: %+v", o)
	}
}

func TestOrder_MarshalJSON_OmitsAccrualWhenNotProceeded(t *testing.T) {
	o := Order{
		Number:     "A1",
		Status:     "NEW",
		Accrual:    999,
		UploadedAt: time.Date(2025, 12, 21, 10, 0, 0, 0, time.UTC),
		UserID:     1,
	}

	b, err := json.Marshal(o)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal: %v; json=%s", err, string(b))
	}

	if m["number"] != "A1" {
		t.Fatalf("number=%v want=A1", m["number"])
	}
	if m["status"] != "NEW" {
		t.Fatalf("status=%v want=NEW", m["status"])
	}
	if _, ok := m["accrual"]; ok {
		t.Fatalf("accrual should be omitted, json=%s", string(b))
	}
	if m["uploaded_at"] != o.UploadedAt.Format(time.RFC3339) {
		t.Fatalf("uploaded_at=%v want=%v", m["uploaded_at"], o.UploadedAt.Format(time.RFC3339))
	}
	if _, ok := m["user_id"]; ok {
		t.Fatalf("user_id must not be present in MarshalJSON, json=%s", string(b))
	}
}

func TestOrder_MarshalJSON_IncludesAccrualWhenProceeded(t *testing.T) {
	o := Order{
		Number:     "B2",
		Status:     "PROCEEDED",
		Accrual:    500,
		UploadedAt: time.Date(2025, 12, 21, 11, 0, 0, 0, time.UTC),
		UserID:     2,
	}

	b, err := json.Marshal(o)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal: %v; json=%s", err, string(b))
	}

	if m["number"] != "B2" {
		t.Fatalf("number=%v want=B2", m["number"])
	}
	if m["status"] != "PROCEEDED" {
		t.Fatalf("status=%v want=PROCEEDED", m["status"])
	}
	if m["accrual"] == nil {
		t.Fatalf("accrual must be present, json=%s", string(b))
	}
	if int(m["accrual"].(float64)) != 500 {
		t.Fatalf("accrual=%v want=500", m["accrual"])
	}
	if m["uploaded_at"] != o.UploadedAt.Format(time.RFC3339) {
		t.Fatalf("uploaded_at=%v want=%v", m["uploaded_at"], o.UploadedAt.Format(time.RFC3339))
	}
	if _, ok := m["user_id"]; ok {
		t.Fatalf("user_id must not be present in MarshalJSON, json=%s", string(b))
	}
}
