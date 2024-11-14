package handlers

import (
	"encoding/json"
	"github.com/ashishsonamm/setu-splitwise/utils"
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
)

func GetDashboard(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)

	query := `
		SELECT 
			SUM(CASE WHEN e.expense_type = 'group' THEN c.amount ELSE 0 END) AS group_balance,
			SUM(CASE WHEN e.expense_type = 'personal' THEN c.amount ELSE 0 END) AS personal_balance
		FROM contributors c
		JOIN expenses e ON c.expense_id = e.id
		WHERE c.user_id = $1
	`
	row := utils.DB.QueryRow(query, userID)
	var groupBalance, personalBalance float64
	if err := row.Scan(&groupBalance, &personalBalance); err != nil {
		http.Error(w, "Failed to fetch dashboard data", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"group_balance":    groupBalance,
		"personal_balance": personalBalance,
	})
}

func UserDashboardHandler(w http.ResponseWriter, r *http.Request) {
	userID, _ := strconv.Atoi(mux.Vars(r)["user_id"])

	rows, err := utils.DB.Query(`
        SELECT 
            SUM(CASE WHEN ec.user_id = $1 THEN ec.amount ELSE -ec.amount END) AS balance
        FROM expense_contributors ec
        JOIN expenses e ON ec.expense_id = e.id
        WHERE ec.user_id = $1 OR e.created_by = $1
        GROUP BY ec.user_id`, userID)
	if err != nil {
		http.Error(w, "Failed to fetch user balance", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var balance float64
	for rows.Next() {
		rows.Scan(&balance)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"balance": balance})
}
