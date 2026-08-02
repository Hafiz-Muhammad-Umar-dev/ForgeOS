// Command gateway runs the DevOS API Gateway HTTP server.
//
// Usage:
//
//	export DEVOS_JWT_SECRET="my-hmac-secret"
//	export DEVOS_DATABASE_URL="postgres://user:pass@localhost:5432/devos?sslmode=disable"
//	go run ./apps/gateway/cmd
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	gw "github.com/Hafiz-Muhammad-Umar12/ForgeOS/apps/gateway"
	gwAuth "github.com/Hafiz-Muhammad-Umar12/ForgeOS/apps/gateway/auth"
	ingressBus "github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/bus"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/ingress"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/intents"
	"github.com/Hafiz-Muhammad-Umar12/ForgeOS/core/store"
)

func main() {
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

	ctx := context.Background()
	provider := gwAuth.NewJWTAdapter([]byte(jwtSecret))

	// Set up persistence: PostgreSQL when a DSN is provided, else in-memory.
	var db store.Store
	dsn := os.Getenv("DEVOS_DATABASE_URL")
	if dsn != "" {
		pgCfg := store.DefaultConfig()
		pgCfg.DSN = dsn
		pg := store.NewPGStore(pgCfg)
		if err := pg.Start(ctx); err != nil {
			log.Fatalf("connect to database: %v", err)
		}
		defer pg.Close(ctx)

		migrator := store.NewMigrator(pg, intents.Migrations())
		if err := migrator.Run(ctx); err != nil {
			log.Fatalf("run migrations: %v", err)
		}
		db = pg
		log.Printf("gateway: connected to PostgreSQL")
	} else {
		db = store.NewFakeStore()
		log.Printf("gateway: DEVOS_DATABASE_URL not set; using in-memory store")
	}

	// Build the intents service (Handler → Service → Repository → Store).
	intentsRepo := intents.NewRepository(db)
	intentsSvc := intents.NewService(intentsRepo)

	// Connect to the bus for the ingress.
	bus := ingressBus.NewNatsBus(ingressBus.WithNatsURL(natsURL))
	if err := bus.Connect(ctx); err != nil {
		log.Fatalf("connect to bus: %v", err)
	}
	defer bus.Close(ctx)

	ing := ingress.NewRESTAdapter(ingress.WithIngressBus(bus))

	cfg := gw.DefaultGatewayConfig()
	cfg.ListenAddr = listenAddr

	gateway := gw.NewGateway(cfg, provider, ing, intentsSvc)

	if err := gateway.Init(ctx); err != nil {
		log.Fatalf("init: %v", err)
	}
	if err := gateway.Start(ctx); err != nil {
		log.Fatalf("start: %v", err)
	}

	fmt.Printf("gateway: listening on %s\n", cfg.ListenAddr)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\ngateway: shutting down...")
	if err := gateway.Stop(ctx); err != nil {
		log.Printf("gateway: stop error: %v", err)
	}
}
