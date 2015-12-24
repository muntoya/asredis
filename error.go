package asredis

import "errors"

var (
	ErrNotConnected = errors.New("redis: not connected")
	ErrExpectingLinefeed = errors.New("redis: expecting a linefeed byte")
	ErrUnexpectedReplyType = errors.New("redis: can't parse reply")
	ErrUnexpectedCtrlType = errors.New("redis: can't process control command")
)
