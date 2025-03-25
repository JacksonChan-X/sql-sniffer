package helper

import (
	"strings"
)

// 判断mysql的SQL语句中有多少个param_count
func GetParamCount(sql string) uint16 {
	count := strings.Count(sql, "?")
	return uint16(count)
}
