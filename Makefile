BINARY := ./bin/mediacrunchd

.PHONY: prepare-testdata fmt fmt-check lint build test bench-smoke ci clean

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

bench-smoke: prepare-testdata
	go test ./... -run TestNonExistent -bench BenchmarkHashSmall -benchtime=50ms

ci: fmt-check lint test build bench-smoke

clean:
	rm -rf ./bin ./build/* ./testdata/sample.jpg
