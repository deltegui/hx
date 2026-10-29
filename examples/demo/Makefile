.PHONY: build serve

build:
	GOOS=js GOARCH=wasm go build -o main.wasm .
	cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" .

serve: build
	@echo "http://localhost:8080"
	python3 -m http.server 8080
