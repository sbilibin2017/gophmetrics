# Paths
src := api/grpc
dst := pkg/grpc

# Generate Go code from .proto files
gen-proto:
	@mkdir -p $(dst)
	@for protofile in $(wildcard $(src)/*.proto); do \
		echo "Generating $$protofile into $(dst)"; \
		protoc \
			--proto_path=$(src) \
			--go_out=$(dst) --go_opt=paths=source_relative \
			--go-grpc_out=$(dst) --go-grpc_opt=paths=source_relative \
			$$protofile; \
	done

# Generate mocks
gen-mock:
	mockgen -source=$(file) \
		-destination=$(dir $(file))/$(notdir $(basename $(file)))_mock.go \
		-package=$(shell basename $(dir $(file)))

# Run tests and generate coverage profile, filtered for internal packages only
test:
	go test -coverprofile=coverage.out ./...
	rm coverage.out

# Generate swagger docs from handlers in internal/handlers,
# use cmd/server/main.go as entry point for swag init,
# output to api/http folder
gen-swag:
	swag init -g cmd/server/main.go -o api/http