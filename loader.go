package game

import "fmt"

const (
	LoaderAutoloader uint8 = iota + 1 // Starts at 1 to avoid GOB's zero-value issue
	LoaderFastReload
	LoaderManualReload
	LoaderEnd
)

func DetermineMaxMagMultiplier(loader_type uint8) float64 {
	switch loader_type {
	case LoaderAutoloader:
		return 2
	default:
		return 1
	}
}

func DetermineReloadSpeedMultiplier(loader_type uint8) float64 {
	switch loader_type {
	case LoaderAutoloader:
		return 5
	case LoaderManualReload:
		return 2
	default:
		return 1
	}
}

func DetermineLoaderName(loader_type uint8) string {
	switch loader_type {
	case LoaderAutoloader:
		return "autoloader"
	case LoaderFastReload:
		return "standard"
	case LoaderManualReload:
		return "manual"
	default:
		return "missing!"
	}
}

func DetermineLoaderDesc(loader_type uint8) string {
	switch loader_type {
	case LoaderAutoloader:
		return "Increases magazine size,\n at the cost of reload speed.\n Must empty magazine to reload"
	case LoaderFastReload:
		return "Standard loader"
	case LoaderManualReload:
		return "Manual reload"
	default:
		return "missing!"
	}
}

func DetermineLoaderStats(loader_type uint8) string {
	switch loader_type {
	default:
		return fmt.Sprintf("\n - Magazine: %.1fx\n - Reload speed: %.1fx",
			DetermineMaxMagMultiplier(loader_type), DetermineReloadSpeedMultiplier(loader_type))
	}
}
