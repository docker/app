package main

import (
	"github.com/docker/app/internal/yatee/yatee"
	"github.com/gopherjs/gopherjs/js"
)

func processJS(input, settings string) (string, string) {
	res, err := yatee.ProcessStrings(input, settings)
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
