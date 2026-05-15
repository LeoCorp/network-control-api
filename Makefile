.PHONY: swagger run build

# Regenerate docs/docs.go, docs/swagger.json, docs/swagger.yaml from annotations.
swagger:
	go run github.com/swaggo/swag/cmd/swag@v1.16.4 init -g cmd/server/main.go -o docs --parseDependency --parseInternal

run:
	go run ./cmd/server

build:
	go build -o bin/server ./cmd/server
