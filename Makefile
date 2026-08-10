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
