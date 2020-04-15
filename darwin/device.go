package darwin

import (
	"context"
	"fmt"

	"github.com/JuulLabs-OSS/ble"

	"sync"
)

type connectResult struct {
	conn *conn
	err  error
}

// Device is either a Peripheral or Central device.
type Device struct {
	cm *CentralMgr

	role int // 1: peripheralManager (server), 0: centralManager (client)

	rspc chan msg

	conns    map[string]*conn
	connLock sync.Mutex

	// Only used in client/centralManager implementation
	advHandler ble.AdvHandler
	chConn     chan *connectResult

	// Only used in server/peripheralManager implementation
	chars map[int]*ble.Characteristic
	base  int
}

// NewDevice returns a BLE device.
func NewDevice(opts ...ble.Option) (*Device, error) {
	d := &Device{
		rspc:   make(chan msg),
		conns:  make(map[string]*conn),
		chConn: make(chan *connectResult),
		chars:  make(map[int]*ble.Characteristic),
		base:   1,
	}
	if err := d.Option(opts...); err != nil {
		return nil, err
	}
	d.cm = NewCentralMgr(d)

	// Make sure CoreBluetooth is running.
	err := StartBTLoop()
	if err != nil {
		return nil, err
	}

	return d, nil
}

// Option sets the options specified.
func (d *Device) Option(opts ...ble.Option) error {
	var err error
	for _, opt := range opts {
		err = opt(d)
	}
	return err
}

// Scan ...
func (d *Device) Scan(ctx context.Context, allowDup bool, h ble.AdvHandler) error {
	d.advHandler = h

	d.cm.Scan(!allowDup)

	<-ctx.Done()
	d.cm.StopScan()

	return ctx.Err()
}

// Dial ...
func (d *Device) Dial(ctx context.Context, a ble.Addr) (ble.Client, error) {
	err := d.cm.Connect(a)
	if err != nil {
		return nil, err
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-d.chConn:
		if res.err != nil {
			return nil, res.err
		} else {
			res.conn.SetContext(ctx)
			return NewClient(d.cm, res.conn)
		}
	}
}

// Stop ...
func (d *Device) Stop() error {
	return nil
}

func (d *Device) findConn(a ble.Addr) *conn {
	d.connLock.Lock()
	defer d.connLock.Unlock()

	return d.conns[a.String()]
}

func (d *Device) connectSuccess(a ble.Addr) {
	d.connLock.Lock()
	defer d.connLock.Unlock()

	fail := func(err error) {
		d.chConn <- &connectResult{
			err: err,
		}
	}

	if d.conns[a.String()] != nil {
		fail(fmt.Errorf("failed to add connection: already exists: addr=%s", a.String()))
		return
	}

	txMTU, err := d.cm.attMTU(a)
	if err != nil {
		fail(fmt.Errorf("failed to add connection: %v", err))
		return
	}

	c := newConn(d, a, txMTU)
	d.conns[a.String()] = c
	d.chConn <- &connectResult{
		conn: c,
	}
}

func (d *Device) connectFail(err error) {
	d.chConn <- &connectResult{
		err: err,
	}
}

func (d *Device) delConn(a ble.Addr) {
	d.connLock.Lock()
	defer d.connLock.Unlock()

	delete(d.conns, a.String())
}

func (d *Device) AddService(svc *ble.Service) error {
	return ble.ErrNotImplemented
}
func (d *Device) RemoveAllServices() error {
	return ble.ErrNotImplemented
}
func (d *Device) SetServices(svcs []*ble.Service) error {
	return ble.ErrNotImplemented
}
func (d *Device) Advertise(ctx context.Context, adv ble.Advertisement) error {
	return ble.ErrNotImplemented
}
func (d *Device) AdvertiseNameAndServices(ctx context.Context, name string, uuids ...ble.UUID) error {
	return ble.ErrNotImplemented
}
func (d *Device) AdvertiseMfgData(ctx context.Context, id uint16, b []byte) error {
	return ble.ErrNotImplemented
}
func (d *Device) AdvertiseServiceData16(ctx context.Context, id uint16, b []byte) error {
	return ble.ErrNotImplemented
}
func (d *Device) AdvertiseIBeaconData(ctx context.Context, b []byte) error {
	return ble.ErrNotImplemented
}
func (d *Device) AdvertiseIBeacon(ctx context.Context, u ble.UUID, major, minor uint16, pwr int8) error {
	return ble.ErrNotImplemented
}
