package routes

import (
	"github.com/ashishsonamm/setu-splitwise/handlers"
	"github.com/ashishsonamm/setu-splitwise/middleware"
	"github.com/gorilla/mux"
)

func RegisterRoutes() *mux.Router {
	router := mux.NewRouter()

	router.HandleFunc("/api/user", handlers.CreateUser).Methods("POST")
	router.HandleFunc("/api/login", handlers.Login).Methods("POST")

	api := router.PathPrefix("/api").Subrouter()
	api.Use(middleware.JWTAuth)
	api.HandleFunc("/group", handlers.CreateGroup).Methods("POST")
	api.HandleFunc("/group/addUser", handlers.AddUserToGroup).Methods("POST")
	api.HandleFunc("/group/removeUser", handlers.RemoveUserFromGroup).Methods("POST")
	api.HandleFunc("/group/{groupId}/balances", handlers.GetGroupBalances).Methods("GET")
	api.HandleFunc("/group/{groupId}/balances/{userId}", handlers.GetUserBalanceInAGroup).Methods("GET")
	api.HandleFunc("/group/{groupId}/expenses", handlers.GetGroupExpenses).Methods("GET")

	api.HandleFunc("/expense", handlers.AddExpense).Methods("POST")
	api.HandleFunc("/users/{userId}/balance", handlers.GetPersonalBalance).Methods("GET")

	api.HandleFunc("/settle/personal", handlers.SettlePersonalBalance).Methods("POST")
	api.HandleFunc("/settle/{groupId}/group/{user1Id}/{user2Id}", handlers.SettleGroupBalanceBetweenUsers).Methods("POST")

	return router
}
