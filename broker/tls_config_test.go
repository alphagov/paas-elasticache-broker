package broker_test

import (
	"crypto/tls"
	"crypto/x509"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/alphagov/paas-elasticache-broker/broker"
	. "github.com/alphagov/paas-elasticache-broker/test"
)

var _ = Describe("TLSConfig", Ordered, func() {
	var (
		certPEM         []byte
		keyPEM          []byte
		tlsConfig       *TLSConfig
		generatedConfig *tls.Config
		err             error
	)

	BeforeAll(func() {
		// Generate test certificates and keys
		certPEM, keyPEM, _, err = GenerateTestCert()
		Expect(err).NotTo(HaveOccurred())
	})

	BeforeEach(func() {

		tlsConfig = &TLSConfig{
			Certificate: string(certPEM),
			PrivateKey:  string(keyPEM),
		}
	})

	Describe("GenerateTLSConfig", func() {
		BeforeEach(func() {
			// Generate a tls.Config structure out of the TLSConfig instance
			generatedConfig, err = tlsConfig.GenerateTLSConfig()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should generate a valid tls.Config structure", func() {
			Expect(len(generatedConfig.Certificates)).To(Equal(1))
			Expect(generatedConfig.MinVersion).To(Equal(uint16(tls.VersionTLS12)))
			Expect(generatedConfig.CipherSuites).To(Equal([]uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			}))

			cert, err := x509.ParseCertificate(generatedConfig.Certificates[0].Certificate[0])
			Expect(err).NotTo(HaveOccurred())
			Expect(cert.Subject.CommonName).To(Equal("example.com"))

			intermediateCert, err := x509.ParseCertificate(generatedConfig.Certificates[0].Certificate[1])
			Expect(err).NotTo(HaveOccurred())
			Expect(intermediateCert.Subject.CommonName).To(Equal("Intermediate CA"))
		})
	})

	Describe("validate", func() {
		It("should return an error if Certificate is empty", func() {
			tlsConfig.Certificate = ""
			err = tlsConfig.Validate()
			Expect(err).To(MatchError("Config error: TLS certificate required"))
		})

		It("should return an error if PrivateKey is empty", func() {
			tlsConfig.PrivateKey = ""
			err = tlsConfig.Validate()
			Expect(err).To(MatchError("Config error: TLS private key required"))
		})

		It("should return an error if Certificate is invalid", func() {
			tlsConfig.Certificate = "invalid"
			err = tlsConfig.Validate()
			Expect(err).To(MatchError("Invalid Certificate and key: tls: failed to find any PEM data in certificate input"))
		})

		It("should not return an error if all fields are present", func() {
			err = tlsConfig.Validate()
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
