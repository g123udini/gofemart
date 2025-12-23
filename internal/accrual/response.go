package accrual

const (
	AccrualStatusRegistered string = "REGISTERED"
	AccrualStatusProcessing string = "PROCESSING"
	AccrualStatusInvalid    string = "INVALID"
	AccrualStatusProcessed  string = "PROCESSED"
)

type Accrual struct {
	OrderID uint64   `json:"order,string"`
	Status  string   `json:"status"`
	Accrual *float64 `json:"accrual"`
}
