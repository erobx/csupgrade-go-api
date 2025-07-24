package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/erobx/csupgrade-go-api/internal/config"
	"github.com/erobx/csupgrade-go-api/internal/db"
	"github.com/erobx/csupgrade-go-api/internal/rest"
	"github.com/erobx/csupgrade-go-api/internal/server"
	"github.com/erobx/csupgrade-go-api/internal/tradeup"
	"github.com/erobx/csupgrade-go-api/internal/ws"
)

func main() {
	ctx := context.Background()

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// DBs
	postgres, err := db.InitPostgres(ctx, cfg)
	if err != nil {
		panic(err)
	}

	valkey, err := db.InitValkey(ctx, cfg)
	if err != nil {
		panic(err)
	}

	// Stores
	defaultStore := db.NewDefaultStore(postgres, valkey)

	// Services
	tradeupService := tradeup.NewService(defaultStore)
	wsHub := ws.NewHub()

	go wsHub.Run()

	// Handlers
	tradeupHandler := rest.NewTradeupHandler(tradeupService)
	wsHandler := ws.NewHandler(wsHub)

	server, err := server.NewServer(ctx, tradeupHandler, wsHandler)
	if err != nil {
		log.Fatal(err)
	}

	server.SetupRoutes()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("Gracefull shutdown...")
		if err := server.Shutdown(); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
		os.Exit(0)
	}()
	
	if err := server.Start(":8080"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func generate() {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Println("Error generating RSA key:", err)
		return
	}

	// Encode private key to PEM format
	privBytes := x509.MarshalPKCS1PrivateKey(priv)
	privPem := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privBytes,
	}

	// Encode to string
	privPemString := string(pem.EncodeToMemory(privPem))
	file, err := os.Create(".env")
	if err != nil {
		fmt.Println("Error creating .env file:", err)
		return
	}
	defer file.Close()

	// Write the private key string to the file
	_, err = file.WriteString("RSA_PRIVATE_KEY=" + privPemString + "\n")
	if err != nil {
		fmt.Println("Error writing to .env file:", err)
		return
	}
}
