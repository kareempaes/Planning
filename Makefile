.PHONY: build run test docker-up docker-down clean

build:
	go build -o bin/server ./cmd/app

run:
	DB_DRIVER=sqlite DB_DSN=":memory:" go run ./cmd/app

test:
	go test ./... -v -count=1

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down -v

clean:
	rm -rf bin/
