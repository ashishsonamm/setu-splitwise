package handlers

import (
	"database/sql"
	"encoding/json"
	"github.com/ashishsonamm/setu-splitwise/models"
	"github.com/ashishsonamm/setu-splitwise/utils"
	"net/http"
)

func Login(w http.ResponseWriter, r *http.Request) {
	var loginReq models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	var user models.User
	query := `SELECT id, password FROM users WHERE email = $1`
	err := utils.DB.QueryRow(query, loginReq.Email).Scan(&user.ID, &user.Password)
	if err == sql.ErrNoRows || user.Password != loginReq.Password {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	} else if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	token, err := utils.CreateJWT(user.ID)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}
