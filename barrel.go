package game

import "fmt"

const (
	BarrelStandard uint8 = iota + 1
	BarrelHeavy
	BarrelRubber
	BarrelEnd
)

func DetermineAdditionalBounces(barrel_type uint8) int {
	switch barrel_type  {
	case BarrelRubber:
		return 1
	default:
		return 0
	}
}

func DetermineVelocityMultiplier(barrel_type uint8) float64 {
	switch barrel_type  {
	case BarrelHeavy:
		return 1.45
	case BarrelRubber:
		return 1
	default:
		return 1.2
	}
}

func DetermineBarrelName(barrel_type uint8) string {
	switch barrel_type {
	case BarrelStandard:
		return "standard"
	case BarrelHeavy:
		return "heavy"
	case BarrelRubber:
		return "rubber"
	default:
		return "missing!"
	}
}

func DetermineBarrelDesc(barrel_type uint8) string {
	switch barrel_type {
	case BarrelHeavy:
		return "Increases bullet velocity"
	case BarrelStandard:
		return "Standard barrel"
	case BarrelRubber:
		return "Bullet richochet an additional time\n at the expense of bullet velocity"
	default:
		return "missing!"
	}
}

func DetermineBarrelStats(barrel_type uint8) string {
	switch barrel_type {
	default:
		additional_bounces := DetermineAdditionalBounces(barrel_type)
		if additional_bounces != 0{
			return fmt.Sprintf("\n - Velocity multiplier: %.1fx\n - Bullet richochet modifier: %d",
				DetermineVelocityMultiplier(barrel_type), additional_bounces)
		}

		return fmt.Sprintf("\n - Velocity multiplier: %.1fx",
			DetermineVelocityMultiplier(barrel_type))
	}
}
