package main_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"os"
	"os/exec"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/alphagov/paas-elasticache-broker/broker"
	"github.com/alphagov/paas-elasticache-broker/test"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	main "github.com/alphagov/paas-elasticache-broker"
)

var _ = Describe("main", Ordered, func() {

	Describe("Broker command", Ordered, func() {
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
	})

	Describe("broker starts listener", Ordered, func() {
		It("Starts the a listener on http", func() {

			config, err := broker.LoadConfig("./test/fixtures/config.json")
			Expect(err).NotTo(HaveOccurred())
			logger := lager.NewLogger("elasticache-broker")
			b := broker.New(config, nil, logger)

			httpServer, listener, err := main.CreateListener(b, logger, config, "8081")
			Expect(err).NotTo(HaveOccurred())

			go func() {
				httpServer.Serve(*listener)
			}()

			Eventually(func() error {
				_, err := http.Get("http://localhost:8081/healthcheck")
				return err
			}, 10*time.Second, 1*time.Second).Should(Succeed())

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err = httpServer.Shutdown(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Starts the a listener on https", func() {

			certPEM, keyPEM, caPEM, err := test.GenerateTestCert()
			Expect(err).NotTo(HaveOccurred())

			filename, err := test.SetTLSConfigOptions("./test/fixtures/config.json", certPEM, keyPEM)
			Expect(err).NotTo(HaveOccurred())
			defer os.Remove(filename)

			config, err := broker.LoadConfig(filename)
			Expect(err).NotTo(HaveOccurred())
			logger := lager.NewLogger("elasticache-broker")
			b := broker.New(config, nil, logger)

			httpServer, listener, err := main.CreateListener(b, logger, config, "8444")
			Expect(err).NotTo(HaveOccurred())

			go func() {
				httpServer.Serve(*listener)
			}()

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
				_, err := httpClient.Get("https://localhost:8444/healthcheck")
				return err
			}, 10*time.Second, 1*time.Second).Should(Succeed())

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err = httpServer.Shutdown(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

	})
})
