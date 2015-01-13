package httpstream

import (
	"github.com/ncdc/httpstream/api"
	"github.com/ncdc/httpstream/spdy"
)

func NewRequestUpgrader() api.RequestUpgrader {
	return spdy.NewRequestUpgrader()
}

func NewResponseUpgrader() api.ResponseUpgrader {
	return spdy.NewResponseUpgrader()
}
