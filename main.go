package main

import (
	"github.com/ashishsonamm/setu-splitwise/routes"
	"github.com/ashishsonamm/setu-splitwise/utils"
	"github.com/joho/godotenv"
	"log"
	"net/http"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}
	utils.InitDB()
	router := routes.RegisterRoutes()
	log.Fatal(http.ListenAndServe(":8080", router))
}
