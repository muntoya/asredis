package asredis

import "errors"

var (
	ErrExpectingLinefeed = errors.New("redis: expecting a linefeed byte")
	ErrUnexpectedReplyType = errors.New("redis: can't parse reply")
)
