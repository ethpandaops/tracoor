proto:
	buf generate --template buf.gen.yaml ./pkg/proto/tracoor
	buf generate --template buf-api.gen.yaml ./pkg/proto/tracoor/api
build-web:
	@echo "Building web frontend..."
	@npm --prefix ./web install && npm --prefix ./web run build
