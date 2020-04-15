package darwin

import (
	"github.com/JuulLabs-OSS/ble"
	"github.com/raff/goble/xpc"
)

/*
type Advertisement interface {
	X LocalName() string
	X ManufacturerData() []byte
	X ServiceData() []ServiceData
	X Services() []UUID
	- OverflowService() []UUID
	X TxPowerLevel() int
	X Connectable() bool
	- SolicitedService() []UUID

	X RSSI() int
	Addr() Addr
}
*/

type adv struct {
	localName   string
	rssi        int
	mfgData     []byte
	powerLevel  int
	connectable bool
	svcUUIDs    []ble.UUID
	svcData     []ble.ServiceData
	peerUUID    ble.Addr

	args xpc.Dict
	ad   xpc.Dict
}

func (a *adv) LocalName() string {
	return a.localName
}

func (a *adv) ManufacturerData() []byte {
	return a.mfgData
}

func (a *adv) ServiceData() []ble.ServiceData {
	return a.svcData
}

func (a *adv) Services() []ble.UUID {
	return a.svcUUIDs
}

func (a *adv) OverflowService() []ble.UUID {
	return nil // TODO
}

func (a *adv) TxPowerLevel() int {
	return a.powerLevel
}

func (a *adv) SolicitedService() []ble.UUID {
	return nil // TODO
}

func (a *adv) Connectable() bool {
	return a.connectable
}

func (a *adv) RSSI() int {
	return a.rssi
}

func (a *adv) Addr() ble.Addr {
	return a.peerUUID
}
