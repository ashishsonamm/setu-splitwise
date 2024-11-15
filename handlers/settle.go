package handlers

import (
	"encoding/json"
	"github.com/ashishsonamm/setu-splitwise/models"
	"github.com/ashishsonamm/setu-splitwise/utils"
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
)

func SettlePersonalBalance(w http.ResponseWriter, r *http.Request) {
	var req models.PersonalExpenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	query := `
        SELECT 
            SUM(ao.balance) AS balance
        FROM 
            amounts_owed ao
        JOIN 
            expenses e ON ao.expense_id = e.id
        WHERE 
            e.expense_type = 'personal' AND (ao.user_id = $1 OR ao.user_id = $2)
        GROUP BY 
            ao.user_id
        HAVING 
            SUM(ao.balance) < 0
    `

	var debtorBalance, creditorBalance float64
	err := utils.DB.QueryRow(query, req.PayerID, req.PayeeID).Scan(&debtorBalance)
	if err != nil {
		http.Error(w, "Failed to fetch personal balance", http.StatusInternalServerError)
		return
	}

	creditorBalance = -debtorBalance

	if debtorBalance < 0 && creditorBalance > 0 {
		_, err = utils.DB.Exec(
			"INSERT INTO personal_settlements (debtor_id, creditor_id, amount) VALUES ($1, $2, $3)",
			req.PayerID, req.PayeeID, -debtorBalance,
		)
		if err != nil {
			http.Error(w, "Failed to create personal settlement", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":   "Personal balance settled",
			"settled":   -debtorBalance,
			"remaining": 0,
		})
	} else {
		http.Error(w, "No balance to settle", http.StatusBadRequest)
	}
}

func SettleGroupBalanceBetweenUsers(w http.ResponseWriter, r *http.Request) {
	groupIDStr := mux.Vars(r)["groupId"]
	user1IDStr := mux.Vars(r)["user1Id"]
	user2IDStr := mux.Vars(r)["user2Id"]

	groupID, err := strconv.Atoi(groupIDStr)
	user1ID, err2 := strconv.Atoi(user1IDStr)
	user2ID, err3 := strconv.Atoi(user2IDStr)
	if err != nil || err2 != nil || err3 != nil {
		http.Error(w, "Invalid group or user ID", http.StatusBadRequest)
		return
	}

	query := `
        SELECT 
            SUM(ao.balance) AS user1_balance, 
            SUM(ao2.balance) AS user2_balance
        FROM 
            amounts_owed ao
        JOIN 
            amounts_owed ao2 ON ao.expense_id = ao2.expense_id
        JOIN 
            expenses e ON ao.expense_id = e.id
        WHERE 
            e.group_id = $1 AND ao.user_id = $2 AND ao2.user_id = $3
        GROUP BY 
            ao.user_id, ao2.user_id
    `

	var user1Balance, user2Balance float64
	err = utils.DB.QueryRow(query, groupID, user1ID, user2ID).Scan(&user1Balance, &user2Balance)
	if err != nil {
		http.Error(w, "Failed to fetch group balances", http.StatusInternalServerError)
		return
	}

	if user1Balance < 0 && user2Balance > 0 {
		amount := min(-user1Balance, user2Balance)
		_, err = utils.DB.Exec(
			"INSERT INTO group_settlements (group_id, debtor_id, creditor_id, amount) VALUES ($1, $2, $3, $4)",
			groupID, user1ID, user2ID, amount,
		)
		if err != nil {
			http.Error(w, "Failed to create group settlement", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":   "Group balance settled",
			"settled":   amount,
			"remaining": 0,
		})
	} else {
		http.Error(w, "No balance to settle", http.StatusBadRequest)
	}
}
