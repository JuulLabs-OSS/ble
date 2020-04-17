#import <Foundation/Foundation.h>
#import <CoreBluetooth/CoreBluetooth.h>
#import "bt.h"

@implementation CMgr

- (id)
init
{
    self = [super init];
    if (self == nil) {
        return nil;
    }

    // Ensure queue is initialized.
    bt_init();

    _manager = [[CBCentralManager alloc] initWithDelegate:self
                                                    queue:bt_queue];
    return self;
}

- (void)
dealloc
{
    [_manager release];
    [super dealloc];
}

/**
 * Retrieves the CMgr's ID.  Used for passing a CMgr between objc and Go code.
 */
- (uintptr_t)
ID
{
    return (uintptr_t)_manager;
}

/**
 * Called whenever the central manager's state is updated.
 */
- (void)
centralManagerDidUpdateState:(CBCentralManager *)cm
{
    int enabled;
    char *msg;

    assert(cm == _manager);

    switch ([cm state]) {
    case CBManagerStateUnsupported:
        enabled = 0;
        msg = "The platform/hardware doesn't support Bluetooth Low Energy";
        break;
    case CBManagerStateUnauthorized:
        enabled = 0;
        msg = "The app is not authorized to use Bluetooth Low Energy";
        break;
    case CBManagerStatePoweredOff:
        enabled = 0;
        msg = "Bluetooth is currently powered off";
        break;
    case CBManagerStatePoweredOn:
        enabled = 1;
        msg = "Bluetooth is currently powered on";
        break;
    case CBManagerStateUnknown:
        enabled = 0;
        msg = "Bluetooth state unknown";
        break;
    default:
        enabled = 0;
        msg = "Bluetooth state invalid";
        break;
    }

    BTStateChanged([self ID], enabled, msg);
}

/**
 * Called when the central manager discovers a peripheral while scanning for
 * devices.
 */
- (void) centralManager:(CBCentralManager *)cm
didDiscoverPeripheral:(CBPeripheral *)prph
    advertisementData:(NSDictionary *)advData
    RSSI:(NSNumber *)RSSI
{
    struct discovered_prph dp = {0};

    dp.rssi = [RSSI intValue];
    dp.local_name = dict_string(advData, CBAdvertisementDataLocalNameKey);
    dp.mfg_data = dict_bytes(advData, CBAdvertisementDataManufacturerDataKey);
    dp.power_level = dict_int(advData, CBAdvertisementDataTxPowerLevelKey);
    dp.connectable = dict_int(advData, CBAdvertisementDataIsConnectable);

    const NSArray *arr = [advData objectForKey:CBAdvertisementDataServiceUUIDsKey];
    const char *svc_uuids[[arr count]];
    for (int i = 0; i < [arr count]; i++) {
        const CBUUID *uuid = [arr objectAtIndex:i];
        svc_uuids[i] = [[uuid UUIDString] UTF8String];
    }
    dp.svc_uuids = svc_uuids;
    dp.num_svc_uuids = [arr count];

    const NSDictionary *dict = [advData objectForKey:CBAdvertisementDataServiceDataKey];
    const NSArray *keys = [dict allKeys];

    const char *svc_data_uuids[[keys count]];
    struct byte_arr svc_data_values[[keys count]];

    for (int i = 0; i < [keys count]; i++) {
        const CBUUID *uuid = [keys objectAtIndex:i];
        svc_data_uuids[i] = [[uuid UUIDString] UTF8String];

        const NSData *data = [dict objectForKey:uuid];
        svc_data_values[i].data = [data bytes];
        svc_data_values[i].length = [data length];
    }
    dp.svc_data_uuids = svc_data_uuids;
    dp.svc_data_values = svc_data_values;
    dp.num_svc_data = [keys count];

    const NSUUID *uuid = [prph identifier];
    dp.peer_uuid = [[uuid UUIDString] UTF8String];

    BTPeripheralDiscovered([self ID], &dp);
}

/**
 * Called when the central manager successfully connects to a peripheral.
 */
