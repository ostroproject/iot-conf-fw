package neard

import (
	"fmt"
	"errors"
)

const (
	UnknownMessage = iota

	Activate
	Adapter
	State
	Error
	WlanHandover
	ConfigData
	StartApplication
)

type AdapterEvent struct {
	Event string
	Name string
	Path string
	Enabled bool
	Active bool
}

type Credentials struct {
	Network int
	SSID string
	AuthType string
	Key string
	EncType string
	X509Cert string
}

type Message struct {
	Id int
	data interface{}
}

func (me *Message) Bool() (boolean bool) {
	if b, ok := me.data.(bool); ok {
		boolean = b
	} else {
		boolean = false
	}
	return
}

func (me *Message) Int() (integer int) {
	if i, ok := me.data.(int); ok {
		integer = i
	} else {
		integer = 0
	}
	return
}

func (me *Message) String() (str string) {
	if s, ok := me.data.(string); ok {
		str = s
	} else {
		str = fmt.Sprintf("%v", me.data)
	}
	return
}

func (me *Message) Error() (err error) {
	if s, ok := me.data.(string); ok {
		err = errors.New(s)
	} else {
		err = errors.New("unknown error")
	}
	return
}

func (me *Message) AdapterEvent() (ev AdapterEvent) {
	if e, ok := me.data.(AdapterEvent); ok {
		ev = e
	} else {
		ev = AdapterEvent{Event: "Unknown"}
	}
	return
}

func (me *Message) Credentials() (cred Credentials) {
	if c, ok := me.data.(Credentials); ok {
		cred = c
	} else {
		cred = Credentials{}
	}
	return
}

