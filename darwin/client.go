package darwin

import (
	"fmt"

	"github.com/JuulLabs-OSS/ble"
)

// A Client is a GATT client.
type Client struct {
	profile *ble.Profile
	name    string
	cm      *CentralMgr

	id   ble.UUID
	conn *conn
}

// NewClient ...
func NewClient(cm *CentralMgr, c ble.Conn) (*Client, error) {
	as := c.RemoteAddr().String()
	id, err := ble.Parse(as)
	if err != nil {
		return nil, fmt.Errorf("connection has invalid address: addr=%s", as)
	}

	return &Client{
		conn: c.(*conn),
		cm:   cm,
		id:   id,
	}, nil
}

// Addr returns UUID of the remote peripheral.
func (cln *Client) Addr() ble.Addr {
	return cln.conn.RemoteAddr()
}

// Name returns the name of the remote peripheral.
// This can be the advertised name, if exists, or the GAP device name, which takes priority.
func (cln *Client) Name() string {
	return cln.name
}

// Profile returns the discovered profile.
func (cln *Client) Profile() *ble.Profile {
	return cln.profile
}

// DiscoverProfile discovers the whole hierarchy of a server.
func (cln *Client) DiscoverProfile(force bool) (*ble.Profile, error) {
	if cln.profile != nil && !force {
		return cln.profile, nil
	}
	ss, err := cln.DiscoverServices(nil)
	if err != nil {
		return nil, fmt.Errorf("can't discover services: %s", err)
	}
	for _, s := range ss {
		cs, err := cln.DiscoverCharacteristics(nil, s)
		if err != nil {
			return nil, fmt.Errorf("can't discover characteristics: %s", err)
		}
		for _, c := range cs {
			_, err := cln.DiscoverDescriptors(nil, c)
			if err != nil {
				return nil, fmt.Errorf("can't discover descriptors: %s", err)
			}
		}
	}
	cln.profile = &ble.Profile{Services: ss}
	return cln.profile, nil
}

// DiscoverServices finds all the primary services on a server. [Vol 3, Part G, 4.4.1]
// If filter is specified, only filtered services are returned.
func (cln *Client) DiscoverServices(ss []ble.UUID) ([]*ble.Service, error) {
	err := cln.cm.discoverServices(cln.id, ss)
	if err != nil {
		return nil, err
	}

	ev := <-cln.conn.evl.svcsDiscovered
	if ev.err != nil {
		return nil, ev.err
	}

	svcs := []*ble.Service{}
	for _, dsvc := range ev.svcs {
		svcs = append(svcs, &ble.Service{
			UUID: dsvc.uuid,
			ID:   dsvc.id,
		})
	}
	if cln.profile == nil {
		cln.profile = &ble.Profile{Services: svcs}
	}
	return svcs, nil
}

// DiscoverIncludedServices finds the included services of a service. [Vol 3, Part G, 4.5.1]
// If filter is specified, only filtered services are returned.
func (cln *Client) DiscoverIncludedServices(ss []ble.UUID, s *ble.Service) ([]*ble.Service, error) {
	return nil, ble.ErrNotImplemented
}

// DiscoverCharacteristics finds all the characteristics within a service. [Vol 3, Part G, 4.6.1]
// If filter is specified, only filtered characteristics are returned.
func (cln *Client) DiscoverCharacteristics(cs []ble.UUID, s *ble.Service) ([]*ble.Characteristic, error) {
	err := cln.cm.discoverCharacteristics(cln.id, s.ID, cs)
	if err != nil {
		return nil, err
	}

	ev := <-cln.conn.evl.chrsDiscovered
	if ev.err != nil {
		return nil, ev.err
	}

	for _, dchr := range ev.chrs {
		s.Characteristics = append(s.Characteristics, &ble.Characteristic{
			UUID:     dchr.uuid,
			Property: dchr.properties,
			ID:       dchr.id,
		})
	}
	return s.Characteristics, nil
}

// DiscoverDescriptors finds all the descriptors within a characteristic. [Vol 3, Part G, 4.7.1]
// If filter is specified, only filtered descriptors are returned.
func (cln *Client) DiscoverDescriptors(ds []ble.UUID, c *ble.Characteristic) ([]*ble.Descriptor, error) {
	err := cln.cm.discoverDescriptors(cln.id, c.ID)
	if err != nil {
		return nil, err
	}

	ev := <-cln.conn.evl.dscsDiscovered
	if ev.err != nil {
		return nil, ev.err
	}

	for _, ddsc := range ev.dscs {
		c.Descriptors = append(c.Descriptors, &ble.Descriptor{
			UUID: ddsc.uuid,
			ID:   ddsc.id,
		})
	}
	return c.Descriptors, nil
}

