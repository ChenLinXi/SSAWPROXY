package proxy

import "fmt"

type protocolError string

func (pe protocolError) Error() string {
	return fmt.Sprintf("proxy: %s (possible server error or unsupported concurrent read by application)", string(pe))
}
