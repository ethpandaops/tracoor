# Local build and test. CGO_ENABLED=0 matches the Dockerfile, goreleaser and CI: nothing
# here needs cgo, and with it on, ethkit and go-ethereum each compile their own copy of
# libsecp256k1, so linking any binary fails with "multiple definition of secp256k1_*" on
# toolchains whose binutils does not tolerate it.
export CGO_ENABLED = 0

.PHONY: build test lint run proto build-web

build:
	go build ./...

test:
	go test ./...

lint:
	golangci-lint run ./...

# Local dev run. Single mode boots the server and its agents in one process, which is the
# whole stack short of a real deployment. The config carries node credentials, so it is
# gitignored rather than shipped: copy example_single_config.yaml to single.yaml and point
# it at your nodes, or override with `make run CONFIG=yours.yaml`. The server serves the
# frontend from web/build/frontend, so run `make build-web` once before the first run.
CONFIG ?= single.yaml

run:
	go run . single --single-config $(CONFIG)

# buf takes the module itself as the input and narrows it with --path; a module subdirectory
# has not been a valid input since buf 1.5x. Generated paths are therefore module-relative,
# which is why both templates write to the module root, and why the two gateway artifacts are
# moved to the locations they have always been imported from.
proto:
	buf generate --template buf.gen.yaml --path pkg/proto/tracoor
	buf generate --template buf-api.gen.yaml --path pkg/proto/tracoor/api
	mv pkg/proto/tracoor/api/api.pb.gw.go pkg/api/api.pb.gw.go
	mv pkg/proto/tracoor/api/api.swagger.json pkg/api/openapiv2/api.swagger.json
build-web:
	@echo "Building web frontend..."
	@npm --prefix ./web install && npm --prefix ./web run build
