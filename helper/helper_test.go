package helper

import (
	"testing"
)

func TestGetIPByInterface(t *testing.T) {
	ip4s, ip6s, err := GetIPByInterface("feth490")
	if err != nil {
		t.Error(err)
	}
	t.Log(ip4s)
	t.Log(ip6s)
}
