#!/bin/bash
set -eu

GOARCH=wasm GOOS=js go build -ldflags="-w -s -buildid=" -trimpath -o "../server/spa/client.wasm" main.go
#ln -fs /home/wanderer/go/misc/wasm/wasm_exec.js ../server/spa/wasmexec.js
cp /home/wanderer/go/misc/wasm/wasm_exec.js ../server/spa/wasmexec.js
#go run /home/wanderer/go/src/crypto/tls/generate_cert.go --host localhost
#go build
