proto:
	buf generate --template buf.gen.yaml ./pkg/proto/tracoor
	buf generate --template buf-api.gen.yaml ./pkg/proto/tracoor/api