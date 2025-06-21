package test

import (
	"crypto/x509"
	"log"
	"os"
)

var (
	serverHost = "localhost" + os.Getenv("SERVER_HOST")
)

const (
	localCertFile = "../util/cert.pem"
)

func loadCerts() (*x509.CertPool, error) {
	// Get the SystemCertPool, continue with an empty pool on error.
	rootCAs, _ := x509.SystemCertPool()
	if rootCAs == nil {
		rootCAs = x509.NewCertPool()
	}

	// Read in the cert file.
	certs, err := os.ReadFile(localCertFile)
	if err != nil {
		return rootCAs, err
	}

	// Append our cert to the system pool.
	if ok := rootCAs.AppendCertsFromPEM(certs); !ok {
		log.Println("No certs appended, using system certs only")
	}
	return rootCAs, nil
}
