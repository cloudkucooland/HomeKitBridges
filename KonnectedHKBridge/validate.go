package konnectedkhbridge

import (
	"fmt"
	"net"
	"strings"
)

// Validate checks the top-level configuration
func (c *Config) Validate() error {
	if c.Pin == "" {
		c.Pin = "80899303" // Default HomeKit Pin
	}

	// Ensure ListenAddr is valid if provided, or can be resolved
	if c.ListenAddr != "" {
		host, port, err := net.SplitHostPort(c.ListenAddr)
		if err != nil {
			return fmt.Errorf("invalid ListenAddr (expected ip:port): %w", err)
		}
		if port == "" {
			return fmt.Errorf("ListenAddr requires a port")
		}
		// Basic check if IP is valid
		if host != "" && net.ParseIP(host) == nil {
			return fmt.Errorf("invalid IP in ListenAddr: %s", host)
		}
	}

	if len(c.Devices) == 0 {
		return fmt.Errorf("no devices configured")
	}

	// Validate individual devices
	macs := make(map[string]bool)
	for i := range c.Devices {
		if err := c.Devices[i].Validate(); err != nil {
			return fmt.Errorf("device %d: %w", i, err)
		}

		// Check for duplicate MACs
		mac := strings.ToLower(c.Devices[i].Mac)
		if macs[mac] {
			return fmt.Errorf("duplicate MAC address found: %s", mac)
		}
		macs[mac] = true
	}

	return nil
}

// Validate checks individual device settings
func (d *Device) Validate() error {
	if d.Mac == "" {
		return fmt.Errorf("MAC address is required")
	}

	// Normalize MAC (remove colons/dashes if necessary, but keep it consistent)
	d.Mac = strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(d.Mac, ":", ""), "-", ""))
	if len(d.Mac) != 12 {
		return fmt.Errorf("invalid MAC address length: %s", d.Mac)
	}

	if len(d.Zones) == 0 {
		return fmt.Errorf("device %s has no zones configured", d.Mac)
	}

	pins := make(map[uint8]bool)
	for _, z := range d.Zones {
		if z.Pin == 0 {
			// Some Konnected boards use Pin 0, but usually it's a mistake in config
			// We'll allow it but keep an eye on it.
		}
		if pins[z.Pin] {
			return fmt.Errorf("duplicate pin %d in device %s", z.Pin, d.Mac)
		}
		pins[z.Pin] = true

		switch z.Type {
		case "motion", "door", "buzzer", "unused":
			// valid
		default:
			return fmt.Errorf("unsupported zone type '%s' for pin %d", z.Type, z.Pin)
		}
	}

	return nil
}
