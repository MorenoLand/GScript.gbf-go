# GScript.gbf-go

Minimal GS2 bytecode decompiler written in Go.

gbf-go reads GScript `.gs2bc` bytecode, parses the function table, string
table, and instruction stream, then emits readable GS2-like source. It is
intentionally small: one decompiler core, a CLI entrypoint, and a WASM
entrypoint for browser use.

## Features

- Decompiles raw `.gs2bc` files or whitespace-separated hex input.
- Recovers named functions and common parameter prologues.
- Handles the known GS2 VM opcode set used by the current fixtures.
- Reconstructs calls, object methods, arrays, foreach/counting loops, dispatch
  chains, and GUI constructor blocks.
- Builds as a native CLI or as WebAssembly.

## CLI

Decompile next to the input file:

```sh
go run . path/to/weapon.gs2bc
```

Write to a specific output path:

```sh
go run . -o out.gs2 path/to/weapon.gs2bc
```

Pipe or paste hex:

```sh
cat bytecode.hex | go run .
```

## WASM Example

Build the browser example:

```sh
GOOS=js GOARCH=wasm go build -o examples/web/gbf.wasm .
wasm_exec="$(find "$(go env GOROOT)" -path "*/wasm_exec.js" -type f | head -n 1)"
cp "$wasm_exec" examples/web/
```

The web example exposes the decompiler through Go's WASM runtime and lets you
load a `.gs2bc` file or paste hex input. It expects `gbf.wasm` and
`wasm_exec.js` beside `examples/web/index.html`.

## Development

Run checks:

```sh
go test -count=1 ./...
go build .
```

Build WASM directly:

```sh
GOOS=js GOARCH=wasm go build -o examples/web/gbf.wasm .
```

## License

Apache License 2.0. See `LICENSE`.
