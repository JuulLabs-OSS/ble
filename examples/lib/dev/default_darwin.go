package dev

import (
	"github.com/JuulLabs-OSS/ble"
	"github.com/JuulLabs-OSS/ble/darwin"
)

// DefaultDevice ...
func DefaultDevice(opts ...ble.Option) (d ble.Device, err error) {
	return darwin.NewDevice(opts...)
}
