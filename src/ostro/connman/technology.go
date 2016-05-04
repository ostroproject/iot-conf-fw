package connman

import (
	"fmt"
	"time"
	"reflect"
	"github.com/godbus/dbus"
)

const (
	TechnologyGetProperties = "net.connman.Technology.GetProperties"
	TechnologySetProperty = "net.connman.Technology.SetProperty"
	TechnologyScan = "net.connman.Technology.Scan"
)

type Technology struct {
	object dbus.BusObject
	Type string
	Name string
	Powered bool
	Tethering bool
	TetheringIdentifier string
	TetheringPassphrase string
}

var technologies = map[string]*Technology{}

func NewTechnology(ty string) (*Technology, error) {
	server, serr := NewServer()
	if serr != nil {
		return nil, serr
	}
	
	var t *Technology = technologies[ty]

	if t == nil {
		path := dbus.ObjectPath("/net/connman/technology/" + ty)

		t = &Technology{object: server.conn.Object(server.address, path), Type: ty}
	
		if err := t.GetProperties(); err != nil {
			return t, err
		}

		technologies[ty] = t
	}

	return t, nil
}

func (t *Technology) GetProperties() error {
	props := make(map[string]dbus.Variant)
	
	if err := t.object.Call(TechnologyGetProperties, 0).Store(&props); err != nil {
		return err
	}

	if err := DictStore(props, reflect.ValueOf(t).Elem()); err != nil {
		return err
	}
	
	return nil
}

func (t *Technology) setProperty(name string, value interface{}) error {
	//defer func() {
	//	err, _ = recover().(error)
	//}()

	vari := dbus.MakeVariant(value)

	call := t.object.Call(TechnologySetProperty, 0, name, vari)
	
	if call.Err != nil {
		return fmt.Errorf("can't set property '%s': %s", name, call.Err.Error())
	}

	return nil
}


func (t *Technology) SetTetheringParameters(id string, pwd string) error {
	if id != t.TetheringIdentifier {
		if err := t.setProperty("TetheringIdentifier", id); err != nil {
			return err
		}
	}

	if pwd != t.TetheringPassphrase {
		if err := t.setProperty("TetheringPassphrase", pwd); err != nil {
			return err
		}
	}

	if err := t.GetProperties(); err != nil {
		return err
	}

	if id != t.TetheringIdentifier || pwd != t.TetheringPassphrase {
		return fmt.Errorf("failed to set credentials")
	}

	return nil
}

func (t *Technology) SetPowered(power bool) error {
	var scanErr error

	if power != t.Powered {
		state := "disable"
		if power {
			state = "enable"
		}
		Printf(LogInfo, "%s %s\n", state, t.Type)

		needScan := power && (t.Type == "wifi" || t.Type == "bluetooth")

		if err := t.setProperty("Powered", power); err != nil {
			return err
		}

		if err := t.GetProperties(); err != nil {
			return err
		}

		if power != t.Powered {
			return fmt.Errorf("failed to %s", state)
		}

		if needScan {
			for i := 0;  i < 5;  i++ {
				if scanErr = t.Scan(); scanErr == nil {
					break
				}

				Printf(LogDebug, "%v\n", scanErr) 

				time.Sleep(1 * time.Second)
			}

			if scanErr != nil {
				return scanErr
			}

			m, merr := NewManager()
			if merr != nil {
				return merr
			}

			if err := m.GetServices(); err != nil {
				return err
			}
		}
	}

	return nil
}

func (t *Technology) SetTethering(tether bool) error {
	if tether != t.Tethering {
		state := "off"
		if tether {
			state = "on"
		}
		Printf(LogInfo, "switch %s tethering %s\n", t.Type, state)
		
		needScan := t.Tethering
		
		if err := t.setProperty("Tethering", tether); err != nil {
			return err
		}
		
		if err := t.GetProperties(); err != nil {
			return err
		}

		if tether != t.Tethering {
			return fmt.Errorf("failed to set tethering")
		}

		if needScan {
			t.Scan()
		}
	}

	return nil
}

func (t *Technology) Scan() error {
	if t.Powered {
		if t.Type != "wifi" || (t.Type == "wifi" && !t.Tethering) {
			Printf(LogDebug, "start to scan %s services\n", t.Type)

			call := t.object.Call(TechnologyScan, 0)
	
			if call.Err != nil {
				return fmt.Errorf("Failed to scan '%s': %s", t.Type, call.Err.Error())
			}

			Printf(LogDebug, "%s scan complete\n", t.Type)
		}
	}

	return nil
}
