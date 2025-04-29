# build
FROM golang:alpine

WORKDIR /src
COPY . .

RUN go mod download

WORKDIR /src/cmd/server

RUN go build -ldflags="-s -w" -a -buildvcs=false -o /src/bin/flow_processor

# run
FROM debian

WORKDIR /bin

COPY --from=0 /src/bin/flow_processor /bin/flow_processor
#COPY --from=0 /src/configs /app_configs

CMD flow_processor
