.PHONY: proto-gen proto-clean lint mocks test

proto-gen:
	buf generate
	mv api/openapi.yaml api/general-service.openapi.yaml

PROTO_GEN_DIR = api/qubic/aggregation/general/v1

proto-clean:
	rm -f $(PROTO_GEN_DIR)/*.pb.go $(PROTO_GEN_DIR)/*.pb.gw.go api/general-service.openapi.yaml

proto-lint:
	buf lint

mocks:
	go generate ./domain/ ./clients/

test:
	go test ./...