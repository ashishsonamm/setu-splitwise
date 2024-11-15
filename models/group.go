package models

type Group struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type AddOrRemoveUserToGroupRequest struct {
	GroupID int `json:"groupId"`
	UserID  int `json:"userId"`
}
