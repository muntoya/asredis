package asredis

import "errors"

var (
	ErrNotConnected = errors.New("redis: not connected")
	ErrNotRunning = errors.New("redis: shutdown and can't use any more")
	ErrWaitReplyTimeout = errors.New("redis: wait reply time out")
	ErrExpectingLinefeed = errors.New("redis: expecting a linefeed byte")
	ErrUnexpectedReplyType = errors.New("redis: can't parse reply")
	ErrUnexpectedCtrlType = errors.New("redis: can't process control command")
)
