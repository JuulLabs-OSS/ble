package darwin

import "github.com/JuulLabs-OSS/ble"

type eventConnected struct {
	addr ble.Addr
	err  error
}

type eventSvcsDiscovered struct {
	err error
}

type eventChrsDiscovered struct {
	err error
}

type eventDscsDiscovered struct {
	err error
}

type eventChrRead struct {
	uuid ble.UUID
	err  error
}

type eventDscRead struct {
	err error
}

type eventChrWritten struct {
	err error
}

type eventDscWritten struct {
	err error
}

type eventNotifyChanged struct {
	err error
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
	close(evl.chrWritten)
	close(evl.dscRead)
	close(evl.dscWritten)
	close(evl.notifyChanged)
	close(evl.rssiRead)
	close(evl.disconnected)
}