- (void) centralManager:(CBCentralManager *)cm
didConnectPeripheral:(CBPeripheral *)prph
{
    prph.delegate = self;

    const NSUUID *uuid = [prph identifier];
    const char *str = [[uuid UUIDString] UTF8String];
    BTPeripheralConnected([self ID], (char *)str, 0);
}

/**
 * Called when the central manager fails to connect to a peripheral.
 */
- (void)    centralManager:(CBCentralManager *)cm
didFailToConnectPeripheral:(CBPeripheral *)prph
                     error:(NSError *)err
{
    const NSUUID *uuid = [prph identifier];
    const char *str = [[uuid UUIDString] UTF8String];
    BTPeripheralConnected([self ID], (char *)str, [err code]);
}

/**
 * Called when a connection to a peripheral is terminated.
 */
- (void) centralManager:(CBCentralManager *)cm
didDisconnectPeripheral:(CBPeripheral *)prph
                  error:(NSError *)err
{
    const NSUUID *uuid = [prph identifier];
    const char *str = [[uuid UUIDString] UTF8String];

    int reason = -1;
    if (err != NULL) {
        reason = [err code];
    }

    BTPeripheralDisconnected([self ID], (char *)str, reason);
}

/**
 * Called when the central manager successfully discovers services on a
 * peripheral.
 */
- (void) peripheral:(CBPeripheral *)prph
didDiscoverServices:(NSError *)err
{
    const NSUUID *uuid = [prph identifier];
    const char *str = [[uuid UUIDString] UTF8String];

    const NSArray<CBService *> *nsarr = [prph services];

    int status = 0;
    if (err != NULL) {
        status = [err code];
    }

    struct discovered_svc svcs[[nsarr count]];
    for (int i = 0; i < [nsarr count]; i++) {
        CBService *cbsvc = [nsarr objectAtIndex:i];

        svcs[i] = (struct discovered_svc) {
            .id = (uintptr_t)cbsvc,
            .uuid = [[[cbsvc UUID] UUIDString] UTF8String],
        };
    }

    BTServicesDiscovered([self ID], (char *)str, status, svcs, [nsarr count]); 
}

/**
 * Called when the central manager successfully discovers the characteristics
 * in a peripheral's service.
 */
- (void)                  peripheral:(CBPeripheral *)prph
didDiscoverCharacteristicsForService:(CBService *)svc
                               error:(NSError *)err
{
    const NSUUID *uuid = [prph identifier];
    const char *str = [[uuid UUIDString] UTF8String];

    const NSArray<CBCharacteristic *> *nsarr = [svc characteristics];

    int status = 0;
    if (err != NULL) {
        status = [err code];
    }

    struct discovered_chr chrs[[nsarr count]];
    for (int i = 0; i < [nsarr count]; i++) {
        CBCharacteristic *cbchr = [nsarr objectAtIndex:i];

        chrs[i] = (struct discovered_chr) {
            .id = (uintptr_t)cbchr,
            .uuid = [[[cbchr UUID] UUIDString] UTF8String],
            .properties = (uint8_t)[cbchr properties],
        };
    }

    BTCharacteristicsDiscovered([self ID], (char *)str, status, chrs, [nsarr count]); 
}

/**
 * Called when the central manager successfully discovers descriptors in a
 * peripheral's cahracteristic.
 */
- (void)                     peripheral:(CBPeripheral *)prph
didDiscoverDescriptorsForCharacteristic:(CBCharacteristic *)chr
                                  error:(NSError *)err
{
    const NSUUID *uuid = [prph identifier];
    const char *str = [[uuid UUIDString] UTF8String];

    const NSArray<CBDescriptor *> *nsarr = [chr descriptors];

    int status = 0;
    if (err != NULL) {
        status = [err code];
    }

    struct discovered_dsc dscs[[nsarr count]];
    for (int i = 0; i < [nsarr count]; i++) {
        CBDescriptor *cbdsc = [nsarr objectAtIndex:i];

        dscs[i] = (struct discovered_dsc) {
            .id = (uintptr_t)cbdsc,
            .uuid = [[[cbdsc UUID] UUIDString] UTF8String],
        };
    }

    BTDescriptorsDiscovered([self ID], (char *)str, status, dscs, [nsarr count]); 
}

