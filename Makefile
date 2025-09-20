.PHONY: test
test:
	go test -v -race -coverpkg=./... -coverprofile=coverage.out.tmp ./...
	cat coverage.out.tmp | grep -v "examples" > coverage.out && rm coverage.out.tmp

.PHONY: cover
cover: test
	go tool cover -func coverage.out

.PHONY: lint
lint:
	golangci-lint run -c golangci.yaml
