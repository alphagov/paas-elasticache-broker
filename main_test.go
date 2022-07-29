package main_test

import (
	"os/exec"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("broker command", Ordered, func() {
	AfterAll(func() {
		gexec.CleanupBuildArtifacts()
	})

	It("exits nonzero when given a path to a nonexistent config file", func() {
		path, err := gexec.Build("github.com/alphagov/paas-elasticache-broker")
		Expect(err).NotTo(HaveOccurred())

		cmd := exec.Command(path, "-config", "anonexistentconfigfile")
		session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())

		session.Wait()
		Expect(session).To(gexec.Exit(1))
	})
})
