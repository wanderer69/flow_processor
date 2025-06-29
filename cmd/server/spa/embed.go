package spa

import (
	"embed"
)

// index.html client.wasm wasmexec.js style.css vars.css
//
//go:embed *
var SPA embed.FS
