.PHONY: proto-gen proto-clean lint mocks test

proto-gen:
	buf generate
	mv api/openapi.yaml api/ipo-service.openapi.yaml

PROTO_GEN_DIR = api/qubic/aggregation/ipo/v1

proto-clean:
	rm -f $(PROTO_GEN_DIR)/*.pb.go $(PROTO_GEN_DIR)/*.pb.gw.go api/ipo-service.openapi.yaml

proto-lint:
	buf lint

mocks:
	go generate ./domain/ ./clients/

test:
	go test ./...