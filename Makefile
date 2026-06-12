COVER_PKGS := ./internal/app/models/identity ./internal/app/models/projects ./internal/app/services/identity ./internal/app/services/projects ./internal/config ./internal/env ./internal/httpapi ./internal/phylax ./internal/store

.PHONY: build coverage fmt migrate-down migrate-up seed run swagger test

build:
	go build -o bin/phylax ./cmd/phylax
	go build -o bin/phylax-migrate ./cmd/phylax-migrate
	go build -o bin/phylax-seed ./cmd/phylax-seed

fmt:
	gofmt -w cmd internal

swagger:
	go run github.com/swaggo/swag/cmd/swag init -g main.go -d cmd/phylax,internal/httpapi,internal/phylax,internal/app/controllers,internal/app/types/dto/identity,internal/app/types/dto/projects,internal/app/validators/identity,internal/app/validators/projects,internal/app/types/enum -o docs --parseInternal

coverage:
	go test $(COVER_PKGS) -coverprofile=coverage.out
	go tool cover -func=coverage.out

test:
	go test ./...

run:
	PHYLAX_ENV=development go run ./cmd/phylax

migrate-up:
	go run ./cmd/phylax-migrate -direction up

migrate-down:
	go run ./cmd/phylax-migrate -direction down

seed:
	go run ./cmd/phylax-seed
