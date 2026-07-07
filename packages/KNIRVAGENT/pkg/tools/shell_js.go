//go:build js && wasm

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"syscall/js"
)

func executeHostCommand(ctx context.Context, req HostCommandRequest) (string, error) {
	result, err := callJSHost(ctx, "exec", map[string]any{
		"command":     req.Command,
		"working_dir": req.WorkingDir,
		"timeout_ms":  req.TimeoutMS,
	})
	if err != nil {
		return "", err
	}
	return hostResultString(result)
}

func callJSHost(ctx context.Context, method string, payload map[string]any) (js.Value, error) {
	host := js.Global().Get("knirvagentHost")
	if host.IsUndefined() || host.IsNull() {
		return js.Value{}, fmt.Errorf("browser host bridge globalThis.knirvagentHost is not configured")
	}

	fn := host.Get(method)
	if fn.IsUndefined() || fn.IsNull() || fn.Type() != js.TypeFunction {
		return js.Value{}, fmt.Errorf("browser host bridge missing method %q", method)
	}

	value := fn.Invoke(js.ValueOf(payload))
	if value.IsUndefined() || value.IsNull() {
		return js.ValueOf(map[string]any{"output": ""}), nil
	}
	if value.Get("then").Type() != js.TypeFunction {
		return value, nil
	}

	done := make(chan struct{})
	var resolved js.Value
	var rejected js.Value

	then := js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) > 0 {
			resolved = args[0]
		}
		close(done)
		return nil
	})
	catchFn := js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) > 0 {
			rejected = args[0]
		}
		close(done)
		return nil
	})
	defer then.Release()
	defer catchFn.Release()

	value.Call("then", then).Call("catch", catchFn)
	select {
	case <-ctx.Done():
		return js.Value{}, ctx.Err()
	case <-done:
	}

	if !rejected.IsUndefined() && !rejected.IsNull() {
		return js.Value{}, fmt.Errorf(jsErrorString(rejected))
	}
	return resolved, nil
}

func hostResultString(value js.Value) (string, error) {
	if value.Type() == js.TypeString {
		return value.String(), nil
	}
	if errValue := value.Get("error"); !errValue.IsUndefined() && !errValue.IsNull() && errValue.String() != "" {
		return "", fmt.Errorf(errValue.String())
	}
	for _, key := range []string{"output", "stdout", "content"} {
		v := value.Get(key)
		if !v.IsUndefined() && !v.IsNull() {
			return v.String(), nil
		}
	}
	jsonValue := js.Global().Get("JSON")
	if !jsonValue.IsUndefined() {
		return jsonValue.Call("stringify", value).String(), nil
	}
	return fmt.Sprintf("%v", value), nil
}

func jsErrorString(value js.Value) string {
	if value.Type() == js.TypeString {
		return value.String()
	}
	msg := value.Get("message")
	if !msg.IsUndefined() && !msg.IsNull() && msg.String() != "" {
		return msg.String()
	}
	data, _ := json.Marshal(value)
	if len(data) > 0 && string(data) != "{}" {
		return string(data)
	}
	return fmt.Sprintf("%v", value)
}
