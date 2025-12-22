package model

import (
	"encoding/json"
	"time"
)

type Withdrawal struct {
	UserID      int       `json:"user_id"`
	Number      string    `json:"number"`
	Sum         int       `json:"sum"`
	ProcessedAt time.Time `json:"processed_at"`
}

func (w *Withdrawal) ScanFields() []any {
	return []any{
		&w.UserID,
		&w.Number,
		&w.Sum,
		&w.ProcessedAt,
	}
}

func (w Withdrawal) MarshalJSON() ([]byte, error) {
	type dto struct {
		Order       string  `json:"order"`
		Sum         float32 `json:"sum"`
		ProcessedAt string  `json:"processed_at"`
	}

	return json.Marshal(dto{
		Order:       w.Number,
		Sum:         float32(w.Sum) / 100,
		ProcessedAt: w.ProcessedAt.Format(time.RFC3339),
	})
}
