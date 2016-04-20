package neard

import (
	"github.com/godbus/dbus"
	"ostro/confs"
)

const (
	agentService = "org.ostro.neard"
	agentPath = "/org/ostro/neard"
	handoverInterface = "org.neard.HandoverAgent"
	ndefInterface = "org.neard.NDEFAgent"

	RegisterHandoverAgent = "org.neard.AgentManager.RegisterHandoverAgent"
	UnregisterHandoverAgent = "org.neard.AgentManager.UnregisterHandoverAgent"
	RegisterNDEFAgent = "org.neard.AgentManager.RegisterNDEFAgent"
	UnregisterNDEFAgent = "org.neard.AgentManager.UnregisterNDEFAgent"
)

type handoverAgent struct {
	srv *server
	carriers map[string]bool
}

type ndefAgent struct {
	srv *server
	types map[string]bool
}



type agent struct {
	tagType map[string]bool
	handover *handoverAgent
	ndef *ndefAgent
}



var agnt *agent = nil


func newAgent(srv *server, tagType []string) bool {
	if agnt == nil {
		if reply, err := srv.conn.RequestName(agentService, dbus.NameFlagDoNotQueue); err != nil {
			confs.Errorf("Failed to obtain D-Bus name '%s': %v", agentService, err)
			return false
		} else {
			if reply != dbus.RequestNameReplyPrimaryOwner {
				confs.Errorf("D-Bus name '%s' already taken", agentService)
				return false
			}
		}
		
		agnt = &agent{tagType: map[string]bool{}, handover: nil, ndef: nil}
	}

	for _, t := range tagType {
		if !agnt.tagType[t] {
			switch t {
			case "wifi", "bluetooth":
				if agnt.handover == nil {
					if agnt.handover = newHandoverAgent(srv); agnt.handover == nil {
						return false
					}
				}
				if !agnt.handover.registerCarrier(srv, t) {
					return false
				}
			case "text", "uri":
				if agnt.ndef == nil {
					if agnt.ndef = newNdefAgent(srv); agnt.ndef == nil {
						return false
					}
				}
				if !agnt.ndef.registerType(srv, t) {
					return false
				}
			default:
				confs.Errorf("unsupported tag type '%s'", t)
			}

			agnt.tagType[t] = true
		}
	}

	return true
}

func newHandoverAgent(srv *server) *handoverAgent {
	a := handoverAgent{srv: srv, carriers: map[string]bool{}}

	srv.conn.Export(a, agentPath, handoverInterface)

	confs.Debugf("Handover agent created")

	return &a
}

func (me *handoverAgent) registerCarrier(srv *server, carrier string) bool {
	if carrier != "wifi" && carrier != "bluetooth" {
		confs.Errorf("unsupported carrier '%s'", carrier)
		return false
	}

	if !me.carriers[carrier] {
		objPath := dbus.ObjectPath(agentPath)
		if err := srv.neardObject.Call(RegisterHandoverAgent, 0, objPath, carrier).Err; err != nil {
			confs.Errorf("can't register Handover carrier '%s': %v", carrier, err)
			return false
		}
		me.carriers[carrier] = true
		confs.Debugf("Handover agent for '%s' carrier successfully registered", carrier)
	}
	
	return true
}

func (me handoverAgent) RequestOOB(data map[string]dbus.Variant) *dbus.Error {
	confs.Debugf("RequestOOB: %v", data)
	return nil
}

func (me handoverAgent) PushOOB(data map[string]dbus.Variant) *dbus.Error {
	confs.Debugf("PushOOB called")
	if variant, exists := data["WSC"]; exists {
		if wsc, ok := variant.Value().([]byte); ok {
			if creds, ok := WSCParser(wsc); ok {
				me.srv.sendWlanHandoverRequest(creds)
			}
		} else {
			confs.Debugf("Couldn't cast WSC record to byte array")
		}
	}
	return nil
}

func (me handoverAgent) Release() *dbus.Error {
	confs.Debugf("Handover agent Release")
	return nil
}


func newNdefAgent(srv *server) *ndefAgent {
	a := ndefAgent{srv: srv, types: map[string]bool{}}

	srv.conn.Export(a, agentPath, ndefInterface)

	confs.Debugf("NDEF agent created")

	return &a
}

func (me *ndefAgent) registerType(srv *server, typ string) bool {
	var t string

	switch typ {
	case "text":
		t = "Text"
	case "uri":
		t = "URI"
	default:
		confs.Errorf("Unsupported ndef type '%s'", typ)
		return false
	}

	if !me.types[t] {
		objPath := dbus.ObjectPath(agentPath)
		if err := srv.neardObject.Call(RegisterNDEFAgent, 0, objPath, t).Err; err != nil {
			confs.Errorf("can't register NDEF type '%s': %v", t, err)
			return false
		}
		me.types[t] = true
		confs.Debugf("NDEF agent for '%s' type successfully registered", t)
	}

	return true
}


func (me ndefAgent) GetNDEF(data map[string]dbus.Variant) *dbus.Error {
	confs.Debugf("GetNdef: %v", data)
	return nil
}

func (me ndefAgent) Release() *dbus.Error {
	confs.Debugf("NDEF agent Release")
	return nil
}
