package main

import (
	"context"
	"fmt"
	"time"

	"github.com/robertkrimen/otto"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	vm := otto.New()

	errChan := make(chan error, 1)
	resultChan := make(chan interface{}, 1)

	go func() {
		// Test simple boolean return
		code := `return true;`
		fmt.Println("Executing code:", code)
		
		result, err := vm.Run(code)
		if err != nil {
			errChan <- err
			return
		}

		fmt.Println("Result from vm.Run:", result)
		
		val, err := result.Export()
		if err != nil {
			errChan <- err
			return
		}

		fmt.Printf("Exported value: %v (%T)\n", val, val)
		resultChan <- val
	}()

	select {
	case <-ctx.Done():
		vm.Interrupt <- func() {}
		fmt.Println("Timeout:", ctx.Err())
	case err := <-errChan:
		fmt.Println("Error:", err)
	case result := <-resultChan:
		fmt.Println("Success:", result)
	}
}
