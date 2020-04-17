#import "bt.h"

struct byte_arr
nsdata_to_byte_arr(const NSData *nsdata)
{
    return (struct byte_arr) {
        .data = [nsdata bytes],
        .length = [nsdata length],
    };
}

NSData *
byte_arr_to_nsdata(const struct byte_arr *ba)
{
    return [NSData dataWithBytes: ba->data length: ba->length];
}

NSString *
str_to_nsstring(const char *s)
{
    return [[NSString alloc] initWithCString:s encoding:NSUTF8StringEncoding];
}

NSUUID *
str_to_nsuuid(const char *s)
{
    NSString *nss = str_to_nsstring(s);
    return [[NSUUID alloc] initWithUUIDString:nss];
}

CBUUID *
str_to_cbuuid(const char *s)
{
    NSString *nss = str_to_nsstring(s);
    return [CBUUID UUIDWithString:nss];
}

int 
dict_int(NSDictionary *dict, NSString *key)
{
    return [[dict objectForKey:key] intValue];
}

const char *
dict_string(NSDictionary *dict, NSString *key)
{
    return [[dict objectForKey:key] UTF8String];
}

const void *
dict_data(NSDictionary *dict, NSString *key, int *out_len)
{
    NSData *data;

    data = [dict objectForKey:key];

    *out_len = [data length];
    return [data bytes];
}

const struct byte_arr
dict_bytes(NSDictionary *dict, NSString *key)
{
    const NSData *nsdata = [dict objectForKey:key];
    return nsdata_to_byte_arr(nsdata);
}
