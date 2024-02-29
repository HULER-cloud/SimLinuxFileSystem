package in_out

import (
	"fmt"
	"strings"
)

var CH_IN chan string  // shell前后端通信用，从shell到simdisk
var CH_OUT chan string // 从simdisk到shell

var Ignore_print int

func Out(message string) {

	if Ignore_print == 1 {
		CH_OUT <- message
	} else {
		if message != "finish" {
			if strings.HasPrefix(message, "simdisk") &&
				strings.HasSuffix(message, ">>> ") {
				fmt.Print(message)
			} else {
				fmt.Println(message)
			}
		}
	}
}
