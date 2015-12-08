package common

type Error interface {
	error

	IsRedisError() bool
}
