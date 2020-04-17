package darwin

// CBHandlers: Go handlers for asynchronous CoreBluetooth callbacks.

/*
// See cutil.go for C compiler flags.
#import "bt.h"
*/
import "C"

import (
	"fmt"
	"log"
	"unsafe"

	"github.com/JuulLabs-OSS/ble"
)

//export BTStateChanged
func BTStateChanged(mgrID C.uintptr_t, enabled C.int, msg *C.char) {
	id := uintptr(mgrID)
	state := BTState{
		Enabled: enabled != 0,
		Msg:     C.GoString(msg),
	}

	found := cmgrStateChanged(id, state)
	if !found {
		log.Printf("state change event for unknown manager: id=%d state=%+v", id, state)
	}
}

//export BTPeripheralDiscovered
func BTPeripheralDiscovered(cmgrID C.uintptr_t, dp *C.struct_discovered_prph) {
	d := findCmgr(cmgrID)
	if d == nil || d.advHandler == nil {
		return
	}

	a := &adv{
		rssi:        int(dp.rssi),
		localName:   C.GoString(dp.local_name),
		mfgData:     byteArrToByteSlice(dp.mfg_data),
		powerLevel:  int(dp.power_level),
		connectable: dp.connectable != 0,
	}

	var err error

	a.peerUUID, err = ble.Parse(C.GoString(dp.peer_uuid))
	if err != nil {
		log.Printf("BTPeripheralDiscovered failed: %v", err)
		return
	}

	for i := 0; i < int(dp.num_svc_uuids); i++ {
		str := cArrGetStr(dp.svc_uuids, i)
		uuid, err := ble.Parse(str)
		if err != nil {
			log.Printf("BTPeripheralDiscovered failed: %v", err)
			return
		}

		a.svcUUIDs = append(a.svcUUIDs, uuid)
	}

	for i := 0; i < int(dp.num_svc_data); i++ {
		str := cArrGetStr(dp.svc_data_uuids, i)
		uuid, err := ble.Parse(str)
		if err != nil {
			log.Printf("BTPeripheralDiscovered failed: %v", err)
			return
		}

		value := cArrGetAddr(unsafe.Pointer(dp.svc_data_values), unsafe.Sizeof(*dp.svc_data_values), i)
		valueB := byteArrToByteSlice(*(*C.struct_byte_arr)(value))

		a.svcData = append(a.svcData, ble.ServiceData{
			UUID: uuid,
			Data: valueB,
		})
	}

	go d.advHandler(a)
}

//export BTPeripheralConnected
func BTPeripheralConnected(cmgrID C.uintptr_t, uuidStr *C.char, status C.int) {
	d := findCmgr(cmgrID)
	if d == nil {
		log.Printf("BTPeripheralConnected failed: device not found: ID=%v", cmgrID)
		return
	}

	uuid, err := ble.Parse(C.GoString(uuidStr))
	if err != nil {
		log.Printf("BTPeripheralConnected failed: %v", err)
		return
	}

	if status != 0 {
		d.connectFail(fmt.Errorf("failed to connect: status=%d uuid=%s"))
		return
	}

	d.connectSuccess(ble.Addr(uuid))
}

//export BTPeripheralDisconnected
func BTPeripheralDisconnected(cmgrID C.uintptr_t, uuidStr *C.char, reason C.int) {
	_, c, err := findCmgrAndConn(cmgrID, uuidStr)
	if err != nil {
		log.Printf("BTPeripheralDisconnected failed: %v", err)
		return
	}

	c.evl.disconnected <- &eventDisconnected{
		reason: int(reason),
	}
}

//export BTServicesDiscovered
func BTServicesDiscovered(cmgrID C.uintptr_t, uuidStr *C.char, status C.int,
	svcs *C.struct_discovered_svc, numSvcs C.int) {

	_, c, err := findCmgrAndConn(cmgrID, uuidStr)
	if err != nil {
		log.Printf("BTServicesDiscovered failed: %v", err)
		return
	}

	if status != 0 {
		c.evl.svcsDiscovered <- &eventSvcsDiscovered{
			err: fmt.Errorf("failed to discover services: %d", int(status)),
		}
		return
	}

	var dsvcs []discoveredSvc

	for i := 0; i < int(numSvcs); i++ {
		elem := cArrGetAddr(unsafe.Pointer(svcs), unsafe.Sizeof(*svcs), i)
		svc := (*C.struct_discovered_svc)(elem)

		svcUUID, err := ble.Parse(C.GoString(svc.uuid))
		if err != nil {
			log.Printf("BTServicesDiscovered failed: %v", err)
			return
		}
		dsvcs = append(dsvcs, discoveredSvc{
			id:   uintptr(svc.id),
			uuid: svcUUID,
		})
	}

	c.evl.svcsDiscovered <- &eventSvcsDiscovered{
		svcs: dsvcs,
	}
}

