package neard

import (
	"fmt"
	"os"
	"path"
	"github.com/godbus/dbus"
	"ostro/confs"
)

type ObjectInterface interface {
	getObjectPath() string
	storeProperties(props map[string]dbus.Variant) bool
}

type Server struct {
	events   map[string]bool
	reqchan  chan<- Message
	Evchan  <-chan Message
}

type server struct {
	conn *dbus.Conn
	rootObject dbus.BusObject
	neardObject dbus.BusObject
	address string
	events []string
	sigchan <-chan *dbus.Signal
	reqchan <-chan Message
	evchan chan<- Message
	objectInterfaces map[string]ObjectInterface
}

const (
	objectManagerInterface = "org.freedesktop.DBus.ObjectManager"
	interfacesAddedMember = "InterfacesAdded"

	interfacesAddedFilter = "type='signal', path='/',interface='" + objectManagerInterface + "', member='" + interfacesAddedMember + "'"
	
	getManagedObjects = "org.freedesktop.DBus.ObjectManager.GetManagedObjects"
	interfacesAdded = objectManagerInterface + "." + interfacesAddedMember

	propertyInterface = "org.freedesktop.DBus.Properties"
	setMember = "Set"
	setProperty =  propertyInterface + "." + setMember
	getMember = "Get"
	getProperty =  propertyInterface + "." + getMember
	getAllMember = "GetAll"
	getAllProperties =  propertyInterface + "." + getAllMember
	propertiesChangedMember = "PropertiesChanged"
	propertiesChanged = propertyInterface + "." + propertiesChangedMember
)

var (
	srvif *Server = nil
)

func Initialize() bool {
	if err := confs.Initialize(); err != nil {
		return false
	}

	return true;
}

func NewServer(events []string) *Server {
	if srvif != nil {
		for _, ev := range events {
			if !srvif.events[ev] {
				confs.Errorf("attempt to subsequent creation of Neard server with mismatching events")
				return nil
			}
		}
	} else {
		if len(events) > 0 {
			if conn, err := dbus.SystemBus(); err != nil {
				confs.Errorf("Failed to get D-Bus System Bus: %v", err)
			} else {
				address := "org.neard"
				rootObject := conn.Object(address, dbus.ObjectPath("/"))
				neardObject := conn.Object(address, dbus.ObjectPath("/org/neard"))
				
				conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, interfacesAddedFilter)
				
				sigchan := make(chan *dbus.Signal, 20)
				reqchan := make(chan Message)
				evchan := make(chan Message, 30)
				
				conn.Signal(sigchan)
				
				srv := &server{
					conn: conn,
					rootObject: rootObject,
					neardObject: neardObject,
					address: address,
					events: events,
					sigchan: sigchan,
					reqchan: reqchan,
					evchan: evchan,
					objectInterfaces: map[string]ObjectInterface{}}
				
				srvif = &Server{
					events: map[string]bool{},
					reqchan: reqchan,
					Evchan: evchan}

				for _, ev := range events {
					srvif.events[ev] = true
				}

				go srv.loop()
			}
		}
	}

	return srvif
}

func (me *Server) Activate(pattern string) {
	me.reqchan <- Message{Id: Activate, data: pattern}
}


func (me *server) sendAdapterEvent(ev *AdapterEvent) {
	me.evchan <- Message{Id: Adapter, data: *ev}
}

func (me *server) sendConfigData(data string) {
	me.evchan <- Message{Id: ConfigData, data: data}
}

func (me *server) sendWlanHandoverRequest(cred *Credentials) {
	me.evchan <- Message{Id: WlanHandover, data: *cred}
}

func (me *server) sendError(format string, args ...interface{}) {
	me.evchan <- Message{Id: Error, data: fmt.Sprintf(format,  args...)}
}


func (me *server) connect() {
	if me.getAdapters() {
		if !newAgent(me, me.events) {
			confs.Errorf("can't recover from errors: giving up ...")
			os.Exit(1)
		}
	}
}

func (me *server) loop() {
	me.connect()

	for {
		select {
		case req := <-me.reqchan:
			me.requestHandler(req)
		case sig := <-me.sigchan:
			me.signalHandler(sig)
		}
	}
}

