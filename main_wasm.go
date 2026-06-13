//go:build js && wasm

package main

import "syscall/js"

func main() {
	js.Global().Set("GByteDecompileText", js.FuncOf(decompileTextJS))
	js.Global().Set("GByteDecompileBytes", js.FuncOf(decompileBytesJS))
	select {}
}

func decompileTextJS(this js.Value, args []js.Value) any {
	if len(args) == 0 {
		return resultObject("", "missing input text")
	}
	data, err := parseHexBytes(args[0].String())
	if err != nil {
		data = []byte(args[0].String())
	}
	return decompileResult(data)
}

func decompileBytesJS(this js.Value, args []js.Value) any {
	if len(args) == 0 {
		return resultObject("", "missing input bytes")
	}
	input := args[0]
	data := make([]byte, input.Get("byteLength").Int())
	js.CopyBytesToGo(data, input)
	return decompileResult(data)
}

func decompileResult(data []byte) js.Value {
	output, err := decompileData(data)
	if err != nil {
		return resultObject("", err.Error())
	}
	return resultObject(output, "")
}

func resultObject(output, errText string) js.Value {
	obj := js.Global().Get("Object").New()
	obj.Set("ok", errText == "")
	obj.Set("output", output)
	obj.Set("error", errText)
	return obj
}
