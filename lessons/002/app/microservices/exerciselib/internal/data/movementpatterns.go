package data

import (
	"time"
)

// MovementPatternType defines the different movement pattern categories
type MovementPatternType string

// Define all possible movement pattern types as constants
const (
	MovementPatternSquat            MovementPatternType = "Squat"
	MovementPatternHipHinge         MovementPatternType = "Hip Hinge"
	MovementPatternVerticalPull     MovementPatternType = "Vertical Pull"
	MovementPatternVerticalPush     MovementPatternType = "Vertical Push"
	MovementPatternHorizontalPull   MovementPatternType = "Horizontal Pull"
	MovementPatternHorizontalPush   MovementPatternType = "Horizontal Push"
	MovementPatternHorizontalHipExt MovementPatternType = "Horizontal Hip Extension"
	MovementPatternPullOver         MovementPatternType = "Pull Over"
	MovementPatternFly              MovementPatternType = "Fly"
	MovementPatternIsolation        MovementPatternType = "Isolation"
)

// AllMovementPatternTypes returns a slice of all movement pattern types
func AllMovementPatternTypes() []MovementPatternType {
	return []MovementPatternType{
		MovementPatternSquat,
		MovementPatternHipHinge,
		MovementPatternVerticalPull,
		MovementPatternVerticalPush,
		MovementPatternHorizontalPull,
		MovementPatternHorizontalPush,
		MovementPatternHorizontalHipExt,
		MovementPatternPullOver,
		MovementPatternFly,
		MovementPatternIsolation,
	}
}

type MovementPattern struct {
	ID                    int64               `json:"id"`
	CreatedAt             time.Time           `json:"-"`
	PatternType           MovementPatternType `json:"patternType"`
	PrimaryMuscleGroups   []string            `json:"primaryMuscles"`
	SecondaryMuscleGroups []string            `json:"secondaryMuscles"`
	Movements             []Exercise          `json:"movements,omitempty"`
	Version               int32               `json:"version,omitzero"`
}

// Add this for PostgreSQL:
//
// CREATE TYPE movement_pattern_type AS ENUM (
//     'Squat',
//     'Hip Hinge',
//     'Vertical Pull',
//     'Vertical Push',
//     'Horizontal Pull',
//     'Horizontal Push',
//     'Horizontal Hip Extension',
//     'Pull Over',
//     'Fly',
//     'Isolation'
// );
