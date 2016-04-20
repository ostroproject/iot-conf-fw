package neard

import (
	"path"
	"regexp"
	"reflect"
	"github.com/godbus/dbus"
	"ostro/confs"
)

type adapter struct {
	srv *server
	object dbus.BusObject
	name string
	path string
	Mode string
	Powered bool
	Polling bool
	Protocols []string
}

const (
	adapterInterface = "org.neard.Adapter"
	startPollLoop = "org.neard.Adapter.StartPollLoop"
)

var (
	adapters = map[string]*adapter{}
)

func newAdapter(srv *server, objPath string) (ad *adapter) {
	var name string = path.Base(objPath)
	var object dbus.BusObject
	var exists bool
	
	if ad, exists = adapters[name]; !exists {
		object = srv.conn.Object(srv.address, dbus.ObjectPath(objPath))
		ad = &adapter{srv: srv, object: object, name: name, path: objPath}
		adapters[name] = ad

		srv.registerObject(ad)

		srv.sendAdapterEvent(&AdapterEvent{
			Event: "created",
			Name: name,
			Path: objPath})
	}

	return
}

func findAdapter(server *server, pattern string) *adapter {
	var (
		re *regexp.Regexp
		err error
		ad *adapter
		found bool
		name string
	)

	if pattern == "" || pattern == "*" {
		pattern = ".*"
	}
	
	if re, err = regexp.Compile(pattern); err != nil {
		confs.Debugf("can't find adapter because of bad pattern '%s': %v", pattern, err)
		return nil
	}

	found = false
	
	for name, ad = range adapters {
		if re.Match([]byte(name)) {
			found = true
			break
		}
	}

	if found {
		return ad
	}
	
	confs.Debugf("can't find matching adapter for '%s' pattern", pattern)
	
	return nil
}

func (me *adapter) active() bool {
	return me.Powered && (me.Polling || me.Mode != "Idle")
}

func (me *adapter) storeProperties(props map[string]dbus.Variant) bool {
	confs.Debugf("set properties of '%s': %v", me.name, props)

	oldPowered := me.Powered
	oldActive := me.active()

	success := DictStore(props, reflect.ValueOf(me).Elem())

	if !success {
		confs.Errorf("Failed to store properties of '%s' adapter", me.name)
		return false
	}

	newActive := me.active()
	
	if oldPowered != me.Powered || oldActive != newActive {
		me.srv.sendAdapterEvent(&AdapterEvent{
			Event: "PropertyChanged",
			Name: me.name,
			Path: me.path,
			Enabled: me.Powered,
			Active: newActive})
	}
	
	return true
}

func (me *adapter) getProperties() bool {
	props := map[string]dbus.Variant{}

	if err := me.object.Call(getAllProperties, 0, adapterInterface).Store(&props); err != nil {
		confs.Errorf("Failed to get properties of '%s': %v", me.name, err)
		return false
	}

	return me.storeProperties(props)
}

func (me *adapter) setProperty(name string, value interface{}) bool {
	if err := me.object.Call(setProperty, 0, adapterInterface, name, dbus.MakeVariant(value)).Err; err != nil {
		confs.Errorf("failed to set '%s' property to '%v' on '%s': %v", name, value, me.name, err)
		return false
	}

	me.getProperties()

	return true
}

func (me *adapter) enable(enable bool) bool {
	if enable == me.Powered {
		return true
	}

	if !me.setProperty("Powered", enable) {
		return false
	}

	if enable != me.Powered {
		confs.Errorf("Failed to set 'Powered' to %v on '%s': value remained %v", enable, me.name, me.Powered)
		return false
	}

	confs.Debugf("Powered set to %v on '%s'", me.Powered, me.name);
	
	return true
}


func (me *adapter) startPolling(mode string) bool {
	if mode != "Initiator" && mode != "Target" && mode != "Dual" {
		confs.Debugf("attempt to start polling with invalid mode '%s'", mode)
		return false
	}

	if mode == me.Mode {
		return true
	}
	
	if err := me.object.Call(startPollLoop, 0, mode).Err; err != nil {
		confs.Errorf("failed to start polling (mode '%s') on '%s': %v", mode, me.name, err)
		return false
	}

	return true
}

func (me *adapter) getObjectPath() string {
	return me.path
}
