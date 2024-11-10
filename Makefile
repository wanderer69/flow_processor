
all: protoc

protoc:
	@echo "Generating Go files"
	cd api/client && protoc --go_out=../../pkg/proto --go-grpc_out=../../pkg/proto --go-grpc_opt=paths=source_relative --go_opt=paths=source_relative *.proto

.PHONY: protoc
