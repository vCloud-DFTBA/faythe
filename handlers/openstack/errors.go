package openstack

import (
	"fmt"
	"strings"
)

// UnknownOpenStackError happends when the OpenStack name [ops-name] in webhook url
// didn't match with any OpenStack names in Configuration.
type UnknownOpenStackError struct {
	correct []string
	wrong   string
}

func (e *UnknownOpenStackError) Error() string {
	return fmt.Sprintf("Unknow openstack %s, please check your webhook url. It has to contain OpenStack name like configuration (%s)",
		e.wrong, strings.Join(e.correct, ", "))
}
