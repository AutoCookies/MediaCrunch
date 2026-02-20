BINARY := ./bin/mediacrunchd

.PHONY: prepare-testdata build test clean

prepare-testdata:
	./testdata/generate_sample.sh

build: prepare-testdata
	@mkdir -p ./bin
	go build -o $(BINARY) ./cmd/mediacrunchd

test: prepare-testdata
	go test ./...

clean:
	rm -rf ./bin ./build/*.webp ./build/test-*.webp ./build/out.webp ./testdata/sample.jpg