// ReadCharacteristic reads a characteristic value from a server. [Vol 3, Part G, 4.8.1]
func (cln *Client) ReadCharacteristic(c *ble.Characteristic) ([]byte, error) {
	ch, err := cln.conn.addChrReader(c)
	if err != nil {
		return nil, fmt.Errorf("failed to read characteristic: %v", err)
	}
	defer cln.conn.delChrReader(c)

	err = cln.cm.readCharacteristic(cln.id, c.ID)
	if err != nil {
		return nil, err
	}

	ev := <-ch
	if ev.err != nil {
		return nil, ev.err
	}

	c.Value = ev.value

	return ev.value, nil
}

// ReadLongCharacteristic reads a characteristic value which is longer than the MTU. [Vol 3, Part G, 4.8.3]
func (cln *Client) ReadLongCharacteristic(c *ble.Characteristic) ([]byte, error) {
	return cln.ReadCharacteristic(c)
}

// WriteCharacteristic writes a characteristic value to a server. [Vol 3, Part G, 4.9.3]
func (cln *Client) WriteCharacteristic(c *ble.Characteristic, b []byte, noRsp bool) error {
	err := cln.cm.writeCharacteristic(cln.id, c.ID, b, noRsp)
	if err != nil {
		return err
	}
	if !noRsp {
		ev := <-cln.conn.evl.chrWritten
		if ev.err != nil {
			return ev.err
		}
	}

	return nil
}

// ReadDescriptor reads a characteristic descriptor from a server. [Vol 3, Part G, 4.12.1]
func (cln *Client) ReadDescriptor(d *ble.Descriptor) ([]byte, error) {
	err := cln.cm.readDescriptor(cln.id, d.ID)
	if err != nil {
		return nil, err
	}

	ev := <-cln.conn.evl.dscRead
	if ev.err != nil {
		return nil, ev.err
	}

	d.Value = ev.value

	return ev.value, nil
}

// WriteDescriptor writes a characteristic descriptor to a server. [Vol 3, Part G, 4.12.3]
func (cln *Client) WriteDescriptor(d *ble.Descriptor, b []byte) error {
	err := cln.cm.writeDescriptor(cln.id, d.ID, b)
	if err != nil {
		return err
	}

	ev := <-cln.conn.evl.dscWritten
	if ev.err != nil {
		return ev.err
	}

	return nil
}

// ReadRSSI retrieves the current RSSI value of remote peripheral. [Vol 2, Part E, 7.5.4]
func (cln *Client) ReadRSSI() int {
	err := cln.cm.readRSSI(cln.id)
	if err != nil {
		return 0
	}

	ev := <-cln.conn.evl.rssiRead
	if ev.err != nil {
		return 0
	}

	return ev.rssi
}

// ExchangeMTU set the ATT_MTU to the maximum possible value that can be
// supported by both devices [Vol 3, Part G, 4.3.1]
func (cln *Client) ExchangeMTU(mtu int) (int, error) {
	// TODO: find the xpc command to tell OS X the rxMTU we can handle.
	return cln.conn.TxMTU(), nil
}

// Subscribe subscribes to indication (if ind is set true), or notification of a
// characteristic value. [Vol 3, Part G, 4.10 & 4.11]
func (cln *Client) Subscribe(c *ble.Characteristic, ind bool, fn ble.NotificationHandler) error {
	cln.conn.addSub(c, fn)

	err := cln.cm.subscribe(cln.id, c.ID)
	if err != nil {
		cln.conn.delSub(c)
		return err
	}

	ev := <-cln.conn.evl.notifyChanged
	if ev.err != nil {
		cln.conn.delSub(c)
		return ev.err
	}

	if !ev.enabled {
		cln.conn.delSub(c)
		return fmt.Errorf("subscribed to notifications but CB says disabled: uuid=%s", c.UUID.String())
	}

	return nil
}

// Unsubscribe unsubscribes to indication (if ind is set true), or notification
// of a specified characteristic value. [Vol 3, Part G, 4.10 & 4.11]
func (cln *Client) Unsubscribe(c *ble.Characteristic, ind bool) error {
	err := cln.cm.unsubscribe(cln.id, c.ID)
	if err != nil {
		return err
	}

	ev := <-cln.conn.evl.notifyChanged
	if ev.err != nil {
		return ev.err
	}

	if ev.enabled {
		return fmt.Errorf("unsubscribed from notifications but CB says enabled: uuid=%s", c.UUID.String())
	}

	cln.conn.delSub(c)

	return nil
}

// ClearSubscriptions clears all subscriptions to notifications and indications.
func (cln *Client) ClearSubscriptions() error {
	for _, s := range cln.conn.subs {
		if err := cln.Unsubscribe(s.char, false); err != nil {
			return err
		}
	}
	return nil
}

// CancelConnection disconnects the connection.
func (cln *Client) CancelConnection() error {
	err := cln.cm.cancelConnection(cln.conn.addr)
	if err != nil {
		return err
	}

	return nil
}

// Disconnected returns a receiving channel, which is closed when the client disconnects.
func (cln *Client) Disconnected() <-chan struct{} {
	return cln.conn.Disconnected()
}

// Conn returns the client's current connection.
func (cln *Client) Conn() ble.Conn {
	return cln.conn
}

type sub struct {
	fn   ble.NotificationHandler
	char *ble.Characteristic
}
