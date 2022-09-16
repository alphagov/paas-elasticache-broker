.PHONY: test unit integration

test: unit integration

unit:
	go run github.com/onsi/ginkgo/v2/ginkgo -r --skip-package=ci

integration:
	go run github.com/onsi/ginkgo/v2/ginkgo -v -r ci/blackbox