/**
 * Called when the central manager successfully reads a characteristic or when
 * a peripheral sends a notification or indication.
 */
- (void)             peripheral:(CBPeripheral *)prph
didUpdateValueForCharacteristic:(CBCharacteristic *)chr
                          error:(NSError *)err
{
    const NSUUID *uuid = [prph identifier];
    const char *str = [[uuid UUIDString] UTF8String];

    int status = 0;
    if (err != NULL) {
        status = [err code];
    }

    const char *chr_uuid = [[[chr UUID] UUIDString] UTF8String];
    struct byte_arr chr_val = nsdata_to_byte_arr([chr value]);
    BTCharacteristicRead([self ID], (char *)str, status, (char *)chr_uuid, &chr_val);
}

/**
 * Called when a peripheral responds a write request from the central manager's
 * write characteristic request.
 */
- (void)            peripheral:(CBPeripheral *)prph 
didWriteValueForCharacteristic:(CBCharacteristic *)chr 
                         error:(NSError *)err
{
    const NSUUID *uuid = [prph identifier];
    const char *str = [[uuid UUIDString] UTF8String];

    int status = 0;
    if (err != NULL) {
        status = [err code];
    }

    const char *chr_uuid = [[[chr UUID] UUIDString] UTF8String];
    BTCharacteristicWritten([self ID], (char *)str, status, (char *)chr_uuid);
}

/**
 * Called when the central manager successfully reads a descriptor.
 */
- (void)         peripheral:(CBPeripheral *)prph
didUpdateValueForDescriptor:(CBDescriptor *)dsc
                      error:(NSError *)err
{
    const NSUUID *uuid = [prph identifier];
    const char *str = [[uuid UUIDString] UTF8String];

    int status = 0;
    if (err != NULL) {
        status = [err code];
    }

    const char *dsc_uuid = [[[dsc UUID] UUIDString] UTF8String];
    struct byte_arr dsc_val = nsdata_to_byte_arr([dsc value]);
    BTDescriptorRead([self ID], (char *)str, status, (char *)dsc_uuid, &dsc_val);
}

/**
 * Called when a peripheral responds to a write request from the central
 * manager's write descriptor request.
 */
- (void)        peripheral:(CBPeripheral *)prph 
didWriteValueForDescriptor:(CBDescriptor *)dsc 
                     error:(NSError *)err
{
    const NSUUID *uuid = [prph identifier];
    const char *str = [[uuid UUIDString] UTF8String];

    int status = 0;
    if (err != NULL) {
        status = [err code];
    }

    const char *dsc_uuid = [[[dsc UUID] UUIDString] UTF8String];
    BTDescriptorWritten([self ID], (char *)str, status, (char *)dsc_uuid);
}

/**
 * Called when the central manager enables or disables notifications or
 * indications for a peripheral's characteristic.
 */
- (void)                         peripheral:(CBPeripheral *)prph
didUpdateNotificationStateForCharacteristic:(CBCharacteristic *)chr
                                      error:(NSError *)err
{
    const NSUUID *uuid = [prph identifier];
    const char *str = [[uuid UUIDString] UTF8String];

    int status = 0;
    if (err != NULL) {
        status = [err code];
    }

    const char *chr_uuid = [[[chr UUID] UUIDString] UTF8String];
    BTNotificationStateChanged([self ID], (char *)str, status, (char *)chr_uuid, [chr isNotifying]);
}

/**
 * Called when the central manager reads the RSSI associated with a peripheral.
 */
- (void) peripheral:(CBPeripheral *)prph 
        didReadRSSI:(NSNumber *)rssi 
              error:(NSError *)err
{
    const NSUUID *uuid = [prph identifier];
    const char *str = [[uuid UUIDString] UTF8String];

    int status = 0;
    if (err != NULL) {
        status = [err code];
    }

    BTRSSIRead([self ID], (char *)str, status, [rssi intValue]);
}

