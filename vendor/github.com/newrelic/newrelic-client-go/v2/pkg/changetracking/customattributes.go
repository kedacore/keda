package changetracking

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/robertkrimen/otto"
)

// ReadCustomAttributesJS reads a JS object (not JSON) from a file or string and returns it as a map[string]interface{}.
// Use isFile=true to read from a file, isFile=false to use the string directly.
func ReadCustomAttributesJS(input string, isFile bool) (map[string]interface{}, error) {
	var jsRaw string
	if isFile {
		f, err := os.Open(input)
		if err != nil {
			return nil, err
		}
		defer func() {
			cerr := f.Close()
			if cerr != nil {
				fmt.Printf("error closing file: %v\n", cerr)
			}
		}()
		scanner := bufio.NewScanner(f)
		var b strings.Builder
		for scanner.Scan() {
			b.WriteString(scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			return nil, err
		}
		jsRaw = b.String()
	} else {
		jsRaw = input
	}

	// Use otto to evaluate the JS object and convert to Go map
	vm := otto.New()
	// Wrap the object in parentheses to make it a valid JS expression
	value, err := vm.Run("(" + jsRaw + ")")
	if err != nil {
		return nil, err
	}
	obj, err := value.Export()
	if err != nil {
		return nil, err
	}
	attrs, ok := obj.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("custom attributes JS is not an object")
	}
	return attrs, nil
}
