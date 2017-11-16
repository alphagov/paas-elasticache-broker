package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestPaasElasticacheRedisBroker(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PaasElasticacheRedisBroker Suite")
}
