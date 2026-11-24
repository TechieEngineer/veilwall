.PHONY: setup frontend test build run clean
setup:
	bash scripts/setup.sh
frontend:
	@if [ -f web/package-lock.json ]; then npm --prefix web ci; else npm --prefix web install; fi
	npm --prefix web run build
test:
	go test ./...
	go vet ./...
build: frontend
	mkdir -p bin
	CGO_ENABLED=0 go build -trimpath -o bin/veilwall ./cmd/veilwall
run:
	go run ./cmd/veilwall --demo
clean:
	rm -rf bin web/dist
	mkdir -p web/dist
	touch web/dist/.gitkeep
