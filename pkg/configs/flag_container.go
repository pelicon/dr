package configs

import (
	"sync"
)

const (
	AdapterKubeApiserver AdapterName = "ADAPTER_KUBE_APISERVER"
)

type AdapterName string

var (
	flagContainer  *flags
	flagUpdateLock = &sync.Mutex{}
)

type flags struct {
	TransportAdapter AdapterName
}

func UpdateFlagToContainer(transportAdapter AdapterName) {
	flagUpdateLock.Lock()
	defer flagUpdateLock.Unlock()

	flagContainer = &flags{
		TransportAdapter: transportAdapter,
	}
}

func GetFlags() *flags {
	return flagContainer
}
