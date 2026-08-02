// Command gateway runs the DevOS API Gateway HTTP server.
//
// Usage:
//
//	export DEVOS_JWT_SECRET="my-hmac-secret"
//	go run ./apps/gateway/cmd
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/auth"
	gw "github.com/Hafiz-Muhammad-Umar12/ForgeOS/apps/gateway"
	gwAuth "github.com/Hafiz-Muhammad-Umar12/ForgeOS/apps/gateway/auth"
	ingressBus "github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/bus"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/ingress"
)

func main() {
	// Configuration from environment.
	jwtSecret := os.Getenv("DEVOS_JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("DEVOS_JWT_SECRET is required. Set it to the same value used by the auth service.")
	}
	log.Printf("gateway: using JWT secret (len=%d)", len(jwtSecret))

	natsURL := os.Getenv("DEVOS_NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	listenAddr := os.Getenv("DEVOS_GATEWAY_ADDR")
	if listenAddr == "" {
		listenAddr = ":8080"
	}

	// Create dependencies.
	var provider auth.AuthProvider

	// Prefer JWT when a secret is configured.
	provider = gwAuth.NewJWTAdapter([]byte(jwtSecret))

	// Connect to the bus for the ingress.
	bus := ingressBus.NewNatsBus(ingressBus.WithNatsURL(natsURL))
	ctx := context.Background()
	if err := bus.Connect(ctx); err != nil {
		log.Fatalf("connect to bus: %v", err)
	}
	defer bus.Close(ctx)

	// Create the intent ingress (publishes to bus).
	ing := ingress.NewRESTAdapter(ingress.WithIngressBus(bus))

	// Create and start the gateway.
	cfg := gw.DefaultGatewayConfig()
	cfg.ListenAddr = listenAddr

	gateway := gw.NewGateway(cfg, provider, ing)

	if err := gateway.Init(ctx); err != nil {
		log.Fatalf("init: %v", err)
	}
	if err := gateway.Start(ctx); err != nil {
		log.Fatalf("start: %v", err)
	}

	fmt.Printf("gateway: listening on %s\n", cfg.ListenAddr)

	// Wait for signal.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\ngateway: shutting down...")
	if err := gateway.Stop(ctx); err != nil {
		log.Printf("gateway: stop error: %v", err)
	}
}