//export BTCharacteristicsDiscovered
func BTCharacteristicsDiscovered(cmgrID C.uintptr_t, uuidStr *C.char, status C.int,
	chrs *C.struct_discovered_chr, numChrs C.int) {

	_, c, err := findCmgrAndConn(cmgrID, uuidStr)
	if err != nil {
		log.Printf("BTCharacteristicsDiscovered failed: %v", err)
		return
	}

	if status != 0 {
		c.evl.chrsDiscovered <- &eventChrsDiscovered{
			err: fmt.Errorf("failed to discover characteristics: %d", int(status)),
		}
		return
	}

	var dchrs []discoveredChr

	for i := 0; i < int(numChrs); i++ {
		elem := cArrGetAddr(unsafe.Pointer(chrs), unsafe.Sizeof(*chrs), i)
		chr := (*C.struct_discovered_chr)(elem)

		chrUUID, err := ble.Parse(C.GoString(chr.uuid))
		if err != nil {
			log.Printf("BTCharacteristicsDiscovered failed: %v", err)
			return
		}
		dchrs = append(dchrs, discoveredChr{
			id:         uintptr(chr.id),
			uuid:       chrUUID,
			properties: ble.Property(chr.properties),
		})
	}

	c.evl.chrsDiscovered <- &eventChrsDiscovered{
		chrs: dchrs,
	}
}

//export BTDescriptorsDiscovered
func BTDescriptorsDiscovered(cmgrID C.uintptr_t, uuidStr *C.char, status C.int,
	dscs *C.struct_discovered_dsc, numDscs C.int) {

	_, c, err := findCmgrAndConn(cmgrID, uuidStr)
	if err != nil {
		log.Printf("BTDescriptorsDiscovered failed: %v", err)
		return
	}

	if status != 0 {
		c.evl.dscsDiscovered <- &eventDscsDiscovered{
			err: fmt.Errorf("failed to discover descriptors: %d", int(status)),
		}
		return
	}

	var ddscs []discoveredDsc

	for i := 0; i < int(numDscs); i++ {
		elem := cArrGetAddr(unsafe.Pointer(dscs), unsafe.Sizeof(*dscs), i)
		dsc := (*C.struct_discovered_dsc)(elem)

		dscUUID, err := ble.Parse(C.GoString(dsc.uuid))
		if err != nil {
			log.Printf("BTDescriptorsDiscovered failed: %v", err)
			return
		}
		ddscs = append(ddscs, discoveredDsc{
			id:   uintptr(dsc.id),
			uuid: dscUUID,
		})
	}

	c.evl.dscsDiscovered <- &eventDscsDiscovered{
		dscs: ddscs,
	}
}

//export BTCharacteristicRead
func BTCharacteristicRead(cmgrID C.uintptr_t, uuidStr *C.char, status C.int, chrUUID *C.char,
	chrVal *C.struct_byte_arr) {

	_, c, err := findCmgrAndConn(cmgrID, uuidStr)
	if err != nil {
		log.Printf("BTCharacteristicRead failed: %v", err)
		return
	}

	goUUID, err := ble.Parse(C.GoString(chrUUID))
	if err != nil {
		log.Printf("BTCharacteristicRead failed: %v", err)
		return
	}

	fail := func(err error) {
		c.processChrRead(&eventChrRead{
			uuid: goUUID,
			err:  fmt.Errorf("failed to process read characteristic event: %v", err),
		})
	}

	if status != 0 {
		fail(fmt.Errorf("status=%d", int(status)))
		return
	}

	c.processChrRead(&eventChrRead{
		uuid:  goUUID,
		value: byteArrToByteSlice(*chrVal),
	})
}

