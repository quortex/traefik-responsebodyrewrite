## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
GOLANG_CI_LINT?= $(LOCALBIN)/golangci-lint
YAEGI?= $(LOCALBIN)/yaegi

## Tool Versions
GOLANG_CI_LINT_VERSION ?= v1.58.1
YAEGI_VERSION ?= v0.16.1

.PHONY: golangci-lint
golangci-lint: $(GOLANG_CI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANG_CI_LINT): $(LOCALBIN)
	test -s $(LOCALBIN)/golangci-lint || GOBIN=$(LOCALBIN) go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANG_CI_LINT_VERSION)
.PHONY: yaegi
yaegi: $(YAEGI) ## Download yaegi locally if necessary.
$(YAEGI): $(LOCALBIN)
	test -s $(LOCALBIN)/yaegi || GOBIN=$(LOCALBIN) go install github.com/traefik/yaegi/cmd/yaegi@$(YAEGI_VERSION)

.PHONY: lint test vendor clean

export GO111MODULE=on

default: lint test

lint: golangci-lint
	$(GOLANG_CI_LINT) run

test:
	go test -v -cover ./...

yaegi_test: yaegi
	$(YAEGI) test -v .

vendor:
	go mod vendor

clean:
	rm -rf ./vendor

