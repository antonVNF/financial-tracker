include .env
export

up:
	docker-compose up -d postgres

down:
	docker-compose down

run:
	go run main.go

migrate-up:
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down