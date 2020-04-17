package darwin

import "github.com/JuulLabs-OSS/ble"

type discoveredSvc struct {
	id   uintptr
	uuid ble.UUID
}

type discoveredChr struct {
	id         uintptr
	uuid       ble.UUID
	properties ble.Property
}

type discoveredDsc struct {
	id   uintptr
	uuid ble.UUID
}

type eventConnected struct {
	addr ble.Addr
	err  error
}

type eventSvcsDiscovered struct {
	svcs []discoveredSvc
	err  error
}

type eventChrsDiscovered struct {
	chrs []discoveredChr
	err  error
}

type eventDscsDiscovered struct {
	dscs []discoveredDsc
	err  error
}

type eventChrRead struct {
	uuid  ble.UUID
	value []byte
	err   error
}

type eventChrWritten struct {
	uuid ble.UUID
	err  error
}

type eventDscRead struct {
	uuid  ble.UUID
	value []byte
	err   error
}

type eventDscWritten struct {
	uuid ble.UUID
	err  error
}

type eventNotifyChanged struct {
	uuid    ble.UUID
	enabled bool
	err     error
}

type eventRSSIRead struct {
	rssi int
	err  error
}

type eventDisconnected struct {
	reason int
}

type centralEventListener struct {
	svcsDiscovered chan *eventSvcsDiscovered
	chrsDiscovered chan *eventChrsDiscovered
	dscsDiscovered chan *eventDscsDiscovered
	chrRead        chan *eventChrRead
	chrWritten     chan *eventChrWritten
	dscRead        chan *eventDscRead
	dscWritten     chan *eventDscWritten
	notifyChanged  chan *eventNotifyChanged
	rssiRead       chan *eventRSSIRead
	disconnected   chan *eventDisconnected
}

func newCentralEventListener() *centralEventListener {
	return &centralEventListener{
		svcsDiscovered: make(chan *eventSvcsDiscovered),
		chrsDiscovered: make(chan *eventChrsDiscovered),
		dscsDiscovered: make(chan *eventDscsDiscovered),
		chrRead:        make(chan *eventChrRead),
		chrWritten:     make(chan *eventChrWritten),
		dscRead:        make(chan *eventDscRead),
		dscWritten:     make(chan *eventDscWritten),
		notifyChanged:  make(chan *eventNotifyChanged),
		rssiRead:       make(chan *eventRSSIRead),
		disconnected:   make(chan *eventDisconnected),
	}
}

func (evl *centralEventListener) Close() {
	close(evl.svcsDiscovered)
	close(evl.chrsDiscovered)
	close(evl.dscsDiscovered)
	close(evl.chrRead)
	close(evl.chrWritten)
	close(evl.dscRead)
	close(evl.dscWritten)
	close(evl.notifyChanged)
	close(evl.rssiRead)
	close(evl.disconnected)
}
