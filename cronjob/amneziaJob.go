package cronjob

import (
	"github.com/zamibd/MPanel/logger"
	"github.com/zamibd/MPanel/service"
)

// AmneziaJob runs periodically to:
//  1. Sync traffic counters from the wg interface into the DB.
//  2. Disable peers that have exceeded their volume or expiry date.
type AmneziaJob struct{}

func NewAmneziaJob() *AmneziaJob { return &AmneziaJob{} }

func (j *AmneziaJob) Run() {
	svc := service.GetAmneziaService()

	if err := svc.SyncTraffic(); err != nil {
		logger.Warning("amnezia traffic sync failed: ", err)
	}

	if err := svc.DepletePeers(); err != nil {
		logger.Warning("amnezia deplete peers failed: ", err)
	}
}
