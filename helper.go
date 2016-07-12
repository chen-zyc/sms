package sms

import (
	"runtime"
	"strconv"
)

func caller() string {
	_, file, line, ok := runtime.Caller(1)
	if ok {
		return file + ":" + strconv.Itoa(line)
	}
	return "<?>"
}
