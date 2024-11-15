package handlers

import (
	"encoding/json"
	"github.com/ashishsonamm/setu-splitwise/models"
	"github.com/ashishsonamm/setu-splitwise/utils"
	"net/http"
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

func AddUserToGroup(w http.ResponseWriter, r *http.Request) {
	var req models.AddOrRemoveUserToGroupRequest
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

func RemoveUserFromGroup(w http.ResponseWriter, r *http.Request) {
	var req models.AddOrRemoveUserToGroupRequest
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
