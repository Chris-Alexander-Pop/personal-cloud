.PHONY: build-pc install-pc test-pc

build-pc:
	cd cli && go build -o ../bin/pc ./cmd/pc

install-pc: build-pc
	install -m 755 bin/pc $(HOME)/.local/bin/pc

test-pc:
	cd cli && go test ./...
