package mysql

import "errors"

var (
	ErrMalformPacket = errors.New("MALFORM_PACKET")
	ErrorStream      = errors.New("STREAM INVALID")
	ErrTimeOut       = errors.New("stream timeout")
)
