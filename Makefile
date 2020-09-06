GOMAXPROCS:=$(shell nproc)

.PHONY: test
test:
	@mkdir -p tmp
	@go test -v -cover -coverprofile=tmp/cover.out ./...
	@go tool cover -html=tmp/cover.out -o tmp/coverage.html

.PHONY: bench
bench:
	@GOMAXPROCS=$(GOMAXPROCS) go test -bench=. -benchmem ./...
