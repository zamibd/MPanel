package cronjob

import "github.com/zamibd/MPanel/logger"

type WALCheckpointJob struct{}

func NewWALCheckpointJob() *WALCheckpointJob {
	return &WALCheckpointJob{}
}

// Run is a no-op for PostgreSQL — WAL checkpointing is managed automatically
// by the PostgreSQL server and cannot be triggered via a client PRAGMA.
func (s *WALCheckpointJob) Run() {
	logger.Debug("WALCheckpointJob: skipped (PostgreSQL manages WAL automatically)")
}
