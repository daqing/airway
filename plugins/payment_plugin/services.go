package payment_plugin

import (
	"strings"
	"time"

	"github.com/daqing/airway/lib/utils"
)

const PREFIX = "PMT"

func GenerateUUID() string {
	ts := time.Now().Format("20060102150405")

	rand := strings.ToUpper(utils.RandomHex(10))

	return strings.Join([]string{PREFIX, ts, "H", rand}, "")
}
