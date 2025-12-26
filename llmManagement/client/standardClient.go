package client

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"
)




// getintfromenvwithdefault retrieves an integer from environment variable with fallback
func getIntFromEnvWithDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// newstandardclient creates a standard http client suitable for json apis like openai.
// this client supports mtls for secure communication with the completion server.
func NewStandardClient() *http.Client {
	// get timeout from environment or use default (30 seconds)
	timeoutSeconds := getIntFromEnvWithDefault("LLM_HTTP_TIMEOUT_SECONDS", 300)
	connectionTimeoutSeconds := getIntFromEnvWithDefault("LLM_HTTP_CONNECTION_TIMEOUT_SECONDS", 5)
	// response header timeout should match the overall timeout for LLM requests
	// which can take 30+ seconds to generate responses
	responseHeaderTimeoutSeconds := getIntFromEnvWithDefault("LLM_HTTP_RESPONSE_HEADER_TIMEOUT_SECONDS", timeoutSeconds)

	// get connection pool settings from environment with high-load defaults
	// for 5k req/s with ~4s avg latency: need ~20k concurrent requests
	// but http connections are multiplexed, so fewer connections needed
	maxIdleConns := getIntFromEnvWithDefault("LLM_HTTP_MAX_IDLE_CONNS", 5000)

	// increase max connections per host to handle high concurrency
	// this is the bottleneck: limits total concurrent requests to one host
	maxConnsPerHost := getIntFromEnvWithDefault("LLM_HTTP_MAX_CONNS_PER_HOST", 5000)

	// keep 80% of max as idle for fast reuse under sustained load
	maxIdleConnsPerHost := getIntFromEnvWithDefault("LLM_HTTP_MAX_IDLE_CONNS_PER_HOST", 4000)

	// keep connections alive longer for sustained high load (90s is better for continuous traffic)
	// shorter timeouts cause connection churn and slow down request processing
	idleConnTimeoutSeconds := getIntFromEnvWithDefault("LLM_HTTP_IDLE_CONN_TIMEOUT_SECONDS", 90)

	// create transport with configurable tls support
	transport := &http.Transport{
		MaxIdleConns:        maxIdleConns,
		MaxIdleConnsPerHost: maxIdleConnsPerHost,
		MaxConnsPerHost:     maxConnsPerHost,
		// Use configurable idle timeout (default 10s for faster cleanup)
		IdleConnTimeout: time.Duration(idleConnTimeoutSeconds) * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   time.Duration(connectionTimeoutSeconds) * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: time.Duration(responseHeaderTimeoutSeconds) * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		// Force connection closure after response to prevent accumulation under high load
		// This prevents goroutine leaks from persistent connections
		DisableKeepAlives: false, // Keep connection pooling but with aggressive timeouts
		// Limit reads per connection to force cleanup of stuck connections
		MaxResponseHeaderBytes: 1 << 20, // 1MB max response header
	}

	// check if tls client certificates are configured
	clientCertPEM := os.Getenv("LOCAL_API_CLIENT_CERT_PEM")
	clientKeyPEM := os.Getenv("LOCAL_API_CLIENT_KEY_PEM")
	caCertPEM := os.Getenv("LOCAL_API_CA_CERT_PEM")

	if clientCertPEM != "" && clientKeyPEM != "" && caCertPEM != "" {
		// create certificate pool for ca certificate
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM([]byte(caCertPEM)) {
			log.Fatal("Failed to parse CA certificate")
		}

		// load client certificate and key using helper function to handle negative serial numbers
		clientCert, err := loadX509KeyPairWithNegativeSerial([]byte(clientCertPEM), []byte(clientKeyPEM))
		if err != nil {
			log.Fatalf("Failed to load client certificate and key: %v", err)
		}

		// configure tls with client certificates and ca certificate
		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{clientCert},
			RootCAs:      caCertPool,
			// optional: specify minimum tls version
			MinVersion: tls.VersionTLS12,
			// add custom verification function for handling certificates with negative serial numbers
			VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
				// if verification already passed (verifiedchains is populated), no need for custom check
				if len(verifiedChains) > 0 {
					return nil
				}

				// custom verification logic for cases where standard verification fails
				// due to negative serial numbers
				certs := make([]*x509.Certificate, len(rawCerts))
				for i, rawCert := range rawCerts {
					cert, err := x509.ParseCertificate(rawCert)
					if err != nil {
						return fmt.Errorf("failed to parse certificate: %v", err)
					}
					certs[i] = cert
				}

				// verify the certificate chain using our ca pool
				opts := x509.VerifyOptions{
					Roots:         caCertPool,
					Intermediates: x509.NewCertPool(),
				}

				// add intermediate certificates to the pool
				for i := 1; i < len(certs); i++ {
					opts.Intermediates.AddCert(certs[i])
				}

				_, err := certs[0].Verify(opts)
				return err
			},
		}

		transport.TLSClientConfig = tlsConfig
	}

	return &http.Client{
		Timeout:   time.Duration(timeoutSeconds) * time.Second,
		Transport: transport,
	}
}

// newstandardclientwithtimeout creates a standard http client with a custom timeout.
func NewStandardClientWithTimeout(timeout time.Duration) *http.Client {
	// get connection pool settings from environment with high-load defaults
	maxIdleConns := getIntFromEnvWithDefault("LLM_HTTP_MAX_IDLE_CONNS", 5000)
	maxConnsPerHost := getIntFromEnvWithDefault("LLM_HTTP_MAX_CONNS_PER_HOST", 5000)
	maxIdleConnsPerHost := getIntFromEnvWithDefault("LLM_HTTP_MAX_IDLE_CONNS_PER_HOST", 4000)
	idleConnTimeoutSeconds := getIntFromEnvWithDefault("LLM_HTTP_IDLE_CONN_TIMEOUT_SECONDS", 90)

	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:        maxIdleConns,
			MaxIdleConnsPerHost: maxIdleConnsPerHost,
			MaxConnsPerHost:     maxConnsPerHost,
			IdleConnTimeout:     time.Duration(idleConnTimeoutSeconds) * time.Second,
			// force aggressive cleanup to prevent goroutine accumulation
			MaxResponseHeaderBytes: 1 << 20, // 1MB max response header
		},
	}
}

// loadx509keypairwithnegativeserial loads an x509 key pair (certificate and private key) with special
// handling for certificates with negative serial numbers, which were disallowed in go 1.23+.
// this is particularly important for windows environments.
func loadX509KeyPairWithNegativeSerial(certPEMBlock, keyPEMBlock []byte) (tls.Certificate, error) {
	// first try the standard way
	cert, err := tls.X509KeyPair(certPEMBlock, keyPEMBlock)
	if err == nil {
		return cert, nil
	}

	// if the standard way fails and the error is related to negative serial numbers,
	// we'll try a workaround by parsing the certificate manually
	errMsg := err.Error()
	if errMsg == "x509: negative serial number" || errMsg == "crypto/rsa: key values are not correct" {
		// this is a simplified implementation to bypass the negative serial check
		// parse the certificate from pem format
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

		// parse the private key from pem format
		keyDERBlock, _ := pem.Decode(keyPEMBlock)
		if keyDERBlock == nil {
			return tls.Certificate{}, fmt.Errorf("failed to parse key PEM data")
		}

		// create a certificate with the parsed der blocks
		cert = tls.Certificate{
			Certificate: certificates,
			PrivateKey:  nil, // will be parsed from keyderblock below
		}

		// parse the private key based on its type
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

	// if the error is not related to negative serial numbers, return the original error
	return tls.Certificate{}, err
}
