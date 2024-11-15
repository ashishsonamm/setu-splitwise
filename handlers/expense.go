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

func GetGroupExpenses(w http.ResponseWriter, r *http.Request) {
	groupIDStr := mux.Vars(r)["groupId"]
	groupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		http.Error(w, "Invalid group ID", http.StatusBadRequest)
		return
	}

	query := `
		SELECT e.id, e.description, e.amount, e.split_type, e.expense_type, e.created_by,
		       ao.user_id, ao.owed, c.contribution_amount, c.paid_amount, ao.balance
		FROM expenses e
		JOIN amounts_owed ao ON e.id = ao.expense_id
		JOIN contributors c ON ao.expense_id = c.expense_id and ao.user_id = c.user_id
		WHERE e.group_id = $1
	`

	rows, err := utils.DB.Query(query, groupID)
	if err != nil {
		http.Error(w, "Failed to fetch group expenses", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	expenses := make(map[int]map[string]interface{})
	for rows.Next() {
		var expenseID, createdBy, userID int
		var description, splitType, expenseType string
		var amount, owed, contributionAmount, paidAmount, balance float64

		if err := rows.Scan(&expenseID, &description, &amount, &splitType, &expenseType, &createdBy, &userID, &owed, &contributionAmount, &paidAmount, &balance); err != nil {
			http.Error(w, "Failed to parse expense details", http.StatusInternalServerError)
			return
		}

		if _, exists := expenses[expenseID]; !exists {
			expenses[expenseID] = map[string]interface{}{
				"id":           expenseID,
				"description":  description,
				"amount":       amount,
				"split_type":   splitType,
				"expense_type": expenseType,
				"created_by":   createdBy,
				"contributors": []map[string]interface{}{},
			}
		}

		expenses[expenseID]["contributors"] = append(
			expenses[expenseID]["contributors"].([]map[string]interface{}),
			map[string]interface{}{
				"user_id":             userID,
				"contribution_amount": contributionAmount,
				"paid_amount":         paidAmount,
				"balance":             balance,
			},
		)
	}

	if err = rows.Err(); err != nil {
		http.Error(w, "Failed to fetch group expenses", http.StatusInternalServerError)
		return
	}

	var expenseList []map[string]interface{}
	for _, expense := range expenses {
		expenseList = append(expenseList, expense)
	}

	json.NewEncoder(w).Encode(expenseList)
}
