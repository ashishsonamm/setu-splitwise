package handlers

import (
	"encoding/json"
	"github.com/ashishsonamm/setu-splitwise/utils"
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
)

type Expense struct {
	ID           int           `json:"id"`
	Description  string        `json:"description"`
	Amount       float64       `json:"amount"`
	SplitType    string        `json:"split_type"`   // Types like "equally", "percentage", "share-wise", "absolute"
	ExpenseType  string        `json:"expense_type"` // "group" or "personal"
	CreatedBy    int           `json:"created_by"`   // User ID of the person who created the expense
	GroupID      *int          `json:"group_id"`     // Group ID, if applicable
	Contributors []Contributor `json:"contributors"` // List of users who contributed to this expense
	AmountsOwed  []AmountOwed  `json:"amounts_owed"` // List of users and the amount they owe or are owed
}

type Contributor struct {
	UserID     int     `json:"user_id"`
	PaidAmount float64 `json:"paid_amount"`          // Amount paid by the user for this expense
	Percentage float64 `json:"percentage,omitempty"` // Percentage for percentage-based split
	Share      float64 `json:"share,omitempty"`      // Share for share-wise split
	Amount     float64 `json:"amount,omitempty"`     // Absolute amount for absolute-based split
}

type AmountOwed struct {
	UserID  int     `json:"user_id"`
	Owed    float64 `json:"owed"`
	Balance float64 `json:"balance"`
}

func AddExpense(w http.ResponseWriter, r *http.Request) {
	var expense Expense
	if err := json.NewDecoder(r.Body).Decode(&expense); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	expenseType := "personal"
	if expense.GroupID != nil {
		expenseType = "group"
	}

	query := `INSERT INTO expenses (group_id, description, amount, created_by, split_type, expense_type) 
			  VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`
	err := utils.DB.QueryRow(query, expense.GroupID, expense.Description, expense.Amount, expense.CreatedBy, expense.SplitType, expenseType).Scan(&expense.ID)
	if err != nil {
		http.Error(w, "Failed to add expense", http.StatusInternalServerError)
		return
	}

	totalAmount := expense.Amount
	totalShares := totalShares(expense.Contributors)

	for _, contributor := range expense.Contributors {
		var contributionAmount float64
		switch expense.SplitType {
		case "equal":
			contributionAmount = totalAmount / float64(len(expense.Contributors))
		case "percentage":
			contributionAmount = totalAmount * (contributor.Percentage / 100)
		case "absolute":
			contributionAmount = contributor.Amount
		case "share-wise":
			contributionAmount = totalAmount * (contributor.Share / totalShares)
		default:
			http.Error(w, "Invalid split type", http.StatusBadRequest)
			return
		}

		_, err := utils.DB.Exec(`INSERT INTO contributors (expense_id, user_id, paid_amount, contribution_amount) VALUES ($1, $2, $3, $4)`, expense.ID, contributor.UserID, contributor.PaidAmount, contributionAmount)
		if err != nil {
			http.Error(w, "Failed to add contributor", http.StatusInternalServerError)
			return
		}
	}

	for _, contributor := range expense.Contributors {
		var owedAmount float64
		switch expense.SplitType {
		case "equal":
			owedAmount = totalAmount / float64(len(expense.Contributors))
		case "percentage":
			owedAmount = totalAmount * (contributor.Percentage / 100)
		case "absolute":
			owedAmount = contributor.Amount
		case "share-wise":
			owedAmount = totalAmount * (contributor.Share / totalShares)
		default:
			http.Error(w, "Invalid split type", http.StatusBadRequest)
			return
		}
		balance := contributor.PaidAmount - owedAmount

		_, err := utils.DB.Exec(
			`INSERT INTO amounts_owed (expense_id, user_id, owed, balance) VALUES ($1, $2, $3, $4)`,
			expense.ID, contributor.UserID, owedAmount, balance,
		)
		if err != nil {
			http.Error(w, "Failed to store amount owed", http.StatusInternalServerError)
			return
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"message": "Expense added successfully", "expense_id": expense.ID})
}

func totalShares(contributors []Contributor) float64 {
	total := 0.0
	for _, c := range contributors {
		total += c.Share
	}
	return total
}

func GetPersonalBalance(w http.ResponseWriter, r *http.Request) {
	userIDStr := mux.Vars(r)["userId"]
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	query := `
		WITH personal_expenses AS (
    SELECT e.id AS expense_id
    FROM expenses e
    JOIN contributors c ON e.id = c.expense_id
    WHERE e.expense_type = 'personal' AND c.user_id = $1
)

SELECT 
    c.user_id,
    c.expense_id,
    SUM(c.paid_amount - c.contribution_amount) AS balance
FROM 
    contributors c
JOIN 
    personal_expenses pe ON c.expense_id = pe.expense_id
WHERE c.user_id != $1
GROUP BY 
    c.user_id, c.expense_id;
	`

	rows, err := utils.DB.Query(query, userID)
	if err != nil {
		http.Error(w, "Failed to fetch personal balance details", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var balances []map[string]interface{}
	for rows.Next() {
		var amount float64
		var fromUser, expenseId int

		err := rows.Scan(&fromUser, &expenseId, &amount)
		if err != nil {
			http.Error(w, "Error scanning balance details", http.StatusInternalServerError)
			return
		}

		balances = append(balances, map[string]interface{}{
			"amount": amount,
			"from":   fromUser,
			"to":     userID,
		})
	}

	if err := rows.Err(); err != nil {
		http.Error(w, "Error iterating over balance details", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(balances)
}