func (me *server) requestHandler(req Message) {
	switch req.Id {

	case Activate:
		pattern := req.String()
		if adapter := findAdapter(me, pattern);  adapter != nil {
			if !adapter.enable(true) || !adapter.startPolling("Dual") {
				me.sendError("failed to activate '%s' adapter", adapter.name)
			}
		} else {
			me.sendError("can't find any matching adapter for '%s' pattern", pattern)
		}

	default:
		confs.Debugf("ignoring unsupprted request (id: %d)", req.Id)
	}
}

func (me *server) signalHandler(sig *dbus.Signal) {
	if sig.Path == "/" && sig.Name == interfacesAdded {
		objPath := dbus.ObjectPath("")
		ifs := map[string]map[string]dbus.Variant{}

		if err := dbus.Store(sig.Body, &objPath, &ifs); err != nil {
			confs.Errorf("failed to parse D-Bus interfacesAdded message: %v", err)
		} else {
			pathBase := path.Base(string(objPath))
			if len(pathBase) > 6 && pathBase[:6] == "record" {
				me.recordHandler(ifs)
			}
		}
	} else if obj, exists := me.findObject(string(sig.Path)); exists && sig.Name == propertiesChanged {
		interf := ""
		changed := map[string]dbus.Variant{}
		invalid := []string{}

		if err := dbus.Store(sig.Body, &interf, &changed, &invalid); err != nil {
			confs.Errorf("failed to parse D-Bus propertyChanged message: %v", err)
		} else {
			obj.storeProperties(changed)
		}
	}
}

func (me *server) recordHandler(ifs  map[string]map[string]dbus.Variant) {
	var (
		r map[string]dbus.Variant
		typ, enc, txt dbus.Variant
		exists bool
	)
	
	if r = ifs["org.neard.Record"]; r == nil {
		confs.Debugf("ignoring NDEF record: 'org.neard.Record' interface not implemented")
		return
	}

	if typ = r["Type"]; typ.Signature().String() != "s" {
		confs.Debugf("ignoring NDEF record: missing or wrong 'Type' field (%s)", typ.Signature().String())
		return
	}
	
	switch t := typ.Value().(string); t {
	case "Text":
		if enc = r["Encoding"]; enc.Signature().String() != "s" || enc.Value().(string) != "UTF-8" {
			confs.Debugf("ignoring NDEF record: missing or wrong 'Encoding' field (%s)",
				enc.Signature().String())
			return
		}

		if txt, exists = r["Representation"]; !exists {
			confs.Debugf("ignoring NDEF record: missing 'Represenation' field")
			return
		}
		if signature := txt.Signature().String(); signature != "s" {
			confs.Debugf("ignoring NDEF record: 'Represenation' field has wrong type (%s)", signature)
			return
		}

		me.sendConfigData(txt.Value().(string))
	default:
		confs.Debugf("ignoring NDEF record: unsupported 'Type' '%s'", t)
	}
}

	
func (me *server) getAdapters() bool {
	objects := map[dbus.ObjectPath]map[string]map[string]dbus.Variant{}

	if err := me.rootObject.Call(getManagedObjects, 0).Store(&objects); err != nil {
		confs.Errorf("failed to obtain manged objects: %v", err)
		return false
	}

	for objPath, interfaces := range objects {
		if props := interfaces["org.neard.Adapter"]; props != nil {
			confs.Debugf("Found adapter '%s'", objPath)			
			newAdapter(me, string(objPath)).storeProperties(props)
		}
	}
	
	return true
}

func (me *server) registerObject(obj ObjectInterface) {
	objectPath := obj.getObjectPath()

	if _, exists := me.objectInterfaces[objectPath]; !exists {
		filter := fmt.Sprintf("type='signal', path='%s', interface='%s', member='%s'",
			objectPath, propertyInterface, propertiesChangedMember)
		me.conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, filter)

		me.objectInterfaces[objectPath] = obj
	}
}

func (me *server) findObject(objectPath string) (obj ObjectInterface, exists bool) {
	obj, exists = me.objectInterfaces[objectPath]
	return
}
