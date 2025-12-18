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
	type orderJSON struct {
		Number      string `json:"number"`
		Sum         int    `json:"sum"`
		ProcessedAt string `json:"processed_at"`
	}

	return json.Marshal(orderJSON{
		Number:      w.Number,
		Sum:         w.Sum,
		ProcessedAt: w.ProcessedAt.Format(time.RFC3339),
	})
}
