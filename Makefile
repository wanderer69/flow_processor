
all: protoc

protoc:
	@echo "Generating Go files"
	cd api/client && protoc --go_out=../../pkg/proto/client --go-grpc_out=../../pkg/proto/client --go-grpc_opt=paths=source_relative --go_opt=paths=source_relative *.proto && cd ..
	cd api/front && protoc --go_out=../../pkg/proto/front --go-grpc_out=../../pkg/proto/front --go-grpc_opt=paths=source_relative --go_opt=paths=source_relative *.proto

.PHONY: protoc
