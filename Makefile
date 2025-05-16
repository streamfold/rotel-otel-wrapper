.PHONY: build

build:
	mkdir -p dist && \
	go build -o dist/rotel-otel-wrapper ./cmd/rotel-otel-wrapper
