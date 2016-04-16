package confs

import (
	"fmt"
)

type Error struct {
	text string
	path string
}

func newError(err error, path string) *Error {
	return &Error{ text: err.Error(), path: path }
}

func (me *Error) Error() string {
	return me.text
}

func (me *Error) Path() string {
	return me.path
}

func (me Error) String() string {
	return fmt.Sprintf("%s: %s", me.path, me.text)
}
