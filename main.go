package main

import (
	"fmt"
	"log"
	"net/http"

	"project-empat-golang/config"
	"project-empat-golang/routes"
)

func main() {
	config.LoadEnv()
	config.ConnectDatabase()

	mux := http.NewServeMux()

	// Register routes
	routes.Routes(mux)

	addr := fmt.Sprintf(":%s", config.AppConfig.AppPort)
	log.Printf("Server berjalan di http://localhost%s\n", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

