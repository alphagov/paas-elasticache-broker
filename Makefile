.PHONY: test unit integration

test: unit integration

unit:
	go run github.com/onsi/ginkgo/v2/ginkgo -r --skip-package=ci
integration:
	go run github.com/onsi/ginkgo/v2/ginkgo -timeout=120m -v -r ci/blackbox

generate-fakes:
	go generate ./...
