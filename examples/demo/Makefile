.PHONY: build serve release

build:
	GOOS=js GOARCH=wasm go build -o main.wasm .
	cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" .

serve: build
	@echo "http://localhost:8080"
	python3 -m http.server 8080

# Optimized TinyGo build for GitHub Pages. OUT is the folder that gets published.
# main.wasm is shipped as main.wasm.gz and unpacked in the page with pako.
release:
	test -n "$(OUT)" || { echo "usage: make release OUT=dir"; exit 1; }
	GOOS=js GOARCH=wasm tinygo build --no-debug -opt=z -o "$(OUT)/main.wasm" .
	gzip -9 -f -k "$(OUT)/main.wasm"
	rm "$(OUT)/main.wasm"
	cp "$$(tinygo env TINYGOROOT)/targets/wasm_exec.js" style.css logo.svg "$(OUT)/"