/**
 * Initiates a scan.
 */
- (void) scan:(BOOL)allow_dup
{
    NSDictionary<NSString *,id> *opts;

    if (allow_dup) {
        opts = @{CBCentralManagerScanOptionAllowDuplicatesKey: @YES};
    } else {
        opts = NULL;
    }

    [_manager scanForPeripheralsWithServices:nil options:opts];
}

/**
 * Stops a scan in progress.
 */
- (void) stopScan
{
    [_manager stopScan];
}

/**
 * Looks up the peripheral with the specified UUID.  Returns NULL if not found.
 */
- (CBPeripheral *) peripheralWithUUID:(NSUUID *)uuid
{
    NSArray *uuids = [NSArray arrayWithObject:uuid];

    NSArray *periphs = [_manager retrievePeripheralsWithIdentifiers:uuids];
    if ([periphs count] == 0) {
        return NULL;
    }

    return [periphs objectAtIndex:0];
}

/**
 * Attempts to connect to the peripheral with the specified UUID.
 *
 * @return                      0 on success;
 *                              nonzero if peripheral not found.
 */
- (int) connect:(NSUUID *)peerUUID
{
    CBPeripheral *periph = [self peripheralWithUUID:peerUUID];
    if (periph == NULL) {
        return -1;
    }

    [_manager connectPeripheral:periph options:@{@"kCBConnectOptionNotifyOnDisconnection": @1}];

    return 0;
}

/**
 * Cancels a connect operation in progress.
 *
 * @return                      0 on success;
 *                              nonzero if peripheral not found.
 */
- (int)cancelConnection:(NSUUID *)peerUUID
{
    CBPeripheral *periph = [self peripheralWithUUID:peerUUID];
    if (periph == NULL) {
        return -1;
    }

    [_manager cancelPeripheralConnection:periph];
    return 0;
}

/**
 * Retrieves the ATT MTU a connected peer can receive.
 *
 * @return                      MTU on success;
 *                              negative number if peripheral not found.
 */
- (int) attMTUForPeriphWithUUID:(NSUUID *)peerUUID
{
    CBPeripheral *periph = [self peripheralWithUUID:peerUUID];
    if (periph == NULL) {
        return -1;
    }

    // +3 to account for size of ATT_WRITE_CMD base.
    return [periph maximumWriteValueLengthForType:CBCharacteristicWriteWithoutResponse] + 3;
}

/**
 * Discovers the services that the specified peripheral supports.  The array of
 * CBUUIDs specifies the services to discover, or NULL to discover all
 * services.
 *
 * @return                      0 on success;
 *                              nonzero if peripheral not found.
 */
- (int) discoverServices:(NSUUID *)peerUUID
    services:(NSArray<CBUUID *> *)svcUUIDs
{
    CBPeripheral *periph = [self peripheralWithUUID:peerUUID];
    if (periph == NULL) {
        return -1;
    }

    [periph discoverServices:svcUUIDs];
    return 0;
}

/**
 * Discovers characterstics belonging to the specified peripheral service.  The
 * array of CBUUIDs specifies the characteristics to discover, or NULL to
 * discover all characteristics.
 *
 * @return                      0 on success;
 *                              nonzero if peripheral not found.
 */
- (int) discoverCharacteristics:(NSUUID *)peerUUID
                        service:(CBService *)svc
                characteristics:(NSArray<CBUUID *> *)chrUUIDs
{
    CBPeripheral *periph = [self peripheralWithUUID:peerUUID];
    if (periph == NULL) {
        return -1;
    }

    [periph discoverCharacteristics:chrUUIDs forService:svc];
    return 0;
}

/**
 * Discovers descriptors belonging to the specified peripheral characteristic.
 *
 * @return                      0 on success;
 *                              nonzero if peripheral not found.
 */
