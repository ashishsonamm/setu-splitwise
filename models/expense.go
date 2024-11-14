package models

type GroupExpense struct {
	ID        int     `json:"id"`
	GroupID   int     `json:"group_id"`
	Amount    float64 `json:"amount"`
	SplitType string  `json:"split_type"`
}

type GroupExpenseRequest struct {
	GroupID       int                  `json:"group_id"`
	Amount        float64              `json:"amount"`
	SplitType     string               `json:"split_type"`
	UserWhoPaid   int                  `json:"user_who_paid"`
	UserAmountMap []UserAmountMapEntry `json:"user_amount_map"`
}

type PersonalExpenseRequest struct {
	PayerID int     `json:"payer_id"`
	PayeeID int     `json:"payee_id"`
	Amount  float64 `json:"amount"`
}

type UserAmountMapEntry struct {
	UserID int     `json:"user_id"`
	Amount float64 `json:"amount"`
}
