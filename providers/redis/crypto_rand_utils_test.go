package redis_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/alphagov/paas-elasticache-broker/providers/redis"
)

var _ = Describe("RandomAlphaNum", func() {
	It("generates a random alpha numeric with the proper length", func() {
		randomString := RandomAlphaNum(32)
		Expect(len(randomString)).To(Equal(32))
	})

	It("generates a large number of different passwords", func() {
		numberOfValues := 100000
		lengthOfValues := 32

		observedValues := map[string]bool{}
		for i := 0; i < numberOfValues; i++ {
			value := RandomAlphaNum(lengthOfValues)
			Expect(observedValues[value]).To(BeFalse())
			observedValues[value] = true
		}
	})
})
