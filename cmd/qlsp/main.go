package main

import (
	"github.com/jhchabran/qlsp"
)

func main() {
	err := qlsp.Serve(&qlsp.BaseServer{})
	if err != nil {
		panic(err)
	}
}
