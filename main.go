package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"os"

	"github.com/erobx/csupgrade/backend/internal/app"
	"github.com/erobx/csupgrade/backend/pkg/api"
	"github.com/erobx/csupgrade/backend/pkg/db"
	"github.com/erobx/csupgrade/backend/pkg/repository"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalln(err)
	}

	db, err := db.CreateConnection()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	winnings := make(chan api.Winnings, 1)

	cdnUrl := os.Getenv("SKINS_CDN_URL")
	storage := repository.NewStorage(db, cdnUrl)
	logService := api.NewLogger()
	userService := api.NewUserService(storage, logService)
	storeService := api.NewStoreService(storage, logService)
	tradeupService := api.NewTradeupService(storage, winnings, logService)

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(os.Getenv("RSA_PRIVATE_KEY")))
	if err != nil {
		log.Fatalln(err)
	}

	server := app.NewServer("8080", privateKey, logService, userService, storeService, tradeupService, winnings)
	server.Run()
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
