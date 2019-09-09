package stackstorm

import (
	"fmt"
	"strings"
)

// UnknownStackStormError happens when the Stackstorm name [st2-name] in webhook url
// didn't match with any Stackstorm names in Configuration.
type UnknownStackStormError struct {
	correct []string
	wrong   string
}

func (e *UnknownStackStormError) Error() string {
	return fmt.Sprintf("Unknow stackstorm %s, please check your webhook url. It has to contain Stackstorm name like configuration (%s)",
		e.wrong, strings.Join(e.correct, ","))
}

// WrongBodyError happens when request body isn't in Alertmanager
// template.Data format.
type WrongBodyError struct {
	err error
}

func (e *WrongBodyError) Error() string {
	return fmt.Sprintf("Unable to parse request body because: %s. The request body should be in Alertmanager template.Data format.",
		e.err.Error())
}
