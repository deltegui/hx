.PHONY: build serve check

build:
	GOOS=js GOARCH=wasm go build -o static/main.wasm ./client
	cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" static/

serve: build
	go run ./server

# Needs node 22+ and chromium.
check: build
	node check.mjs
