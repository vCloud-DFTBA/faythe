package stackstorm

import (
	"fmt"
	"strings"
)

// UnknownStackStormError happends when the OpenStack name [ops-name] in webhook url
// didn't match with any OpenStack names in Configuration.
type UnknownStackStormError struct {
	correct []string
	wrong   string
}

func (e *UnknownStackStormError) Error() string {
	return fmt.Sprintf("Unknow stackstorm %s, please check your webhook url. It has to contain OpenStack name like configuration (%s)",
		e.wrong, strings.Join(e.correct, ", "))
}
