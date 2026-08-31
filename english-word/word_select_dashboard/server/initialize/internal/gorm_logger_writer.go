package internal

import (
	"github.com/conchi/go-react-template/server/config"
	"github.com/conchi/go-react-template/server/utils/gormsafe"
)

type Writer = gormsafe.Writer

func NewWriter(general config.GeneralDB) *Writer {
	return gormsafe.NewWriter(general)
}
