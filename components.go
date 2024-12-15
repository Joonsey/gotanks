package game

import (
	"log"
)

// Component masks
const (
	LoaderMask uint32 = 0xFF       // First 8 bits (1-8)
	BarrelMask uint32 = 0xFF00     // Second 8 bits (9-16)
	BulletMask uint32 = 0xFF0000   // Third 8 bits (17-24)
	TracksMask uint32 = 0xFF000000 // Fourth 8 bits (25-32)
)

type Component struct {
	Config uint32
}

// Set updates the specified component in the configuration.
func (c *Component) Set(mask uint32, value uint8) {
	// Clear the bits for the component
	c.Config &= ^mask
	// Set the new value, shifted to the appropriate position
	switch mask {
	case LoaderMask:
		c.Config |= uint32(value)
	case BarrelMask:
		c.Config |= uint32(value) << 8
	case BulletMask:
		c.Config |= uint32(value) << 16
	case TracksMask:
		c.Config |= uint32(value) << 24
	}
}

// Get retrieves the value of a specified component from the configuration.
func (c *Component) Get(mask uint32) uint8 {
	switch mask {
	case LoaderMask:
		return uint8(c.Config & mask)
	case BarrelMask:
		return uint8((c.Config & mask) >> 8)
	case BulletMask:
		return uint8((c.Config & mask) >> 16)
	case TracksMask:
		return uint8((c.Config & mask) >> 24)
	}
	return 0 // Default fallback
}

func (c *Component) LogConfiguration() {
	log.Printf("Loader: %d\n", c.Get(LoaderMask))
	log.Printf("Barrel: %d\n", c.Get(BarrelMask))
	log.Printf("Bullet: %d\n", c.Get(BulletMask))
	log.Printf("Tracks: %d\n", c.Get(TracksMask))
}

const (
	LoaderAutoloader uint8 = iota + 1 // Starts at 1 to avoid GOB's zero-value issue
	LoaderFastReload
	LoaderManualReload
	LoaderEnd
)

const (
	BarrelStandard uint8 = iota + 1
	BarrelSniper
	BarrelHeavy
	BarrelEnd
)

const (
	TracksLight uint8 = iota + 1
	TracksMedium
	TracksHeavy
	TracksEnd
)
