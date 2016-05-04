package connman

import (
	"fmt"
	"reflect"
	"github.com/godbus/dbus"
)

const (
	ServiceGetProperties = "net.connman.Service.GetProperties"
	ServiceSetProperty = "net.connman.Service.SetProperty"
	ServiceConnect = "net.connman.Service.Connect"
	ServiceDisconnect = "net.connman.Service.Disconnect"
)

type ipv4 struct {
	Method string
	Address string
	Netmask string
	Gateway string
}

type ipv6 struct {
	Method string
	Address string
	PrefixLength uint8
	Gateway string
}

type Service struct {
	object dbus.BusObject
	Type string
	Name string
	Id string
	State string
	AutoConnect bool
	Nameservers []string
	Timeservers []string
	Domains []string
	IPv4 ipv4
	IPv6 ipv6
	Security []string
}

var services = map[string]*Service{}

func NewService(id string) (*Service, error) {
	server, serr := NewServer()
	if serr != nil {
		return nil, serr
	}

	var s *Service = services[id]

	if s == nil {
		path := dbus.ObjectPath("/net/connman/service/" + id)

		s = &Service{object: server.conn.Object(server.address, path), Id: id}

		if err := s.GetProperties(); err != nil {
			return s, err
		}
	}

	return s, nil
}

func (s *Service) GetProperties() error {
	props := make(map[string]dbus.Variant)
	
	if err := s.object.Call(ServiceGetProperties, 0).Store(&props); err != nil {
		return err
	}

	if err := DictStore(props, reflect.ValueOf(s).Elem()); err != nil {
		return err
	}
	
	return nil
}

func (s *Service) setProperty(name string, value interface{}) error {
	//defer func() {
	//	err, _ = recover().(error)
	//}()

	vari := dbus.MakeVariant(value)

	call := s.object.Call(ServiceSetProperty, 0, name, vari)
	
	if call.Err != nil {
		return fmt.Errorf("can't set property '%s': %s", name, call.Err.Error())
	}

	return nil
}

func (s *Service) SetAutoConnect(value bool) error {
	if err := s.setProperty("AutoConnect", value); err != nil {
		return err
	}

	if err := s.GetProperties(); err != nil {
		if s.AutoConnect != value {
			return fmt.Errorf("failed to set 'AutoConnect' property")
		}
	}

	return nil
}

func (s *Service) SetIPv4(method string, addr string, mask string, gw string) error {
	value := map[string]dbus.Variant{}
	settings := ""

	value["Method"] = dbus.MakeVariant(method)
	if method == "manual" {
		value["Address"] = dbus.MakeVariant(addr)
		value["Netmask"] = dbus.MakeVariant(mask)
		value["Gateway"] = dbus.MakeVariant(gw)

		settings = fmt.Sprintf(" (%s/%s/%s)", addr, mask, gw)
	}

	if err := s.setProperty("IPv4.Configuration", value); err != nil {
		return err
	}

	if err := s.GetProperties(); err != nil {
		ip := s.IPv4
		ok :=  method == ip.Method &&
			(method != "manual" ||
			(method == "manual" && addr == ip.Address && mask == ip.Netmask && gw == ip.Gateway))
		if !ok {
			return fmt.Errorf("Failed to set 'IPv4.Configuration' property")
		}
	}

	Printf(LogDebug, "set %s/%s IPv4 to %s%s\n", s.Type, s.Name, method, settings)
	
	return nil
}

func (s *Service) Connect() error {
	call := s.object.Call(ServiceConnect, 0)
	
	if call.Err != nil {
		return fmt.Errorf("can't connect %s '%s': %s", s.Type, s.Name, call.Err.Error())
	}

	Printf(LogInfo, "connecting %s/%s service\n", s.Type, s.Name)

	return nil
}

func (s *Service) Disconnect() error {
	call := s.object.Call(ServiceDisconnect, 0)
	
	if call.Err != nil {
		return fmt.Errorf("can't disconnect %s '%s': %s", s.Type, s.Name, call.Err.Error())
	}

	Printf(LogInfo, "disconnecting %s/%s service\n", s.Type, s.Name)

	return nil
}
