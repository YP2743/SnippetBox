run:
	go run ./cmd/web/

build-test:
	go build -gcflags=all="-N -l" ./cmd/web/

test:
	go test -v ./cmd/web/