package stackstorm

import (
	"fmt"
	"strings"
)

// UnknownStackStormError happends when the Stackstorm name [st2-name] in webhook url
// didn't match with any Stackstorm names in Configuration.
type UnknownStackStormError struct {
	correct []string
	wrong   string
}

func (e *UnknownStackStormError) Error() string {
	return fmt.Sprintf("Unknow stackstorm %s, please check your webhook url. It has to contain Stackstorm name like configuration (%s)",
		e.wrong, strings.Join(e.correct, ","))
}
