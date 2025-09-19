.PHONY: test
test:
	go test -v -race -coverpkg=./... -coverprofile=coverage.out ./...

.PHONY: cover
cover: test
	go tool cover -func coverage.out

.PHONY: lint
lint:
	golangci-lint run -c golangci.yaml
