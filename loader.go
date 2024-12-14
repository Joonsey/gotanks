package game

func DetermineMaxMagMultiplier(loader_type uint8) int {
	switch loader_type {
	case LoaderAutoloader:
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
