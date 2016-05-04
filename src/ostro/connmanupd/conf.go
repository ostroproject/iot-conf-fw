package connmanupd

import (
	"fmt"
	"os"
	"encoding/json"
	"ostro/connman"
)

const (
	NoService = "no service"
)

type IP4 struct {
	Method string          // "off" | "dhcp" | "manual"
	Address string
	Netmask string
	Gateway string
}

func (ip4 IP4) String() string {
	if ip4.Method == "off" || ip4.Method == "dhcp" {
		return ip4.Method
	}

	if ip4.Method == "manual" {
		return fmt.Sprintf("%s/%s/%s", ip4.Address, ip4.Netmask, ip4.Gateway)
	}

	return ""
}

func (ip4 IP4) identicalTo(s *connman.Service) bool {
	if ip4.Method != s.IPv4.Method {
		return false
	}

	if ip4.Method == "manual" {
		if ip4.Address != s.IPv4.Address || ip4.Netmask != s.IPv4.Netmask || ip4.Gateway != s.IPv4.Gateway {
			return false
		}
	}
	
	return true
}


type Security struct {
	Mode string            // "none" | "wep" | "psk" | "ieee8021x"
	Passphrase string
}

func (s Security) String() string {
	if s.Mode != "wep" && s.Mode != "psk" {
		return "none"
	}

	return fmt.Sprintf("%s\nPassphrase=%s", s.Mode, s.Passphrase)
}

type Ethernet struct {
	Valid bool
	Enable bool
	Name string            // Service name
	IPv4 IP4
	Connect string
}

const ethernetFmt = `
   Enable   %v
   Name     %s
   IPv4     %v
   Connect  %s
`
func (e Ethernet) String() string {
	var str string = "\n   <not set>\n"

	if e.Valid {
		str =  fmt.Sprintf(ethernetFmt, e.Enable, e.Name, e.IPv4, e.Connect)
	}

	return str
}


type Wifi struct {
	Valid bool
	Enable bool
	Name string            // Network name (SSID)
	Tether bool
	IPv4 IP4
	Security Security
	Connect string         // "auto" | "yes" | "no"
	Hidden bool
}

const wifiFmt = `
   Enable   %v
   Name     %s
   Tether   %v
   IPv4     %v
   Security %s/'%s'
   Connect  %s
   Hidden   %v
`

func (w Wifi) String() string {
	var str string = "\n   <not set>\n"

	if w.Valid {
		str =  fmt.Sprintf(wifiFmt, w.Enable, w.Name, w.Tether, w.IPv4,
			w.Security.Mode, w.Security.Passphrase, w.Connect, w.Hidden)
	}

	return str
}

type Conf struct {
	Wired Ethernet
	WiFi Wifi
}

func NewConf(path string) (*Conf, error) {
	buf, err := readFile(path)
	if err != nil {
		return nil, err
	}

	conf := &Conf{}
	err = json.Unmarshal(buf, conf)

	if err != nil {
		return nil, fmt.Errorf("Invalid JSON in file '%s': %v\n", path, err)
	}

	if !conf.Wired.Enable {
		connman.AddDisabledTechnology("ethernet")
	}
	if !conf.WiFi.Enable {
		connman.AddDisabledTechnology("wifi")
	}

	connman.Printf(connman.LogDebug, "conf: %+v\n", conf)

	return conf, nil
}

// this is doing the same as the ioutil.ReadFile()
// but fails in case the file is biger than the allowed
// 64Kbyte maximum
func readFile(path string) ([]byte, error) {
	file, ferr := os.Open(path)
	if ferr != nil {
		return []byte{}, ferr
	}

	info, ierr := file.Stat()
	if ierr != nil {
		return []byte{}, ierr
	}

	if info.Size() > 65536 {
		return []byte{}, fmt.Errorf("File '%s' length %d exceeds the allowed max (64Kbyte)\n", path, info.Size())
	}
	
	size := int(info.Size())
	buf := make([]byte, size)

	count, err := file.Read(buf)
	if err != nil {
		return []byte{}, fmt.Errorf("Failed to read file '%s': %v\n", path, err)
	}
	if count != size {
		return []byte{}, fmt.Errorf("Failed to read file '%s': partial read (requested %d, read %d)\n", path, size, count)
	}

	return buf, nil
}

func (c *Conf) Install(dir string) map[string]error {
	err := error(nil)
	errors := make(map[string]error)
	
	if err = os.MkdirAll(dir, 0755); err != nil {
		errors["directory"] = fmt.Errorf("failed to create directory '%s': %v", dir, err)
		return errors
	}

	manager, err := connman.NewManager()
	if err != nil {
		errors["manager"] = err
		return errors
	}
	
	for _, t := range manager.Technologies {
		err = error(nil)
		
		switch t.Type {
			
		case "wifi":
			if c.WiFi.Valid {
				servFile := c.readWiFiServiceFile(dir)

				connman.Printf(connman.LogDebug, "Service file: %v\n", servFile)
				
				if err = c.removeWiFiServiceFile(dir); err != nil {
					break
				}
				
				if err = c.updateWiFiServiceOnServer(servFile); err != nil {
					break
				}
				
				if err = c.writeWiFiServiceFile(dir); err != nil {
					break
				}
				
				if err = c.connectWiFiService(); err != nil {
					break
				}
			}

		case "ethernet":
			if c.Wired.Valid {
				if err = c.writeWiredServiceFile(dir); err != nil {
					break
				}
				if err = c.updateWiredServiceOnServer(); err != nil {
					break
				}
			}
		}

		if err != nil {
			errors[t.Type] = err
		}
	}

	return errors
}

func (c Conf) String() {
	fmt.Printf("%v\n%v", c.Wired, c.WiFi)
}
