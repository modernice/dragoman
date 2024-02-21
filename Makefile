.PHONY: docs
docs:
	@./docs.sh

.PHONY: test
test:
	go test ./...
