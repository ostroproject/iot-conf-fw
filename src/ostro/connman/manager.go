package connman

import (
	"fmt"
	"reflect"
	"path/filepath"
	"github.com/godbus/dbus"
)

const (
	ManagerGetProperties = "net.connman.Manager.GetProperties"
	ManagerSetProperty = "net.connman.Manager.SetProperty"
	ManagerGetTechnologies = "net.connman.Manager.GetTechnologies"
	ManagerGetServices = "net.connman.Manager.GetServices"
)

type objList struct {
	Path dbus.ObjectPath
	Props map[string]dbus.Variant
}

type techList struct {
	Type string
	Powered bool
	Connected bool
	Tethering bool
}

type servList struct {
	Type string
	Name string
	Id string
	State string
}

type Manager struct {
	object dbus.BusObject
	State string
	OfflineMode bool
	Technologies []techList
	Services []servList
	WiFiName string
}

var manager *Manager = nil

func NewManager() (*Manager, error) {
	server, serr := NewServer()
	if serr != nil {
		return nil, serr
	}
	
	if manager == nil {
		path := dbus.ObjectPath("/")

		m := &Manager{object: server.conn.Object(server.address, path)}

		if err := m.GetProperties(); err != nil {
			return m, err
		}

		if err := m.GetTechnologies(); err != nil {
			return m, err
		}

		if err := m.scanAll(); err != nil {
			return m, err
		}

		if err := m.GetServices(); err != nil {
			return m, err
		}

		manager = m
	}
		
	return manager, nil
}

func (m *Manager) GetProperties() error {
	props := make(map[string]dbus.Variant)

	if err := m.object.Call(ManagerGetProperties, 0).Store(&props); err != nil {
		return err
	}

	if err := DictStore(props, reflect.ValueOf(m).Elem()); err != nil {
		return err
	}

	return nil
}

func (m *Manager) GetTechnologies() error {
	ol := []objList{}

	if err := m.object.Call(ManagerGetTechnologies, 0).Store(&ol); err != nil {
		return err
	}

	Printf(LogDebug, "found tecnologies:\n")

	tl := []techList{}

	for _, o := range ol {
		t := techList{}
		if err := DictStore(o.Props, reflect.ValueOf(&t).Elem()); err != nil {
			return err
		}
		tl = append(tl, t)
		Printf(LogDebug, "   %s\n", t.Type)
	}

	if len(tl) == 0 {
		Printf(LogDebug, "   <none>\n")
	}

	m.Technologies = tl
		
	return nil
}

func (m *Manager) GetServices() error {
	ol := []objList{}

	if err := m.object.Call(ManagerGetServices, 0).Store(&ol); err != nil {
		return err
	}

	Printf(LogDebug, "found services:\n")
	
	sl := []servList{}
	n := ""

	for _, o := range ol {
		s := servList{Id: filepath.Base(string(o.Path))}
		if err := DictStore(o.Props, reflect.ValueOf(&s).Elem()); err != nil {
			return err
		}
		sl = append(sl, s)
		if s.State == "ready" || s.State == "online" {
			n = s.Name
		}
		Printf(LogDebug, "   %-10s  %-24s  %s\n", s.Type, s.Name, s.State)
	}

	if len(sl) == 0 {
		Printf(LogDebug, "   <none>\n")
	}

	m.Services = sl
	m.WiFiName = n

	return nil
}

func (m *Manager) GetServiceId(typ string, name string) (string, error) {
	for _, s := range m.Services {
		if typ == s.Type && name == s.Name {
			return s.Id, nil
		}
	}

	return "", fmt.Errorf("%s service %s not found", typ, name)
}


func (m *Manager) scanAll() error {
	for _, t := range m.Technologies {
		if !willBeDisabled(t.Type) {
			if t.Type == "wifi" {
				w, werr := NewTechnology(t.Type)
				if werr != nil {
					return werr
				}
				
				if err := w.Scan(); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func willBeDisabled(technology string) bool {
	for _, dt := range DisabledTechnologies {
		if dt == technology {
			return true
		}
	}
	return false
}
