package neard

import (
	"ostro/confs"
)

const (
	TLV_INVALID          = int(0)
	TLV_AUTH_TYPE        = 0x1003
	TLV_CREDENTIAL       = 0x100e
	TLV_ENC_TYPE         = 0x100f
	TLV_PASS_ID          = 0x1012
	TLV_MAC_ADDR         = 0x1020
	TLV_NETWORK_IDX      = 0x1026
	TLV_KEY              = 0x1027
	TLV_SSID             = 0x1045
	TLV_VENDOR_EXTENSION = 0x1049
	TLV_VERSION          = 0x104a
	TLV_X509_CERT        = 0x104c
)


var (
	authTypes = map[int]string{
		0x01: "None",
		0x02: "WPA-Private",
		0x04: "WEP-64",
		0x08: "WEP-Enterprise",
		0x10: "WPA2-Enterprise",
		0x20: "WPA2-Private"}
	encTypes = map[int]string{
		1: "None",
		2: "WEP",
		3: "TKIP",
		4: "AES"}
)

func WSCParser(data []byte) (*Credentials, bool) {
	creds := Credentials{}

	if !tlvDecode(data, &creds) {
		return nil, false
	}
	if creds.SSID == "" || creds.AuthType == "" || creds.Key == "" {
		confs.Errorf("Incomplete WSC data")
		return nil, false
	}
	
	return &creds, true
}

func tlvDecode(data []byte, creds *Credentials) bool {
	var (
		i int = 0
		size = len(data)
		id int
		length int
	)

	for i < size {
		if size - i < 4 {
			confs.Errorf("broken WSC data: not enough room for TLV header")
			return false
		}

		id = int(data[i]) * 256 + int(data[i+1])
		length = int(data[i+2]) * 256 + int(data[i+3])

		if length < 1 {
			confs.Errorf("broken WSC data: TLV length < 1B")
			return false
		}

		i += 4

		if size - i < length {
			confs.Errorf("broken WSC data: not enough room for TLV value")
			return false
		}

		switch id {
		case TLV_CREDENTIAL:
			if !tlvDecode(data[i:i+length], creds) {
				return false
			}
		case TLV_NETWORK_IDX:
			if length != 1 {
				confs.Errorf("broken WSC data: NETWORK_IDX length is not 1")
				return false
			}
			creds.Network = int(data[i])
		case TLV_SSID:
			creds.SSID = string(data[i:i+length])
		case TLV_AUTH_TYPE:
			if length != 2 {
				confs.Errorf("broken WSC data: AUTH_TYPE length is not 2")
				return false
			}
			if atyp, exists := authTypes[int(data[i])*256 + int(data[i+1])]; !exists {
				confs.Errorf("broken WSC data: unsupported AUTH_TYPE");
				return false
			} else {
				creds.AuthType = atyp
			}
		case TLV_KEY:
			creds.Key = string(data[i:i+length])
		case TLV_ENC_TYPE:
			if length != 2 {
				confs.Errorf("broken WSC data: ENC_TYPE length is not 2")
				return false
			}
			if etyp, exists := encTypes[int(data[i])*256 + int(data[i+1])]; exists {
				creds.EncType = etyp
			}
		case TLV_X509_CERT:
			creds.X509Cert = string(data[i:i+length])
		default:
		}

		i += length
	}

	return true
}
