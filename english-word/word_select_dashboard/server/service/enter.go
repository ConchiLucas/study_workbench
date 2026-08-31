package service

import (
	"github.com/conchi/go-react-template/server/service/system"
)

var ServiceGroupApp = new(ServiceGroup)

type ServiceGroup struct {
	SystemServiceGroup system.ServiceGroup
}
