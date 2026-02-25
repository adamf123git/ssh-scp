APP_NAME := ssh-scp
MAIN     := ./cmd/main.go
PREFIX   ?= /usr/local

MIN_COVERAGE ?= 80

.PHONY: build install lint fmt test coverage check-coverage clean

## build: compile the binary
build:
	go build -o ./bin/$(APP_NAME) $(MAIN)

## install: install to PREFIX/bin (default /usr/local/bin)
install: build
	install -d $(PREFIX)/bin
	install -m 755 ./bin/$(APP_NAME) $(PREFIX)/bin/$(APP_NAME)

## lint: run golangci-lint
lint:
	golangci-lint run ./...

## fmt: format all Go source files
fmt:
	go fmt ./...

## test: run all tests
test:
	go test -race ./...

## coverage: run tests with coverage report
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
	@echo ""
	@echo "To view HTML report: go tool cover -html=coverage.out"

## check-coverage: fail if total coverage is below MIN_COVERAGE (default 80%)
check-coverage:
	@go test -coverprofile=coverage.out ./...
	@total=$$(go tool cover -func=coverage.out | grep '^total:' | awk '{print $$NF}' | tr -d '%'); \
	echo "Total coverage: $${total}%"; \
	if [ $$(echo "$${total} < $(MIN_COVERAGE)" | bc -l) -eq 1 ]; then \
		echo "FAIL: coverage $${total}% is below minimum $(MIN_COVERAGE)%"; \
		exit 1; \
	else \
		echo "OK: coverage $${total}% meets minimum $(MIN_COVERAGE)%"; \
	fi

## clean: remove build artifacts
clean:
	rm -f ./bin/$(APP_NAME) coverage.out
