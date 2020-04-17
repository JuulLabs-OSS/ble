package darwin

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/JuulLabs-OSS/ble"
)

func newConn(d *Device, a ble.Addr, txMTU int) *conn {
	c := &conn{
		dev:   d,
		rxMTU: 23,
		txMTU: txMTU,
		addr:  a,
		done:  make(chan struct{}),

		notifiers: make(map[uint16]ble.Notifier),
		subs:      make(map[string]*sub),
		chrReads:  make(map[string]chan *eventChrRead),

		rspc: make(chan msg),
		evl:  newCentralEventListener(),
	}

	go func() {
		<-c.evl.disconnected
		close(c.done)
	}()

	return c
}

type conn struct {
	sync.RWMutex

	dev   *Device
	ctx   context.Context
	rxMTU int
	txMTU int
	addr  ble.Addr
	done  chan struct{}

	rspc chan msg
	evl  *centralEventListener

	notifiers map[uint16]ble.Notifier // central connection only

	subs     map[string]*sub
	chrReads map[string](chan *eventChrRead)
}

func (c *conn) Context() context.Context {
	return c.ctx
}

func (c *conn) SetContext(ctx context.Context) {
	c.ctx = ctx
}

func (c *conn) LocalAddr() ble.Addr {
	// return c.dev.Address()
	return c.addr // FIXME
}

func (c *conn) RemoteAddr() ble.Addr {
	return c.addr
}

func (c *conn) RxMTU() int {
	return c.rxMTU
}

func (c *conn) SetRxMTU(mtu int) {
	c.rxMTU = mtu
}

func (c *conn) TxMTU() int {
	return c.txMTU
}

func (c *conn) SetTxMTU(mtu int) {
	c.Lock()
	c.txMTU = mtu
	c.Unlock()
}

func (c *conn) Read(b []byte) (int, error) {
	return 0, nil
}
func (c *conn) Write(b []byte) (int, error) {
	return 0, nil
}

func (c *conn) Close() error {
	c.evl.Close()
	return nil
}

// Disconnected returns a receiving channel, which is closed when the connection disconnects.
func (c *conn) Disconnected() <-chan struct{} {
	return c.done
}

func (c *conn) processChrRead(ev *eventChrRead) {
	c.RLock()
	defer c.RUnlock()

	uuidStr := uuidStrWithDashes(ev.uuid.String())
	found := false

	ch := c.chrReads[uuidStr]
	if ch != nil {
		ch <- ev
		found = true
	}

	s := c.subs[uuidStr]
	if s != nil {
		s.fn(ev.value)
		found = true
	}

	if !found {
		log.Printf("received characteristic read response without corresponding request: uuid=%s", uuidStr)
	}
}

func (c *conn) addChrReader(char *ble.Characteristic) (chan *eventChrRead, error) {
	uuidStr := uuidStrWithDashes(char.UUID.String())

	c.Lock()
	defer c.Unlock()

	if c.chrReads[uuidStr] != nil {
		return nil, fmt.Errorf("cannot read from the same attribute twice: uuid=%s", uuidStr)
	}

	ch := make(chan *eventChrRead)
	c.chrReads[uuidStr] = ch

	return ch, nil
}

func (c *conn) delChrReader(char *ble.Characteristic) {
	uuidStr := uuidStrWithDashes(char.UUID.String())

	c.Lock()
	defer c.Unlock()

	delete(c.chrReads, uuidStr)
}

func (c *conn) addSub(char *ble.Characteristic, fn ble.NotificationHandler) {
	uuidStr := uuidStrWithDashes(char.UUID.String())

	c.Lock()
	defer c.Unlock()

	// It feels like we should return an error if we are already subscribed to
	// this characteristic.  Just quietly overwrite the existing handler to
	// preserve backwards compatibility.

	c.subs[uuidStr] = &sub{
		fn:   fn,
		char: char,
	}
}

func (c *conn) delSub(char *ble.Characteristic) {
	uuidStr := uuidStrWithDashes(char.UUID.String())

	c.Lock()
	defer c.Unlock()

	delete(c.subs, uuidStr)
}
