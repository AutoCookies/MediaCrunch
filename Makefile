BINARY := ./bin/mediacrunchd

.PHONY: prepare-testdata fmt fmt-check lint build test ci clean

prepare-testdata:
	./testdata/generate_sample.sh

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './build/*')

fmt-check:
	@test -z "$$(gofmt -l $$(find . -name '*.go' -not -path './build/*'))" || (echo 'Go files are not formatted'; gofmt -l $$(find . -name '*.go' -not -path './build/*'); exit 1)

lint:
	go vet ./...

build: prepare-testdata
	@mkdir -p ./bin
	go build -o $(BINARY) ./cmd/mediacrunchd

test: prepare-testdata
	go test ./...

ci: fmt-check lint test build

clean:
	rm -rf ./bin ./build/*.webp ./build/test-*.webp ./build/out.webp ./testdata/sample.jpg
