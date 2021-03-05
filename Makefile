.PHONY: test unit integration

test: unit integration

unit: $(GOPATH)/bin/ginkgo
	ginkgo -r --skipPackage=ci

integration: $(GOPATH)/bin/ginkgo
	ginkgo -v -r ci/blackbox

$(GOPATH)/bin/ginkgo:
	go get -u github.com/onsi/ginkgo/ginkgo@v1.4.0
