package test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/alphagov/paas-elasticache-broker/broker"
)

func SetTLSConfigOptions(configFile string, certficiate []byte, key []byte, ca []byte) (string, error) {

	data, err := os.ReadFile(configFile)
	if err != nil {
		return "", fmt.Errorf("Failed to read config file: %v", err)
	}

	var config broker.Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return "", fmt.Errorf("Failed to unmarshal json file: %v", err)
	}

	config.TLS = &broker.TLSConfig{}
	config.TLS.Certificate = string(certficiate)
	config.TLS.PrivateKey = string(key)
	config.TLS.CA = string(ca)

	data, err = json.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("Failed to marshal json file: %v", err)
	}

	file, err := os.CreateTemp("", "config")
	if err != nil {
		return "", fmt.Errorf("Failed to create temp file: %v", err)
	}
	defer file.Close()

	_, err = file.Write(data)
	if err != nil {
		return "", fmt.Errorf("Failed to write to temp file: %v", err)
	}

	return file.Name(), nil
}

func GenerateTestCert() ([]byte, []byte, []byte, error) {
	// Generate a new RSA private key for the root CA
	rootKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Failed to generate RSA key for root CA: %v", err)
	}

	// Generate a new self-signed X.509 certificate for the root CA
	rootTemplate := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "Root CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(100, 0, 0),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	rootCertDER, err := x509.CreateCertificate(rand.Reader, &rootTemplate, &rootTemplate, &rootKey.PublicKey, rootKey)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Failed to create X.509 certificate for root CA: %v", err)
	}

	// Generate a new RSA private key for the intermediate CA
	intermediateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Failed to generate RSA key for intermediate CA: %v", err)
	}

	// Generate a new self-signed X.509 certificate for the intermediate CA
	intermediateTemplate := x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			CommonName: "Intermediate CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(100, 0, 0),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	intermediateCertDER, err := x509.CreateCertificate(rand.Reader, &intermediateTemplate, &rootTemplate, &intermediateKey.PublicKey, rootKey)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Failed to create X.509 certificate for intermediate CA: %v", err)
	}

	// Generate a new RSA private key for the certificate
	certKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Failed to generate RSA key for certificate: %v", err)
	}

	// Generate a new X.509 certificate signed by the intermediate CA
	template := x509.Certificate{
		SerialNumber: big.NewInt(3),
		Subject: pkix.Name{
			CommonName: "example.com",
		},
		DNSNames:              []string{"example.com"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(100, 0, 0),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &intermediateTemplate, &certKey.PublicKey, intermediateKey)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Failed to create X.509 certificate signed by intermediate CA: %v", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	intermediatePEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: intermediateCertDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(certKey)})
	rootCAPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: rootCertDER})
	certChainPEM := append(certPEM, intermediatePEM...)

	return certChainPEM, keyPEM, rootCAPEM, nil
}
