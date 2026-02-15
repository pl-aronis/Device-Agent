package main

import (
	"fmt"
	"reflect"

	"go.mozilla.org/pkcs7"
)

func main() {
	p7 := &pkcs7.PKCS7{}
	t := reflect.TypeOf(p7)
	for i := 0; i < t.NumMethod(); i++ {
		fmt.Println(t.Method(i).Name)
	}
}
