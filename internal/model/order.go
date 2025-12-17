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
	UserId     int       `json:"user_id"`
}

func (o *Order) ScanFields() []any {
	return []any{
		&o.Number,
		&o.Status,
		&o.Accrual,
		&o.UploadedAt,
		&o.UserId,
	}
}

func (o Order) MarshalJSON() ([]byte, error) {
	type orderJSON struct {
		Number     string `json:"number"`
		Status     string `json:"status"`
		Accrual    *int   `json:"accrual,omitempty"`
		UploadedAt string `json:"uploaded_at"`
	}

	var accrual *int
	if o.Status == "PROCEEDED" {
		accrual = &o.Accrual
	}

	return json.Marshal(orderJSON{
		Number:     o.Number,
		Status:     o.Status,
		Accrual:    accrual,
		UploadedAt: o.UploadedAt.Format(time.RFC3339),
	})
}
