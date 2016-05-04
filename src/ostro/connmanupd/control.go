package connmanupd

import (
	"fmt"
	"strings"
	"time"
	"ostro/connman"
)

const (
	waitForServiceTime = 15	// sec
)

func (c *Conf) getService(typ string, name string) (*connman.Service, error) {
	mgr, merr := connman.NewManager()
	if merr != nil {
		return nil, merr
	}
	
	id, ierr := mgr.GetServiceId(typ, name)
	if ierr != nil {
		return nil, ierr
	}

	service, serr := connman.NewService(id)
	if serr != nil {
		return nil, serr
	}

	return service, nil
}


func (c *Conf) updateWiFiServiceOnServer(sf *WifiServiceFile) error {
	w := c.WiFi
	i4 := w.IPv4

	tech, terr := connman.NewTechnology("wifi")
	if terr != nil {
		return terr
	}

	powerIdentical := (tech.Powered && w.Enable) || (!tech.Powered && !w.Enable) 
	tetherIdentical := (sf.Tether && w.Tether) || (!sf.Tether && !w.Tether)
	i4Identical := (i4.Method == sf.IPv4.Method) &&
		(i4.Method != "manual" ||
		(i4.Method == "manual" &&
		i4.Address == sf.IPv4.Address &&
		i4.Netmask == sf.IPv4.Netmask &&
		i4.Gateway == sf.IPv4.Gateway))
	i6Identical := true
	securityIdentical := (sf.Security.Mode == w.Security.Mode) &&
		(sf.Security.Passphrase == w.Security.Passphrase)

	connman.Printf(connman.LogDebug,
		"wifi changes:\n   Power: %v\n   Tether: %v\n   IPv4: %v\n   IPv6: %v\n   Security: %v",
		!powerIdentical,
		!tetherIdentical,
		!i4Identical, !i6Identical,
		!securityIdentical)
		
	changed := !powerIdentical ||
		!tetherIdentical ||
		sf.Name != w.Name ||
		!i4Identical || !i6Identical ||
		sf.Security.Mode != w.Security.Mode ||
		sf.Security.Passphrase != w.Security.Passphrase
	
	if !changed {
		connman.Printf(connman.LogInfo, "no change in wifi configuration\n")
		return nil
	}

	
	if !powerIdentical {
		if err := tech.SetPowered(w.Enable);  err != nil {
			return err
		}
	}

	if !tech.Powered {
		return nil
	}
	
	if w.Tether {
		if w.Security.Mode != "psk" {
			return fmt.Errorf("invalid authentication mode '%s'", w.Security.Mode)
		}
		
		if err := tech.SetTetheringParameters(w.Name, w.Security.Passphrase); err != nil {
			return err
		}
		
		if err := tech.SetTethering(true); err != nil {
			return err
		}
	} else {
		tetherIdentical := (sf.Tether && w.Tether) || (!sf.Tether && !w.Tether)
		
		i4 := w.IPv4
		i4Identical := (i4.Method == sf.IPv4.Method) &&
			(i4.Method != "manual" ||
			(i4.Method == "manual" &&
			i4.Address == sf.IPv4.Address &&
			i4.Netmask == sf.IPv4.Netmask &&
			i4.Gateway == sf.IPv4.Gateway))
		i6Identical := true

		connman.Printf(connman.LogDebug, "wifi IPv4 identical: %v\n", i4Identical)
		
		changed := sf.Name != w.Name || !tetherIdentical ||
			!i4Identical || !i6Identical ||
			sf.Security.Mode != w.Security.Mode ||
			sf.Security.Passphrase != w.Security.Passphrase

		if !changed {
			connman.Printf(connman.LogDebug, "no change in wifi configuration\n")
		} else {
			if !tetherIdentical {
				if err := tech.SetTethering(false); err != nil {
					return err
				}
			}
		
			service, serr := c.waitForService("wifi", w.Name)
			if serr != nil {
				return fmt.Errorf("%s: %v", NoService, serr)
			}

			if err := service.Disconnect(); err != nil {
				if !strings.Contains(err.Error(), "Not connected") {
					connman.Printf(connman.LogInfo,
						"Attempt to disconnect failed: %v\n", err)
				}
			}

			if !i4Identical {			
				err := service.SetIPv4(i4.Method, i4.Address, i4.Netmask, i4.Gateway)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (c *Conf) connectWiFiService() error {
	w := c.WiFi
	
	if w.Connect == "no" {
		return nil
	}

	t, terr := connman.NewTechnology("wifi")
	if terr != nil {
		return terr
	}

	if t.Powered && !t.Tethering {
		s, serr := c.getService("wifi", w.Name)
		if serr != nil {
			return fmt.Errorf("%s: %v", NoService, serr)
		}

		auto := false
		if w.Connect == "auto" {
			auto = true
		}
		if err := s.SetAutoConnect(auto);  err != nil {
			return err
		}

		if w.Connect != "no" {
			if err := s.Connect(); err != nil {
				if !strings.Contains(err.Error(), "Already connected") {
					return err
				}
			}
		}
	}

	return nil
}


func (c *Conf) updateWiredServiceOnServer() error {
	e := c.Wired

	tech, terr := connman.NewTechnology("ethernet")
	if terr != nil {
		return terr
	}

	if err := tech.SetPowered(e.Enable);  err != nil {
		return err
	}

	if !tech.Powered {
		return nil
	}

	service, serr := c.waitForService("ethernet", e.Name)
	if serr != nil {
		return fmt.Errorf("%s: %v", NoService, serr)
	}

	if e.Connect == "no" {
		if err := service.Disconnect(); err != nil {
			if !strings.Contains(err.Error(), "Not connected") {
				connman.Printf(connman.LogInfo, "Attempt to disconnect failed: %v\n", err)
			}
		}
	}

	if i4 := e.IPv4; i4.identicalTo(service) {
		connman.Printf(connman.LogInfo, "no change in ethernet IP configuration\n")
	} else {
		if err := service.SetIPv4(i4.Method, i4.Address, i4.Netmask, i4.Gateway); err != nil {
			return err
		}
	}

	auto := false
	if e.Connect == "auto" {
		auto = true
	}
	if err := service.SetAutoConnect(auto); err != nil {
		return err
	}

	if e.Connect != "no" && (service.State != "ready" && service.State != "online" && service.State != "configuration") {
		
		if err := service.Connect();  err != nil {
			return err
		}
	}
	
	return nil
}

func (c *Conf) waitForService(typ, name string) (*connman.Service, error) {
	var serr error = nil
	var service *connman.Service = nil

	mgr, merr := connman.NewManager()
	if merr != nil {
		return nil, merr
	}

	for i := 0;  i < waitForServiceTime + 2;  i++ {
		if i > 1 {
			time.Sleep(1 * time.Second)
		}
		if i > 0 {
			if merr = mgr.GetServices(); merr != nil {
				return nil, merr
			}
		}
		if service, serr = c.getService(typ, name); serr == nil {
			return service, nil
		}
	}

	return nil, serr
}
