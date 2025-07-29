package main

import (
	db "backend/database"
	logdb "backend/logdatabase"
	m "backend/middleware"
	"backend/routes"
	"backend/storage"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
)

func main() {
	r := chi.NewRouter()
	r.Use(m.DBRequestLogger, m.RequestLogger)
	r.Mount("/api/user", routes.InitUser())
	r.Mount("/api/session", routes.InitSession())
	r.Mount("/api/admin", routes.InitAdmin())
	r.Mount("/api/repository", routes.InitRepository())
	r.Mount("/api/file", routes.InitFile())

	p := http.Protocols{}
	p.SetHTTP1(true)
	p.SetHTTP2(true)

	// Generate a private key for TLS connections.
	c := []tls.Certificate{}
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Println("Cannot generate RSA key")
		log.Fatalln(err)
	}
	c = append(c, tls.Certificate{PrivateKey: privateKey})

	server := http.Server{
		Addr:    os.Getenv("SERVER_HOST"),
		Handler: r,
		// Max size is 2^13 Bytes being about 8 kB.
		MaxHeaderBytes: 1 << 13,
		Protocols:      &p,
		// Close the keep-alive connection after receiving no requests for some time.
		IdleTimeout:  time.Minute * 3,
		ReadTimeout:  time.Second * 10,
		WriteTimeout: time.Second * 10,
		TLSConfig:    &tls.Config{Certificates: c},
	}
	storage.InitStorage("aws")
	db.InitDB()
	// This is optional, you can disable logging by removing this line.
	logdb.InitDB()
	fmt.Println("Connected to DB, starting server")
	fmt.Println(server.ListenAndServeTLS("util/cert.pem", "util/key.pem"))
}
