.PHONY: test test-race fmt vet

test:
	./scripts/test.sh

test-race:
	RACE=1 ./scripts/test.sh

fmt:
	go fmt ./...

vet:
	go vet ./...

