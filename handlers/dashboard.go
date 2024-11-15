package handlers

import (
	"encoding/json"
	"github.com/ashishsonamm/setu-splitwise/utils"
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
)

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

func GetGroupBalances(w http.ResponseWriter, r *http.Request) {
	groupIDStr := mux.Vars(r)["groupId"]
	groupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		http.Error(w, "Invalid group ID", http.StatusBadRequest)
		return
	}

	query := `
		SELECT 
			c.user_id,
			SUM(c.paid_amount - c.contribution_amount) AS balance
		FROM 
			contributors c
		JOIN 
			expenses e ON c.expense_id = e.id
		WHERE 
			e.group_id = $1
		GROUP BY 
			c.user_id;
	`

	rows, err := utils.DB.Query(query, groupID)
	if err != nil {
		http.Error(w, "Failed to fetch group balances", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	userBalances := make(map[int]float64)
	for rows.Next() {
		var userID int
		var netBalance float64
		if err := rows.Scan(&userID, &netBalance); err != nil {
			http.Error(w, "Failed to parse group balances", http.StatusInternalServerError)
			return
		}
		userBalances[userID] = netBalance
	}

	if err = rows.Err(); err != nil {
		http.Error(w, "Failed to fetch group balances", http.StatusInternalServerError)
		return
	}

	query = `
		SELECT 
			debtor_id, 
			creditor_id, 
			amount
		FROM 
			group_settlements
		WHERE 
			group_id = $1;
	`

	rows, err = utils.DB.Query(query, groupID)
	if err != nil {
		http.Error(w, "Failed to fetch group settlements", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var debtorID, creditorID int
		var amount float64
		if err := rows.Scan(&debtorID, &creditorID, &amount); err != nil {
			http.Error(w, "Failed to parse group settlements", http.StatusInternalServerError)
			return
		}

		userBalances[debtorID] += amount
		userBalances[creditorID] -= amount
	}

	if err = rows.Err(); err != nil {
		http.Error(w, "Failed to fetch group settlements", http.StatusInternalServerError)
		return
	}

	settlements := calculateSettlements(userBalances)

	json.NewEncoder(w).Encode(settlements)
}

func calculateSettlements(balances map[int]float64) []map[string]interface{} {
	var debtors []int
	var creditors []int

	for userID, balance := range balances {
		if balance < 0 {
			debtors = append(debtors, userID)
		} else if balance > 0 {
			creditors = append(creditors, userID)
		}
	}

	var settlements []map[string]interface{}

	for len(debtors) > 0 && len(creditors) > 0 {
		debtorID := debtors[0]
		creditorID := creditors[0]

		amountOwed := min(-balances[debtorID], balances[creditorID])

		settlements = append(settlements, map[string]interface{}{
			"from":   debtorID,
			"to":     creditorID,
			"amount": amountOwed,
		})

		balances[debtorID] += amountOwed
		balances[creditorID] -= amountOwed

		if balances[debtorID] == 0 {
			debtors = debtors[1:]
		}
		if balances[creditorID] == 0 {
			creditors = creditors[1:]
		}
	}

	return settlements
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func GetUserBalanceInAGroup(w http.ResponseWriter, r *http.Request) {
	groupIDStr := mux.Vars(r)["groupId"]
	userIDStr := mux.Vars(r)["userId"]
	groupID, err := strconv.Atoi(groupIDStr)
	userID, err2 := strconv.Atoi(userIDStr)
	if err != nil || err2 != nil {
		http.Error(w, "Invalid group or user ID", http.StatusBadRequest)
		return
	}

	query := `
        SELECT 
            c.user_id,
            SUM(c.paid_amount - c.contribution_amount) AS balance
        FROM 
            contributors c
        JOIN 
            expenses e ON c.expense_id = e.id
        WHERE 
            e.group_id = $1 AND c.user_id = $2
        GROUP BY 
            c.user_id;
    `

	rows, err := utils.DB.Query(query, groupID, userID)
	if err != nil {
		http.Error(w, "Failed to fetch user balance", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var userBalance float64
	if rows.Next() {
		if err := rows.Scan(&userID, &userBalance); err != nil {
			http.Error(w, "Failed to parse user balance", http.StatusInternalServerError)
			return
		}
	}

	if err = rows.Err(); err != nil {
		http.Error(w, "Failed to fetch user balance", http.StatusInternalServerError)
		return
	}

	query = `
        SELECT 
            debtor_id, 
            creditor_id, 
            amount
        FROM 
            group_settlements
        WHERE 
            group_id = $1 AND (debtor_id = $2 OR creditor_id = $2);
    `

	rows, err = utils.DB.Query(query, groupID, userID)
	if err != nil {
		http.Error(w, "Failed to fetch group settlements", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var debtorID, creditorID int
		var amount float64
		if err := rows.Scan(&debtorID, &creditorID, &amount); err != nil {
			http.Error(w, "Failed to parse group settlements", http.StatusInternalServerError)
			return
		}

		if debtorID == userID {
			userBalance += amount
		} else {
			userBalance -= amount
		}
	}

	if err = rows.Err(); err != nil {
		http.Error(w, "Failed to fetch group settlements", http.StatusInternalServerError)
		return
	}

	balanceDetails := getUserBalanceDetails(userBalance)

	query = `
        SELECT 
            c.user_id,
            SUM(c.paid_amount - c.contribution_amount) AS balance
        FROM 
            contributors c
        JOIN 
            expenses e ON c.expense_id = e.id
        WHERE 
            e.group_id = $1
        GROUP BY 
            c.user_id;
    `

	rows, err = utils.DB.Query(query, groupID)
	if err != nil {
		http.Error(w, "Failed to fetch group balances", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	userBalances := make(map[int]float64)
	for rows.Next() {
		var userID int
		var netBalance float64
		if err := rows.Scan(&userID, &netBalance); err != nil {
			http.Error(w, "Failed to parse group balances", http.StatusInternalServerError)
			return
		}
		userBalances[userID] = netBalance
	}

	if err = rows.Err(); err != nil {
		http.Error(w, "Failed to fetch group balances", http.StatusInternalServerError)
		return
	}

	settlements := calculateSettlements(userBalances)

	filteredSettlements := []map[string]interface{}{}
	for _, settlement := range settlements {
		from := settlement["from"].(int)
		to := settlement["to"].(int)

		if from == userID || to == userID {
			filteredSettlements = append(filteredSettlements, settlement)
		}
	}

	response := struct {
		UserBalance     map[string]interface{}   `json:"user_balance"`
		UserSettlements []map[string]interface{} `json:"user_settlements"`
	}{
		UserBalance:     balanceDetails,
		UserSettlements: filteredSettlements,
	}

	json.NewEncoder(w).Encode(response)
}

func getUserBalanceDetails(balance float64) map[string]interface{} {
	var balanceDetails = map[string]interface{}{
		"user_balance": balance,
	}

	if balance > 0 {
		balanceDetails["status"] = "owed"
		balanceDetails["amount"] = balance
	} else if balance < 0 {
		balanceDetails["status"] = "owes"
		balanceDetails["amount"] = -balance
	} else {
		balanceDetails["status"] = "settled"
		balanceDetails["amount"] = 0
	}

	return balanceDetails
}
