package util

import "strconv"

func MustConvertToBool(input string) bool {
	value, _ := strconv.ParseBool(input)

	return value
}
