package darwin

/*
// See cutil.go for C compiler flags.
#import "bt.h"
*/
import "C"
import (
	"fmt"
	"runtime"
	"sync"
	"unsafe"

	"github.com/JuulLabs-OSS/ble"
)

var cmgrIDDevMap = map[uintptr]*Device{}
var cmgrIDDevMapMutex sync.RWMutex

func findCmgr(id C.uintptr_t) *Device {
	cmgrIDDevMapMutex.RLock()
	defer cmgrIDDevMapMutex.RUnlock()

	return cmgrIDDevMap[uintptr(id)]
}

func addCmgr(id uintptr, d *Device) {
	cmgrIDDevMapMutex.Lock()
	defer cmgrIDDevMapMutex.Unlock()

	cmgrIDDevMap[id] = d
}

func delCmgr(id uintptr) *Device {
	cmgrIDDevMapMutex.Lock()
	defer cmgrIDDevMapMutex.Unlock()

	d := cmgrIDDevMap[id]
	delete(cmgrIDDevMap, id)

	return d
}

func cmgrStateChanged(id uintptr, state BTState) bool {
	// We have to keep the mutex locked even after we retrieve the device from
	// the map.  This necessary in case another thread closes the channel
	// (Stop()) between retrieval and send.
	cmgrIDDevMapMutex.RLock()
	defer cmgrIDDevMapMutex.RUnlock()

	d := cmgrIDDevMap[id]
	if d == nil {
		return false
	}

	d.cm.stateCh <- state
	return true
}

func findCmgrAndConn(cmgrID C.uintptr_t, uuidStr *C.char) (*Device, *conn, error) {
	d := findCmgr(cmgrID)
	if d == nil {
		return nil, nil, fmt.Errorf("no device for central manager: cmgrID=%d",
			uintptr(cmgrID))
	}

	goStr := C.GoString(uuidStr)
	uuid, err := ble.Parse(goStr)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid device uuid: uuid=%s", goStr)
	}

	c := d.findConn(uuid)
	if c == nil {
		return nil, nil, fmt.Errorf("no connection with uuid: cmgrID=%d uuid=%s",
			uintptr(cmgrID), C.GoString(uuidStr))
	}

	return d, c, nil
}

type CentralMgr struct {
	ptr     unsafe.Pointer
	id      uintptr
	stateCh chan BTState
}

func NewCentralMgr() *CentralMgr {
	cm := &CentralMgr{}
	runtime.SetFinalizer(cm, func(x *CentralMgr) {
		x.Stop()
	})

	return cm
}

func (cm *CentralMgr) Start(d *Device) error {
	if cm.ptr != nil {
		return fmt.Errorf("failed to start central manager: already started")
	}

	cm.ptr = unsafe.Pointer(C.cb_alloc_cmgr())
	cm.id = uintptr(C.cb_cmgr_id(cm.ptr))
	cm.stateCh = make(chan BTState)

	addCmgr(cm.id, d)

	err := StartBTLoop(cm.stateCh)
	if err != nil {
		return fmt.Errorf("failed to start central manager: %v", err)
	}

	return nil
}

func (cm *CentralMgr) Stop() {
	if cm.ptr == nil {
		return
	}

	C.cb_destroy_cmgr(cm.ptr)
	cm.ptr = nil

	d := delCmgr(cm.id)
	close(cm.stateCh)

	if d != nil {
		d.closeConns()
	}
}

func (cm *CentralMgr) Scan(filterDups bool) {
	C.cb_scan(cm.ptr, C.bool(filterDups))
}

func (cm *CentralMgr) StopScan() {
	C.cb_stop_scan(cm.ptr)
}

func (cm *CentralMgr) Connect(a ble.Addr) error {
	cs := C.CString(uuidStrWithDashes(a.String()))
	defer C.free(unsafe.Pointer(cs))

	rc := C.cb_connect(cm.ptr, cs)
	if rc != 0 {
		return fmt.Errorf("connect failed: device not found: uuid=%s", a.String())
	}

	return nil
}

func (cm *CentralMgr) cancelConnection(a ble.Addr) error {
	cs := C.CString(uuidStrWithDashes(a.String()))
	defer C.free(unsafe.Pointer(cs))

	rc := C.cb_cancel_connection(cm.ptr, cs)
	if rc != 0 {
		return fmt.Errorf("failed to cancel connection: device not found: uuid=%s", a.String())
	}

	return nil
}

func (cm *CentralMgr) attMTU(a ble.Addr) (int, error) {
	cs := C.CString(uuidStrWithDashes(a.String()))
	defer C.free(unsafe.Pointer(cs))

	mtu := C.cb_att_mtu(cm.ptr, cs)
	if mtu < 0 {
		return 0, fmt.Errorf("failed to determine ATT MTU: device not found: uuid=%s", a.String())
	}

	return int(mtu), nil
}

func (cm *CentralMgr) discoverServices(a ble.Addr, serviceUUIDs []ble.UUID) error {
	cs := C.CString(uuidStrWithDashes(a.String()))
	defer C.free(unsafe.Pointer(cs))

	var carr unsafe.Pointer

	if len(serviceUUIDs) > 0 {
		elemSz := unsafe.Sizeof((*C.char)(nil))
		carr := C.malloc(C.size_t(len(serviceUUIDs)) * C.size_t(elemSz))
		defer C.free(carr)

		garr := (*[1<<30 - 1]*C.char)(carr)
		for i, u := range serviceUUIDs {
			garr[i] = C.CString(u.String())
		}
		defer func() {
			for i, _ := range serviceUUIDs {
				C.free(unsafe.Pointer(garr[i]))
			}
		}()
	}

	rc := C.cb_discover_svcs(cm.ptr, cs, (**C.char)(carr), C.int(len(serviceUUIDs)))
	if rc != 0 {
		return fmt.Errorf("failed to discover services: device not found: uuid=%s", a.String())
	}

	return nil
}