- (int) discoverDescriptors:(NSUUID *)peerUUID
             characteristic:(CBCharacteristic *)chr
{
    CBPeripheral *periph = [self peripheralWithUUID:peerUUID];
    if (periph == NULL) {
        return -1;
    }

    [periph discoverDescriptorsForCharacteristic:chr];
    return 0;
}

/**
 * Reads a peripheral's characteristic.
 *
 * @return                      0 on success;
 *                              nonzero if peripheral not found.
 */
- (int) readCharacteristic:(NSUUID *)peerUUID 
            characteristic:(CBCharacteristic *)chr
{
    CBPeripheral *periph = [self peripheralWithUUID:peerUUID];
    if (periph == NULL) {
        return -1;
    }

    [periph readValueForCharacteristic:chr];
    return 0;
}

/**
 * Writes a peripheral's characteristic.
 *
 * @return                      0 on success;
 *                              nonzero if peripheral not found.
 */
- (int) writeCharacteristic:(NSUUID *)peerUUID 
             characteristic:(CBCharacteristic *)chr
                      value:(struct byte_arr *)val
                 noResponse:(bool)noRsp
{
    CBPeripheral *periph = [self peripheralWithUUID:peerUUID];
    if (periph == NULL) {
        return -1;
    }

    CBCharacteristicWriteType type;
    if (noRsp) {
        type = CBCharacteristicWriteWithoutResponse;
    } else {
        type = CBCharacteristicWriteWithResponse;
    }

    NSData *nsdata = byte_arr_to_nsdata(val);
    [periph writeValue:nsdata forCharacteristic:chr type:type];
    [nsdata release];

    return 0;
}

/**
 * Reads a peripheral's descriptor.
 *
 * @return                      0 on success;
 *                              nonzero if peripheral not found.
 */
- (int) readDescriptor:(NSUUID *)peerUUID 
            descriptor:(CBDescriptor *)dsc
{
    CBPeripheral *periph = [self peripheralWithUUID:peerUUID];
    if (periph == NULL) {
        return -1;
    }

    [periph readValueForDescriptor:dsc];
    return 0;
}

/**
 * Writes a peripheral's descriptor.
 *
 * @return                      0 on success;
 *                              nonzero if peripheral not found.
 */
- (int) writeDescriptor:(NSUUID *)peerUUID 
             descriptor:(CBDescriptor *)dsc
                  value:(struct byte_arr *)val
{
    CBPeripheral *periph = [self peripheralWithUUID:peerUUID];
    if (periph == NULL) {
        return -1;
    }

    NSData *nsdata = byte_arr_to_nsdata(val);
    [periph writeValue:nsdata forDescriptor:dsc];
    [nsdata release];

    return 0;
}

/**
 * Subscribes to notifications or indications for a peripheral's
 * characteristic.
 *
 * @return                      0 on success;
 *                              nonzero if peripheral not found.
 */
- (int) subscribe:(NSUUID *)peerUUID
   characteristic:(CBCharacteristic *)chr
{
    CBPeripheral *periph = [self peripheralWithUUID:peerUUID];
    if (periph == NULL) {
        return -1;
    }

    [periph setNotifyValue:YES forCharacteristic:chr];
    return 0;
}

/**
 * Unsubscribes to notifications or indications for a peripheral's
 * characteristic.
 *
 * @return                      0 on success;
 *                              nonzero if peripheral not found.
 */
- (int) unsubscribe:(NSUUID *)peerUUID
     characteristic:(CBCharacteristic *)chr
{
    CBPeripheral *periph = [self peripheralWithUUID:peerUUID];
    if (periph == NULL) {
        return -1;
    }

    [periph setNotifyValue:NO forCharacteristic:chr];
    return 0;
}

/**
 * Retrieves the RSSI of the connection to the specified peripheral.
 *
 * @return                      0 on success;
 *                              nonzero if peripheral not found.
 */
- (int)readRSSI:(NSUUID *)peerUUID
{
    CBPeripheral *periph = [self peripheralWithUUID:peerUUID];
    if (periph == NULL) {
        return -1;
    }

    [periph readRSSI];
    return 0;
}

@end
