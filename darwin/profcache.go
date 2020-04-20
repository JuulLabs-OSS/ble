package darwin

// Profile Cache

import (
	"fmt"

	"github.com/JuulLabs-OSS/ble"
	"github.com/JuulLabs-OSS/cbgo"
)

type profCache struct {
	svcCbMap map[*ble.Service]cbgo.Service
	chrCbMap map[*ble.Characteristic]cbgo.Characteristic
	dscCbMap map[*ble.Descriptor]cbgo.Descriptor
}

func newProfCache() profCache {
	return profCache{
		svcCbMap: map[*ble.Service]cbgo.Service{},
		chrCbMap: map[*ble.Characteristic]cbgo.Characteristic{},
		dscCbMap: map[*ble.Descriptor]cbgo.Descriptor{},
	}
}

func (p *profCache) addSvc(s *ble.Service, cbs cbgo.Service) {
	p.svcCbMap[s] = cbs
}

func (p *profCache) addChr(c *ble.Characteristic, cbc cbgo.Characteristic) {
	p.chrCbMap[c] = cbc
}

func (p *profCache) addDsc(d *ble.Descriptor, cbd cbgo.Descriptor) {
	p.dscCbMap[d] = cbd
}

func (pc *profCache) findCbService(s *ble.Service) (cbgo.Service, error) {
	cbs, ok := pc.svcCbMap[s]
	if !ok {
		return cbs, fmt.Errorf("no service with UUID=%v", s.UUID)
	}

	return cbs, nil
}

func (pc *profCache) findCbCharacteristic(c *ble.Characteristic) (cbgo.Characteristic, error) {
	cbc, ok := pc.chrCbMap[c]
	if !ok {
		return cbc, fmt.Errorf("no characteristic with UUID=%v", c.UUID)
	}

	return cbc, nil
}

func (pc *profCache) findCbDescriptor(d *ble.Descriptor) (cbgo.Descriptor, error) {
	cbd, ok := pc.dscCbMap[d]
	if !ok {
		return cbd, fmt.Errorf("no descriptor with UUID=%v", d.UUID)
	}

	return cbd, nil
}
