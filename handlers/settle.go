package handlers

import (
	"encoding/json"
	"github.com/ashishsonamm/setu-splitwise/utils"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"strconv"
)

func SettleBalance(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID       int     `json:"user_id"`
		SettleAmount float64 `json:"settle_amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	userID := r.Context().Value("user_id").(int)

	_, err := utils.DB.Exec(
		`INSERT INTO settlements (from_user, to_user, amount, settled_at) VALUES ($1, $2, $3, NOW())`,
		userID, req.UserID, req.SettleAmount,
	)
	if err != nil {
		http.Error(w, "Failed to settle balance", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"message": "Balance settled successfully"})
}

func SettlePersonalBalance(w http.ResponseWriter, r *http.Request) {
	userIDStr := mux.Vars(r)["userId"]
	targetUserIDStr := r.URL.Query().Get("targetUserId")

	userID, err := strconv.Atoi(userIDStr)
	targetUserID, err := strconv.Atoi(targetUserIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	tx, err := utils.DB.Begin()
	if err != nil {
		http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	query := `
		SELECT e.id, c.user_id, c.contribution_amount, c.paid_amount
		FROM expenses e
		JOIN contributors c ON e.id = c.expense_id
		WHERE e.expense_type = 'personal' AND (c.user_id = $1 OR c.user_id = $2)
	`
	rows, err := tx.Query(query, userID, targetUserID)
	if err != nil {
		http.Error(w, "Failed to fetch expenses", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var expenseID, contributorID int
		var contributionAmount, paidAmount float64
		if err := rows.Scan(&expenseID, &contributorID, &contributionAmount, &paidAmount); err != nil {
			http.Error(w, "Failed to parse expense details", http.StatusInternalServerError)
			return
		}

		if contributorID == userID {
			if contributorID == userID {
				newPaidAmount := paidAmount + contributionAmount
				_, err := tx.Exec(`
					UPDATE contributors
					SET paid_amount = $1
					WHERE expense_id = $2 AND user_id = $3
				`, newPaidAmount, expenseID, contributorID)
				if err != nil {
					http.Error(w, "Failed to update paid amount for user 1", http.StatusInternalServerError)
					return
				}
			}
		} else if contributorID == targetUserID {
			_, err := tx.Exec(`
				UPDATE contributors
				SET paid_amount = 0
				WHERE expense_id = $1 AND user_id = $2
			`, expenseID, contributorID)
			if err != nil {
				http.Error(w, "Failed to update paid amount for target user", http.StatusInternalServerError)
				return
			}
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Personal balance settled successfully"))
}

func SettleGroupBalance(w http.ResponseWriter, r *http.Request) {
	userIDStr := mux.Vars(r)["userId"]
	targetUserIDStr := r.URL.Query().Get("targetUserId")
	groupIDStr := mux.Vars(r)["groupId"]

	userID, err := strconv.Atoi(userIDStr)
	targetUserID, err := strconv.Atoi(targetUserIDStr)
	groupID, err := strconv.Atoi(groupIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID, target user ID, or group ID", http.StatusBadRequest)
		return
	}

	query := `
		SELECT e.id, c.user_id, c.contribution_amount, c.paid_amount
		FROM expenses e
		JOIN contributors c ON e.id = c.expense_id
		WHERE e.expense_type = 'group' AND e.group_id = $1 AND (c.user_id = $2 OR c.user_id = $3)
	`
	rows, err := utils.DB.Query(query, groupID, userID, targetUserID)
	if err != nil {
		http.Error(w, "Failed to fetch group expenses", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var expenseID, contributorID int
		var contributionAmount, paidAmount float64
		if err := rows.Scan(&expenseID, &contributorID, &contributionAmount, &paidAmount); err != nil {
			http.Error(w, "Failed to parse group expense details", http.StatusInternalServerError)
			return
		}

		if contributorID == userID {
			newPaidAmount := paidAmount + contributionAmount
			query = `
				UPDATE contributors
				SET paid_amount = $1
				WHERE expense_id = $2 AND user_id = $3
			`
			_, err := utils.DB.Query(query, newPaidAmount, expenseID, contributorID)
			if err != nil {
				http.Error(w, "Failed to update paid amount for user 1", http.StatusInternalServerError)
				return
			}
		} else if contributorID == targetUserID {
			query := `
						UPDATE contributors
						SET paid_amount = 0
						WHERE expense_id = $1 AND user_id = $2
					`
			_, err := utils.DB.Query(query, expenseID, contributorID)

			if err != nil {
				log.Printf("Failed to execute query: %v", err)
				http.Error(w, "Failed to update paid amount", http.StatusInternalServerError)
				return
			}
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Group balance settled successfully"))
}
