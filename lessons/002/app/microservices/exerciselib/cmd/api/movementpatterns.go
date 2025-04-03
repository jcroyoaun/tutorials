package main

import (
	"fmt"
	"net/http"
	"time"

	"exerciselib.jcroyoaun.io/internal/data"
)

// Add a createMovementPatternHandler for the "POST /v1/movement-patterns" endpoint. For now we simply
// return a plain-text placeholder response.
func (app *application) createMovementPatternHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "create a new movement pattern")
}

// Add a showMovementPatternHandler for the "GET /v1/movement-patterns/:id" endpoint. For now, we retrieve
// the interpolated "id" parameter from the current URL and include it in a placeholder
// response.
func (app *application) showMovementPatternHandler(w http.ResponseWriter, r *http.Request) {
	//Movement m = new Movement
	id, err := app.readIDParam(r)

	if err != nil {
		http.NotFound(w, r)
		return
	}
	movement := data.Exercise{
		ID:        id,
		CreatedAt: time.Now(),
		// MovementPatternID:     1,
		Name:                  "Incline Bench Press",
		PrimaryMuscleGroups:   []string{"Chest", "AnteriorDelts"},
		SecondaryMuscleGroups: []string{"Triceps", "Middle Delts"},
		Version:               1,
	}

	movementPattern := data.MovementPattern{
		ID:                    id,
		CreatedAt:             time.Now(),
		PatternType:           data.MovementPatternHorizontalPush,
		PrimaryMuscleGroups:   []string{"Chest", "AnteriorDelts"},
		SecondaryMuscleGroups: []string{"Triceps", "Middle Delts"},
		// Add movement from Movement struct
		Movements: []data.Exercise{movement},
		Version:   1,
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"movementPattern": movementPattern}, nil)

	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) showMovementPatternsHandler(w http.ResponseWriter, r *http.Request) {

	allMovementPatterns := []data.MovementPatternType{
		data.MovementPatternSquat,
		data.MovementPatternHipHinge,
		data.MovementPatternVerticalPull,
		data.MovementPatternVerticalPush,
		data.MovementPatternHorizontalPull,
		data.MovementPatternHorizontalPush,
		data.MovementPatternHorizontalHipExt,
		data.MovementPatternPullOver,
		data.MovementPatternFly,
		data.MovementPatternIsolation,
	}

	err := app.writeJSON(w, http.StatusOK, envelope{"movementPatternTypes": allMovementPatterns}, nil)

	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
