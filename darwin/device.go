package darwin

import (
	"context"
	"errors"
	"fmt"

	"github.com/JuulLabs-OSS/ble"
	"github.com/JuulLabs-OSS/cbgo"

	"sync"
)

type connectResult struct {
	conn *conn
	err  error
}

// Device is either a Peripheral or Central device.
type Device struct {
	cbgo.CentralManagerDelegateBase

	cm cbgo.CentralManager

	conns    map[string]*conn
	connLock sync.Mutex

	advHandler ble.AdvHandler
	chConn     chan *connectResult
	chState    chan struct{}
}

// NewDevice returns a BLE device.
func NewDevice(opts ...ble.Option) (*Device, error) {
	d := &Device{
		cm:      cbgo.NewCentralManager(nil),
		conns:   make(map[string]*conn),
		chConn:  make(chan *connectResult),
		chState: make(chan struct{}),
	}

	d.cm.SetDelegate(d)
	<-d.chState
	if d.cm.State() != cbgo.ManagerStatePoweredOn {
		return nil, fmt.Errorf("central manager has invalid state: have=%d want=%d: is Bluetooth turned on?",
			d.cm.State(), cbgo.ManagerStatePoweredOn)
	}

	go func() {
		for {
			_, ok := <-d.chState
			if !ok {
				break
			}
		}
	}()

	return d, nil
}

// Option sets the options specified.
func (d *Device) Option(opts ...ble.Option) error {
	return nil
}

// Scan ...
func (d *Device) Scan(ctx context.Context, allowDup bool, h ble.AdvHandler) error {
	d.advHandler = h

	d.cm.Scan(nil, &cbgo.CentralManagerScanOpts{
		AllowDuplicates: allowDup,
	})

	<-ctx.Done()
	d.cm.StopScan()

	return ctx.Err()
}

// Dial ...
func (d *Device) Dial(ctx context.Context, a ble.Addr) (ble.Client, error) {
	uuid, err := cbgo.ParseUUID(uuidStrWithDashes(a.String()))
	if err != nil {
		return nil, fmt.Errorf("dial failed: invalid peer address: %s", a)
	}

	prphs := d.cm.RetrievePeripheralsWithIdentifiers([]cbgo.UUID{uuid})
	if len(prphs) == 0 {
		return nil, fmt.Errorf("dial failed: no peer with address: %s", a)
	}

	d.cm.Connect(prphs[0], nil)
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

func (d *Device) DidUpdateState(cmgr cbgo.CentralManager) {
	d.chState <- struct{}{}
}

func (d *Device) DidDiscoverPeripheral(cmgr cbgo.CentralManager, prph cbgo.Peripheral, advFields cbgo.AdvFields, rssi int) {
	if d.advHandler == nil {
		return
	}

	a := &adv{
		localName: advFields.LocalName,
		rssi:      int(rssi),
		mfgData:   advFields.ManufacturerData,
	}
	if advFields.Connectable != nil {
		a.connectable = *advFields.Connectable
	}
	if advFields.TxPowerLevel != nil {
		a.powerLevel = *advFields.TxPowerLevel
	}
	for _, u := range advFields.ServiceUUIDs {
		a.svcUUIDs = append(a.svcUUIDs, ble.UUID(u))
	}
	for _, sd := range advFields.ServiceData {
		a.svcData = append(a.svcData, ble.ServiceData{
			UUID: ble.UUID(sd.UUID),
			Data: sd.Data,
		})
	}
	a.peerUUID = ble.UUID(prph.Identifier())

	d.advHandler(a)
}

func (d *Device) DidConnectPeripheral(cmgr cbgo.CentralManager, prph cbgo.Peripheral) {
	d.connLock.Lock()
	defer d.connLock.Unlock()

	fail := func(err error) {
		d.chConn <- &connectResult{
			err: err,
		}
	}

	a := ble.Addr(prph.Identifier())

	if d.conns[a.String()] != nil {
		fail(fmt.Errorf("failed to add connection: already exists: addr=%s", a.String()))
		return
	}

	c := newConn(d, prph)
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
