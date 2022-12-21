.PHONY: test unit integration

test: unit integration

unit:
	go run github.com/onsi/ginkgo/v2/ginkgo -r --skip-package=ci
integration:
	go run github.com/onsi/ginkgo/v2/ginkgo -timeout=120m -v -r ci/blackbox

generate-fakes:
	go generate ./...

.PHONY: build_amd64
build_amd64:
	mkdir -p amd64
	GOOS=linux GOARCH=amd64 go build -o amd64/elasticache-broker

.PHONY: bosh_scp
bosh_scp: build_amd64
	./scripts/bosh-scp.sh