//export BTCharacteristicWritten
func BTCharacteristicWritten(cmgrID C.uintptr_t, uuidStr *C.char, status C.int, chrUUID *C.char) {
	_, c, err := findCmgrAndConn(cmgrID, uuidStr)
	if err != nil {
		log.Printf("BTCharacteristicWritten failed: %v", err)
		return
	}

	fail := func(err error) {
		c.evl.chrWritten <- &eventChrWritten{
			err: fmt.Errorf("failed to process write characteristic event: %v", err),
		}
	}

	if status != 0 {
		fail(fmt.Errorf("status=%d", status))
		return
	}

	goUUID, err := ble.Parse(C.GoString(chrUUID))
	if err != nil {
		fail(fmt.Errorf("invalid characteristic UUID: UUID=%s", C.GoString(chrUUID)))
		return
	}

	c.evl.chrWritten <- &eventChrWritten{
		uuid: goUUID,
	}
}

//export BTDescriptorRead
func BTDescriptorRead(cmgrID C.uintptr_t, uuidStr *C.char, status C.int, dscUUID *C.char, dscVal *C.struct_byte_arr) {
	_, c, err := findCmgrAndConn(cmgrID, uuidStr)
	if err != nil {
		log.Printf("BTDescriptorRead failed: %v", err)
		return
	}

	goUUID, err := ble.Parse(C.GoString(dscUUID))
	if err != nil {
		log.Printf("BTDescriptorRead failed: %v", err)
		return
	}

	fail := func(err error) {
		c.evl.dscRead <- &eventDscRead{
			uuid: goUUID,
			err:  fmt.Errorf("failed to process read descriptor event: %v", err),
		}
	}

	if status != 0 {
		fail(fmt.Errorf("status=%d", int(status)))
		return
	}

	c.evl.dscRead <- &eventDscRead{
		uuid:  goUUID,
		value: byteArrToByteSlice(*dscVal),
	}
}

//export BTDescriptorWritten
func BTDescriptorWritten(cmgrID C.uintptr_t, uuidStr *C.char, status C.int, dscUUID *C.char) {
	_, c, err := findCmgrAndConn(cmgrID, uuidStr)
	if err != nil {
		log.Printf("BTDescriptorWritten failed: %v", err)
		return
	}

	fail := func(err error) {
		c.evl.dscWritten <- &eventDscWritten{
			err: fmt.Errorf("failed to process write descriptor event: %v", err),
		}
	}

	if status != 0 {
		fail(fmt.Errorf("status=%d", status))
		return
	}

	goUUID, err := ble.Parse(C.GoString(dscUUID))
	if err != nil {
		fail(fmt.Errorf("invalid descriptor UUID: UUID=%s", C.GoString(dscUUID)))
		return
	}

	c.evl.dscWritten <- &eventDscWritten{
		uuid: goUUID,
	}
}

//export BTNotificationStateChanged
func BTNotificationStateChanged(cmgrID C.uintptr_t, uuidStr *C.char, status C.int, chrUUID *C.char, enabled C.bool) {
	_, c, err := findCmgrAndConn(cmgrID, uuidStr)
	if err != nil {
		log.Printf("BTNotificationStateChanged failed: %v", err)
		return
	}

	fail := func(err error) {
		c.evl.notifyChanged <- &eventNotifyChanged{
			err: fmt.Errorf("failed to process notify changed event: %v", err),
		}
	}

	if status != 0 {
		fail(fmt.Errorf("status=%d", status))
		return
	}

	goUUID, err := ble.Parse(C.GoString(chrUUID))
	if err != nil {
		fail(fmt.Errorf("invalid characteristic UUID: UUID=%s", C.GoString(chrUUID)))
		return
	}

	c.evl.notifyChanged <- &eventNotifyChanged{
		uuid:    goUUID,
		enabled: bool(enabled),
	}
}

//export BTRSSIRead
func BTRSSIRead(cmgrID C.uintptr_t, uuidStr *C.char, status C.int, rssi C.int) {
	_, c, err := findCmgrAndConn(cmgrID, uuidStr)
	if err != nil {
		log.Printf("BTRSSIRead failed: %v", err)
		return
	}

	fail := func(err error) {
		c.evl.rssiRead <- &eventRSSIRead{
			err: fmt.Errorf("failed to process RSSI read event: %v", err),
		}
	}

	if status != 0 {
		fail(fmt.Errorf("status=%d", status))
		return
	}

	c.evl.rssiRead <- &eventRSSIRead{
		rssi: int(rssi),
	}
}
