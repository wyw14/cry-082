.PHONY: fmt test race vet run web-test web-build verify

fmt:
	test -z "$$(gofmt -l $$(find . -name '*.go' -not -path './vendor/*'))"

test:
	go test ./...

race:
	go test -race ./...

vet:
	go vet ./...

run:
	go run ./cmd/server

web-test:
	npm --prefix web test -- --run

web-build:
	npm --prefix web run typecheck
	npm --prefix web run build

verify: fmt test race vet web-test web-build
