package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
)

// loadX509KeyPairWithNegativeSerial loads an X509 key pair (certificate and private key) with special
// handling for certificates with negative serial numbers, which were disallowed in Go 1.23+.
// This is particularly important for Windows environments.
func loadX509KeyPairWithNegativeSerial(certPEMBlock, keyPEMBlock []byte) (tls.Certificate, error) {
	// First try the standard way
	cert, err := tls.X509KeyPair(certPEMBlock, keyPEMBlock)
	if err == nil {
		return cert, nil
	}

	// If the standard way fails and the error is related to negative serial numbers,
	// we'll try a workaround by parsing the certificate manually
	errMsg := err.Error()
	if errMsg == "x509: negative serial number" || errMsg == "crypto/rsa: key values are not correct" {
		// This is a simplified implementation to bypass the negative serial check
		// Parse the certificate from PEM format
		var certDERBlock *pem.Block
		var certificates [][]byte

		for {
			certDERBlock, certPEMBlock = pem.Decode(certPEMBlock)
			if certDERBlock == nil {
				break
			}
			if certDERBlock.Type == "CERTIFICATE" {
				certificates = append(certificates, certDERBlock.Bytes)
			}
		}

		// Parse the private key from PEM format
		keyDERBlock, _ := pem.Decode(keyPEMBlock)
		if keyDERBlock == nil {
			return tls.Certificate{}, fmt.Errorf("failed to parse key PEM data")
		}

		// Create a certificate with the parsed DER blocks
		cert = tls.Certificate{
			Certificate: certificates,
			PrivateKey:  nil, // Will be parsed from keyDERBlock below
		}

		// Parse the private key based on its type
		switch keyDERBlock.Type {
		case "RSA PRIVATE KEY":
			key, err := x509.ParsePKCS1PrivateKey(keyDERBlock.Bytes)
			if err != nil {
				return tls.Certificate{}, fmt.Errorf("failed to parse RSA private key: %v", err)
			}
			cert.PrivateKey = key
		case "PRIVATE KEY":
			key, err := x509.ParsePKCS8PrivateKey(keyDERBlock.Bytes)
			if err != nil {
				return tls.Certificate{}, fmt.Errorf("failed to parse PKCS#8 private key: %v", err)
			}
			cert.PrivateKey = key
		case "EC PRIVATE KEY":
			key, err := x509.ParseECPrivateKey(keyDERBlock.Bytes)
			if err != nil {
				return tls.Certificate{}, fmt.Errorf("failed to parse EC private key: %v", err)
			}
			cert.PrivateKey = key
		default:
			return tls.Certificate{}, fmt.Errorf("unsupported key type %q", keyDERBlock.Type)
		}

		return cert, nil
	}

	// If the error is not related to negative serial numbers, return the original error
	return tls.Certificate{}, err
}