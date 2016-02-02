package email

import (
	"fmt"
)

// interfaceMapToBytesMap ...
func interfaceMapToBytesMap(m map[string]interface{}) (map[string][]byte, error) {
	s := make(map[string][]byte)
	for k, v := range m {
		if b, ok := v.([]byte); ok {
			s[k] = b
		} else {
			return map[string][]byte{}, fmt.Errorf("Unable to cast %T to []byte in map: %v", v, m)
		}
	}
	return s, nil
}

// interfaceToStringMap ...
func interfaceToStringMap(m map[string]interface{}) (map[string]string, error) {
	s := make(map[string]string)
	for k, v := range m {
		if b, ok := v.([]byte); ok {
			s[k] = string(b)
		} else {
			return map[string]string{}, fmt.Errorf("Unable to cast %T to []byte in map: %v", v, m)
		}
	}
	return s, nil
}
