all: inttest

build:
	go build ./cmd/pgfga

debug:
	~/go/bin/dlv debug --headless --listen=:2345 --api-version=2 --accept-multiclient ./cmd/pgfga

run:
	./pgfga

fmt:
	gofmt -w .

test: sec lint

sec:
	gosec ./...
lint:
	golangci-lint run

inttest:
	./docker-compose-tests.sh

.PHONY: install-go-test-coverage
install-go-test-coverage:
	go install github.com/vladopajic/go-test-coverage/v2@latest

.PHONY: check-coverage
check-coverage: install-go-test-coverage
	go test $$(go list ./... | grep -v /e2e) -coverprofile=./cover.out -covermode=atomic -coverpkg=./...
	go-test-coverage --config=./.testcoverage.yml

