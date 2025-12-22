package model

import (
	"encoding/json"
	"time"
)

type Model interface {
	ScanFields() []any
}

type Order struct {
	Number     string    `json:"number"`
	Status     string    `json:"status"`
	Accrual    int       `json:"accrual"`
	UploadedAt time.Time `json:"uploaded_at"`
	UserID     int       `json:"user_id"`
}

func (o *Order) ScanFields() []any {
	return []any{
		&o.Number,
		&o.Status,
		&o.Accrual,
		&o.UploadedAt,
		&o.UserID,
	}
}

func (o Order) MarshalJSON() ([]byte, error) {
	type orderJSON struct {
		Number     string   `json:"number"`
		Status     string   `json:"status"`
		Accrual    *float32 `json:"accrual,omitempty"`
		UploadedAt string   `json:"uploaded_at"`
	}

	var acc *float32
	if o.Status == "PROCESSED" && o.Accrual > 0 {
		v := float32(o.Accrual) / 100
		acc = &v
	}

	return json.Marshal(orderJSON{
		Number:     o.Number,
		Status:     o.Status,
		Accrual:    acc,
		UploadedAt: o.UploadedAt.Format(time.RFC3339),
	})
}
