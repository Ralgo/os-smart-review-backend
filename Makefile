build:
	go build -o bin/os-smart-reviews ./cmd/main.go

build-run:
	go build -o bin/os-smart-reviews ./cmd/main.go 
	source .env && ./bin/os-smart-reviews

run:
	source .env && go run ./cmd/main.go