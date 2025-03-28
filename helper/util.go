package helper

import (
	"bytes"
	"io"
	"time"
)

func GetNowStr(isClient bool) string {
	var msg string
	msg += time.Now().Format("2006-01-02 15:04:05")
	if isClient {
		msg += "| cli -> ser |"
	} else {
		msg += "| ser -> cli |"
	}
	return msg
}

func LengthEncodedInt(input []byte) (num uint64, isNull bool, n int) {

	switch input[0] {

	case 0xfb:
		n = 1
		isNull = true
		return
	case 0xfc:
		num = uint64(input[1]) | uint64(input[2])<<8
		n = 3
		return
	case 0xfd:
		num = uint64(input[1]) | uint64(input[2])<<8 | uint64(input[3])<<16
		n = 4
		return
	case 0xfe:
		num = uint64(input[1]) | uint64(input[2])<<8 | uint64(input[3])<<16 |
			uint64(input[4])<<24 | uint64(input[5])<<32 | uint64(input[6])<<40 |
			uint64(input[7])<<48 | uint64(input[8])<<56
		n = 9
		return
	}

	num = uint64(input[0])
	n = 1
	return
}

func LengthEncodedString(b []byte) ([]byte, bool, int, error) {

	num, isNull, n := LengthEncodedInt(b)
	if num < 1 {
		return nil, isNull, n, nil
	}

	n += int(num)

	if len(b) >= n {
		return b[n-int(num) : n], false, n, nil
	}
	return nil, false, n, io.EOF
}

func ReadStringFromByte(b []byte) (string, int) {

	var l int
	l = bytes.IndexByte(b, 0x00)
	if l == -1 {
		l = len(b)
	}
	return string(b[0:l]), l
}
