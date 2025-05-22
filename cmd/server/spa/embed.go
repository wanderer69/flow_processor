package spa

import (
	"embed"
)

//go:embed index.html wasm_exec.js test.wasm wasm_exec.js
var SPA embed.FS
