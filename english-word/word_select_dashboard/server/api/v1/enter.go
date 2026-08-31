package v1

import (
	"github.com/conchi/go-react-template/server/api/v1/system"
)

var ApiGroupApp = new(ApiGroup)

type ApiGroup struct {
	SystemApiGroup system.ApiGroup
}
