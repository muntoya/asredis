package asredis


func JoinArgs(s interface{}, args []interface{}) []interface{} {
	return append([]interface{}{s}, args...)
}
