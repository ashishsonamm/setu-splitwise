package handlers

import (
	"encoding/json"
	"github.com/ashishsonamm/setu-splitwise/models"
	"github.com/ashishsonamm/setu-splitwise/utils"
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
)

func CreateGroup(w http.ResponseWriter, r *http.Request) {
	var group models.Group
	if err := json.NewDecoder(r.Body).Decode(&group); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	query := `INSERT INTO groups (name) VALUES ($1) RETURNING id`
	err := utils.DB.QueryRow(query, group.Name).Scan(&group.ID)
	if err != nil {
		http.Error(w, "Failed to create group", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"message": "Group created successfully", "group_id": group.ID})
}

type AddUserToGroupRequest struct {
	GroupID int `json:"groupId"`
	UserID  int `json:"userId"`
}

func AddUserToGroup(w http.ResponseWriter, r *http.Request) {
	var req AddUserToGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var exists bool
	err := utils.DB.QueryRow("SELECT EXISTS (SELECT 1 FROM groups WHERE id = $1)", req.GroupID).Scan(&exists)
	if err != nil || !exists {
		http.Error(w, "Group not found", http.StatusNotFound)
		return
	}

	err = utils.DB.QueryRow("SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)", req.UserID).Scan(&exists)
	if err != nil || !exists {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	_, err = utils.DB.Exec("INSERT INTO group_users (group_id, user_id) VALUES ($1, $2)", req.GroupID, req.UserID)
	if err != nil {
		http.Error(w, "Failed to add user to group", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "User added to group successfully"})
}

type RemoveUserFromGroupRequest struct {
	GroupID int `json:"groupId"`
	UserID  int `json:"userId"`
}

func RemoveUserFromGroup(w http.ResponseWriter, r *http.Request) {
	var req RemoveUserFromGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var exists bool
	err := utils.DB.QueryRow("SELECT EXISTS (SELECT 1 FROM group_users WHERE group_id = $1 AND user_id = $2)", req.GroupID, req.UserID).Scan(&exists)
	if err != nil || !exists {
		http.Error(w, "User not found in group", http.StatusNotFound)
		return
	}

	_, err = utils.DB.Exec("DELETE FROM group_users WHERE group_id = $1 AND user_id = $2", req.GroupID, req.UserID)
	if err != nil {
		http.Error(w, "Failed to remove user from group", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "User removed from group successfully"})
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
		http.Error(w, "Failed to fetch user balance", http.StatusInternalServerError)
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
