package darwin

import (
	"context"
	"errors"
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

	conns    map[string]*conn
	connLock sync.Mutex

	advHandler ble.AdvHandler
	chConn     chan *connectResult
}

// NewDevice returns a BLE device.
func NewDevice(opts ...ble.Option) (*Device, error) {
	d := &Device{
		cm:     NewCentralMgr(),
		conns:  make(map[string]*conn),
		chConn: make(chan *connectResult),
	}

	err := d.cm.Start(d)
	if err != nil {
		return nil, err
	}

	return d, nil
}

// Option sets the options specified.
func (d *Device) Option(opts ...ble.Option) error {
	return nil
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
	d.cm.Stop()
	return nil
}

func (d *Device) closeConns() {
	d.connLock.Lock()
	defer d.connLock.Unlock()

	for _, c := range d.conns {
		c.Close()
	}
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

	go func() {
		<-c.Disconnected()
		d.delConn(c.addr)
	}()
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
	return errors.New("Not supported")
}
func (d *Device) RemoveAllServices() error {
	return errors.New("Not supported")
}
func (d *Device) SetServices(svcs []*ble.Service) error {
	return errors.New("Not supported")
}
func (d *Device) Advertise(ctx context.Context, adv ble.Advertisement) error {
	return errors.New("Not supported")
}
func (d *Device) AdvertiseNameAndServices(ctx context.Context, name string, uuids ...ble.UUID) error {
	return errors.New("Not supported")
}
func (d *Device) AdvertiseMfgData(ctx context.Context, id uint16, b []byte) error {
	return errors.New("Not supported")
}
func (d *Device) AdvertiseServiceData16(ctx context.Context, id uint16, b []byte) error {
	return errors.New("Not supported")
}
func (d *Device) AdvertiseIBeaconData(ctx context.Context, b []byte) error {
	return errors.New("Not supported")
}
func (d *Device) AdvertiseIBeacon(ctx context.Context, u ble.UUID, major, minor uint16, pwr int8) error {
	return errors.New("Not supported")
}
