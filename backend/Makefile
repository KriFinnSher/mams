.PHONY: unit lint ci

unit:
	go test ./...

lint:
	go vet ./...

ci: lint unit
