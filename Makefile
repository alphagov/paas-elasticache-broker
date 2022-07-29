.PHONY: test unit integration

test: unit integration

unit:
	ginkgo -r --skip-package=ci

integration:
	ginkgo -v -r ci/blackbox
