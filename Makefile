GOROOT := $(shell go env GOROOT)
WASM_EXEC := $(firstword $(wildcard $(GOROOT)/lib/wasm/wasm_exec.js $(GOROOT)/misc/wasm/wasm_exec.js))

.PHONY: run web serve clean

# Gioca in una finestra desktop
run:
	go run .

# Build per il browser in dist/
web:
	mkdir -p dist
	GOOS=js GOARCH=wasm go build -trimpath -ldflags="-s -w" -o dist/game.wasm .
	cp $(WASM_EXEC) dist/
	cp web/index.html dist/

# Build web + server locale su http://localhost:8080
serve: web
	cd dist && python3 -m http.server 8080

clean:
	rm -rf dist
