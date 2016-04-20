package main

import (
	"fmt"
	"flag"
	"time"
	"strings"
	"ostro/neard"
	"ostro/confs"
)

const (
	NeardOriginated = "Neard"
)

var (
	wifiPath = "/local/device/wifi"
)


type adapterEntry struct {
	path string
	enabled bool
	active bool
}

var (
	adapters = map[string]*adapterEntry{}
)

func main() {
	flag.Parse()

	neard.Initialize()

	if server := neard.NewServer([]string{"text", "wifi"}); server != nil {
		timer := time.NewTimer(1 * time.Hour)
		timer.Stop()

		for {
			select {
			case ev := <- server.Evchan:
				eventHandler(server, ev, timer)
			case <- timer.C:
				if name, ad := getAdapter(); ad != nil && !ad.active {
					server.Activate(name)
				}
			}
		}
	}
}

func eventHandler(server *neard.Server, ev neard.Message, timer *time.Timer) {
	switch ev.Id {

	case neard.Adapter:
		adev := ev.AdapterEvent()
		confs.Debugf("adapter event: %v", adev)
		updateAdapter(&adev)
		if _, ad := getAdapter(); ad != nil {
			if !ad.active {
				timer.Reset(2 * time.Second)
			}
		}
		
	case neard.State:
		state := ev.String()
		confs.Debugf("state changed to '%s'", state)
		
	case neard.Error:
		err := ev.Error()
		confs.Errorf("%v", err)

	case neard.WlanHandover:
		handleWlanHandoverRequest(ev.Credentials())

	case neard.ConfigData:
		handleConfigData(ev.String())

	case neard.StartApplication:

	default:
		confs.Debugf("received unsupported event %d", ev.Id)
	}
}

func updateAdapter(ev *neard.AdapterEvent) {
	if ev.Event == "destroyed" {
		delete(adapters, ev.Name)
	} else {
		if ad := adapters[ev.Name]; ad == nil {
			ad = &adapterEntry{path : ev.Path, enabled: ev.Enabled, active: ev.Active}
			adapters[ev.Name] = ad
		} else {
			ad.enabled = ev.Enabled
			ad.active = ev.Active
		}
	}
}

func getAdapter() (string, *adapterEntry) {
	for name, entry := range adapters {
		return name, entry
	}

	return "", nil
}

func handleWlanHandoverRequest(cred neard.Credentials) {
	var frag *confs.ConfFragment
	var err error

	jsn := fmt.Sprintf("{\"Enable\":true,\"Name\":\"%s\",\"IPv4\":\"DHCP\",\"Mode\":\"EndPoint\",",cred.SSID)
	jsn += fmt.Sprintf("\"Security\":{\"Mode\":\"%s\",\"PreSharedKey\":\"%s\"}}", cred.AuthType, cred.Key)


	if frag, err = confs.NewConfFragment(wifiPath, NeardOriginated, []byte(jsn)); err != nil {
		confs.Errorf("Failed to make ConfFragment (path:'%s', data:'%s'): %v", wifiPath, jsn, err)
		return
	}
	if ferr := frag.WriteDropZone(); ferr != nil {
		confs.Errorf("Failed to write ConfFragement to DropZone: %v", ferr)
		return
	}

	confs.Infof("%s config data written to %s", frag.Type(), frag.Path())
}

func handleConfigData(data string) {
	var (
		frag *confs.ConfFragment
		path, conf string
		err error
		ok bool
	)

	if path, conf, ok = parseConfigData(data); !ok {
		confs.Errorf("Failed to parse configuration data: '%s'", data)
		return
	}
	if frag, err = confs.NewConfFragment(path, NeardOriginated, []byte(conf)); err != nil {
		confs.Errorf("Failed to make ConfFragment (path:'%s', data:'%s'): %v", path, conf, err)
		return
	}
	if ferr := frag.WriteDropZone(); ferr != nil {
		confs.Errorf("Failed to write ConfFragement to DropZone: %v", ferr)
		return
	}

	confs.Infof("%s config data written to %s", frag.Type(), frag.Path())
}

func parseConfigData(data string) (string, string, bool) {
	ws := " \t\n\r"
	arr := strings.Split(data, "=")

	if len(arr) > 1 {
		path := strings.Trim(arr[0], ws)
		conf := strings.Trim(strings.Join(arr[1:], "="), ws)

		if len(path) > 0 && len(conf) > 0 {
			return path, conf, true
		}
	}

	return "", "", false
}

	
