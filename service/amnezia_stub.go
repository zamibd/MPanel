//go:build !linux
// +build !linux

package service

// amnezia_stub.go – Non-Linux stub for AmneziaWG interface management.
// AmneziaWG only works on Linux (requires kernel module or userspace TUN).
// These stubs allow the project to compile on macOS/Windows for development.

import "fmt"

import "github.com/zamibd/MPanel/database/model"

func (s *AmneziaService) StartInterface() error { return fmt.Errorf("AmneziaWG requires Linux") }
func (s *AmneziaService) StopInterface() error  { return nil }
func (s *AmneziaService) IsRunning() bool       { return false }
func (s *AmneziaService) SyncTraffic() error    { return nil }

func (s *AmneziaService) liveAddPeer(_ *model.AmneziaPeer) error { return nil }
func (s *AmneziaService) liveRemovePeer(_ string) error          { return nil }