func (cm *CentralMgr) discoverCharacteristics(a ble.Addr, svcID uintptr, characteristicUUIDs []ble.UUID) error {
	cs := C.CString(uuidStrWithDashes(a.String()))
	defer C.free(unsafe.Pointer(cs))

	var carr unsafe.Pointer

	if len(characteristicUUIDs) > 0 {
		elemSz := unsafe.Sizeof((*C.char)(nil))
		carr := C.malloc(C.size_t(len(characteristicUUIDs)) * C.size_t(elemSz))
		defer C.free(carr)

		garr := (*[1000]*C.char)(carr)
		for i, u := range characteristicUUIDs {
			garr[i] = C.CString(u.String())
		}
		defer func() {
			for i, _ := range characteristicUUIDs {
				C.free(unsafe.Pointer(garr[i]))
			}
		}()
	}

	rc := C.cb_discover_chrs(cm.ptr, cs, C.uintptr_t(svcID), (**C.char)(carr), C.int(len(characteristicUUIDs)))
	if rc != 0 {
		return fmt.Errorf("failed to discover characteristics: device not found: uuid=%s", a.String())
	}

	return nil
}

func (cm *CentralMgr) discoverDescriptors(a ble.Addr, chrID uintptr) error {
	cs := C.CString(uuidStrWithDashes(a.String()))
	defer C.free(unsafe.Pointer(cs))

	rc := C.cb_discover_dscs(cm.ptr, cs, C.uintptr_t(chrID))
	if rc != 0 {
		return fmt.Errorf("failed to discover descriptors: device not found: uuid=%s", a.String())
	}

	return nil
}

func (cm *CentralMgr) readCharacteristic(a ble.Addr, chrID uintptr) error {
	cs := C.CString(uuidStrWithDashes(a.String()))
	defer C.free(unsafe.Pointer(cs))

	rc := C.cb_read_chr(cm.ptr, cs, C.uintptr_t(chrID))
	if rc != 0 {
		return fmt.Errorf("failed to read characteristic: device not found: uuid=%s", a.String())
	}

	return nil
}

func (cm *CentralMgr) writeCharacteristic(a ble.Addr, chrID uintptr, val []byte, noRsp bool) error {
	cs := C.CString(uuidStrWithDashes(a.String()))
	defer C.free(unsafe.Pointer(cs))

	byteArr := byteSliceToByteArr(val)
	defer C.free(unsafe.Pointer(byteArr.data))

	rc := C.cb_write_chr(cm.ptr, cs, C.uintptr_t(chrID), &byteArr, C.bool(noRsp))
	if rc != 0 {
		return fmt.Errorf("failed to write characteristic: device not found: uuid=%s", a.String())
	}

	return nil
}

func (cm *CentralMgr) readDescriptor(a ble.Addr, dscID uintptr) error {
	cs := C.CString(uuidStrWithDashes(a.String()))
	defer C.free(unsafe.Pointer(cs))

	rc := C.cb_read_dsc(cm.ptr, cs, C.uintptr_t(dscID))
	if rc != 0 {
		return fmt.Errorf("failed to read descriptor: device not found: uuid=%s", a.String())
	}

	return nil
}

func (cm *CentralMgr) writeDescriptor(a ble.Addr, dscID uintptr, val []byte) error {
	cs := C.CString(uuidStrWithDashes(a.String()))
	defer C.free(unsafe.Pointer(cs))

	byteArr := byteSliceToByteArr(val)
	defer C.free(unsafe.Pointer(byteArr.data))

	rc := C.cb_write_dsc(cm.ptr, cs, C.uintptr_t(dscID), &byteArr)
	if rc != 0 {
		return fmt.Errorf("failed to write descriptor: device not found: uuid=%s", a.String())
	}

	return nil
}

func (cm *CentralMgr) subscribe(a ble.Addr, chrID uintptr) error {
	cs := C.CString(uuidStrWithDashes(a.String()))
	defer C.free(unsafe.Pointer(cs))

	rc := C.cb_subscribe(cm.ptr, cs, C.uintptr_t(chrID))
	if rc != 0 {
		return fmt.Errorf("failed to subscribe: device not found: uuid=%s", a.String())
	}

	return nil
}

func (cm *CentralMgr) unsubscribe(a ble.Addr, chrID uintptr) error {
	cs := C.CString(uuidStrWithDashes(a.String()))
	defer C.free(unsafe.Pointer(cs))

	rc := C.cb_unsubscribe(cm.ptr, cs, C.uintptr_t(chrID))
	if rc != 0 {
		return fmt.Errorf("failed to unsubscribe: device not found: uuid=%s", a.String())
	}

	return nil
}

func (cm *CentralMgr) readRSSI(a ble.Addr) error {
	cs := C.CString(uuidStrWithDashes(a.String()))
	defer C.free(unsafe.Pointer(cs))

	rc := C.cb_read_rssi(cm.ptr, cs)
	if rc != 0 {
		return fmt.Errorf("failed to read RSSI: device not found: uuid=%s", a.String())
	}

	return nil
}
