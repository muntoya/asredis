package common

import "fmt"

type RedisError struct {
	msg string
}

func (this RedisError) Error() string {

	return fmt.Sprintf("REDIS_ERROR - %s", this.msg)
}

func NewRedisError(msg string) error {
	return RedisError{msg}
}

func NewRedisErrorf(format string, args ...interface{}) error {
	return NewRedisError(fmt.Sprintf(format, args...))
}
