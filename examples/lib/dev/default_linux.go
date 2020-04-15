package dev

import (
	"github.com/JuulLabs-OSS/ble"
	"github.com/JuulLabs-OSS/ble/linux"
)

// DefaultDevice ...
func DefaultDevice(opts ...ble.Option) (d ble.Device, err error) {
	return linux.NewDevice(opts...)
}
