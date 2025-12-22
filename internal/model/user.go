package model

import "encoding/json"

type User struct {
	ID       int     `json:"id"`
	Login    string  `json:"name"`
	Password string  `json:"password"`
	Balance  Balance `json:"balance"`
}

type Balance struct {
	Current   int `json:"current"`
	Withdrawn int `json:"withdraw"`
}

func (u *User) ScanFields() []any {
	return []any{
		&u.ID,
		&u.Login,
		&u.Password,
		&u.Balance.Current,
		&u.Balance.Withdrawn,
	}
}

func (b Balance) MarshalJSON() ([]byte, error) {
	type balanceDTO struct {
		Current   float32 `json:"current"`
		Withdrawn float32 `json:"withdrawn"`
	}

	return json.Marshal(balanceDTO{
		Current:   float32(b.Current) / 100,
		Withdrawn: float32(b.Withdrawn) / 100,
	})
}
