package net

import "fmt"
import "net"

func lookupMX(s string) string {
	_, e := net.LookupMX(s)
	if e == nil {
		return ""
	}
	return fmt.Sprintf("%s", e)
}
