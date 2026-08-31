.PHONY: run test seed seed-demo recompute tidy web build
run:
	go run ./cmd/server
test:
	go test ./... -count=1
seed:
	go run ./cmd/seed -mode=catalog
seed-demo:
	go run ./cmd/seed -mode=demo -days=60
recompute:
	go run ./cmd/seed -mode=recompute
tidy:
	go mod tidy
web:
	cd ../frontend && npm ci && npm run build
build: web
	CGO_ENABLED=0 go build -o bin/workbench ./cmd/server
