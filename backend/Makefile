.PHONY: run test swagger tidy doctor cli web web-build web-dev

run:
	go run ./cmd/worker

test:
	go test ./...

swagger:
	go run github.com/swaggo/swag/cmd/swag@latest init \
		-g cmd/worker/main.go \
		-o internal/api/docs \
		--parseDependency \
		--parseInternal

tidy:
	go mod tidy

cli:
	go run ./cmd/cli doctor --suite TR0

doctor:
	go run ./cmd/cli doctor --suite TR0 --format md

web-build:
	cd ../frontend && npm install && npm run build

web-dev:
	cd ../frontend && npm install && npm run dev
