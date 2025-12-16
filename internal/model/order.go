package model

import "time"

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
