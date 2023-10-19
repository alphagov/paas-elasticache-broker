package main_test

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/alphagov/paas-elasticache-broker/test"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("broker command", Ordered, func() {
	var (
		path    string
		command string
		err     error
	)

	BeforeAll(func() {
		path, err = gexec.Build(".")
		Expect(err).NotTo(HaveOccurred())
		command = path + "/paas-elasticache-broker"
	})

	AfterAll(func() {
		gexec.CleanupBuildArtifacts()
	})

	It("exits nonzero when given a path to a nonexistent config file", func() {
		cmd := exec.Command(command, "-config", "anonexistentconfigfile")
		session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())

		session.Wait()
		Expect(session).To(gexec.Exit(1))
	})

	Describe("when given a valid config file", func() {

		It("starts the http server", func() {
			cmd := exec.Command(command, "-config", "./test/fixtures/config.json", "-port", "8080")
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			// Wait for the server to start
			Eventually(func() error {
				_, err := http.Get("http://localhost:8080/healthcheck")
				return err
			}, 10*time.Second, 1*time.Second).Should(Succeed())

			// Stop the server
			session.Interrupt()
			Eventually(session).Should(gexec.Exit())

			// Check the exit code
			Expect(session.ExitCode()).To(Equal(130))
		})

		It("starts the https server", func() {
			certPEM, keyPEM, caPEM, err := test.GenerateTestCert()
			Expect(err).NotTo(HaveOccurred())

			filename, err := test.SetTLSConfigOptions("./test/fixtures/config.json", certPEM, keyPEM)
			Expect(err).NotTo(HaveOccurred())
			defer os.Remove(filename)

			cmd := exec.Command(command, "-config", filename, "-port", "8080")
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())

			certPool := x509.NewCertPool()
			certPool.AppendCertsFromPEM([]byte(caPEM))

			// Create a custom transport with the certificate pool
			transport := &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs:    certPool,
					ServerName: "example.com",
				},
			}

			// Create a http client with the custom transport
			httpClient := &http.Client{
				Transport: transport,
			}

			// Wait for the server to start
			Eventually(func() error {
				_, err := httpClient.Get("https://localhost:8080/healthcheck")
				return err
			}, 10*time.Second, 1*time.Second).Should(Succeed())

			// Stop the server
			session.Interrupt()
			Eventually(session).Should(gexec.Exit())

			// Check the exit code
			Expect(session.ExitCode()).To(Equal(130))
		})
	})
})
