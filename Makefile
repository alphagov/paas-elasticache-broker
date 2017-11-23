.PHONY: test unit integration

test: unit integration

unit: $(GOPATH)/bin/ginkgo
	ginkgo -r --skipPackage=ci

integration: $(GOPATH)/bin/ginkgo
	ginkgo -r ci/blackbox

$(GOPATH)/bin/ginkgo:
	cd vendor/github.com/onsi/ginkgo/ginkgo && go install .
