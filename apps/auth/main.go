// Command auth-service starts the ForgeOS development authentication server.
//
// Usage:
//
//	DEVOS_JWT_SECRET=dev-secret-k8s-switch go run ./apps/auth/
//
// Environment variables:
//
//	PORT             - HTTP listen port (default 8081)
//	DEVOS_JWT_SECRET - HMAC-SHA256 secret for JWT signing (required)
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/apps/auth/authservice"
)

func main() {
	secret := os.Getenv("DEVOS_JWT_SECRET")
	if secret == "" {
		log.Fatal("DEVOS_JWT_SECRET is required")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	auth := authservice.NewAuthenticator([]byte(secret))

	mux := http.NewServeMux()
	mux.HandleFunc("/auth/login", auth.LoginHandler)
	mux.HandleFunc("/auth/me", auth.MeHandler)
	mux.HandleFunc("/auth/refresh", auth.RefreshHandler)

	handler := authservice.CORSMiddleware(mux)

	addr := fmt.Sprintf(":%s", port)
	log.Printf("auth-service: listening on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("auth-service: %v", err)
	}
}
