package main

import (
	"github.com/docker/app/pkg/yatee"
	"github.com/gopherjs/gopherjs/js"
)

func processJS(input, parameters string) (string, string) {
	res, err := yatee.ProcessStrings(input, parameters)
	var errStr string
	if err != nil {
		errStr = err.Error()
	}
	return res, errStr
}

func main() {
	js.Global.Set("yatee", map[string]interface{}{
		"ProcessJS": processJS,
	})
}
