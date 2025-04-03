package main

import (
	"errors"
	"fmt"
	"net/http"

	"exerciselib.jcroyoaun.io/internal/data"
	"exerciselib.jcroyoaun.io/internal/validator"
)

// Add a createExerciseHandler for the "POST /v1/exercises" endpoint. For now we simply
// return a plain-text placeholder response.
func (app *application) createExerciseHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		//MovementPatternID     int32    `json:"movementPatternId"`
		Name                  string   `json:"name"`
		PrimaryMuscleGroups   []string `json:"primaryMuscles"`
		SecondaryMuscleGroups []string `json:"secondaryMuscles"`
	}

	err := app.readJSON(w, r, &input)

	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	exercise := &data.Exercise{
		Name:                  input.Name,
		PrimaryMuscleGroups:   input.PrimaryMuscleGroups,
		SecondaryMuscleGroups: input.SecondaryMuscleGroups,
	}

	v := validator.New()

	if data.ValidateExercise(v, exercise); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.models.Exercises.Insert(exercise)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/exercises/%d", exercise.ID))

	err = app.writeJSON(w, http.StatusCreated, envelope{"exercise": exercise}, headers)

	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

}

// Add a showExerciseHandler for the "GET /v1/exercises/:id" endpoint. For now, we retrieve
// the interpolated "id" parameter from the current URL and include it in a placeholder
// response.

func (app *application) showExerciseHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)

	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	exercise, err := app.models.Exercises.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"exercise": exercise}, nil)

	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) updateExerciseHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the Exercise ID from the URL.
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	exercise, err := app.models.Exercises.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	var input struct {
		Name                  *string  `json:"name"`
		PrimaryMuscleGroups   []string `json:"primaryMuscles"`
		SecondaryMuscleGroups []string `json:"secondaryMuscles"`
	}

	err = app.readJSON(w, r, &input)

	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if input.Name != nil {
		exercise.Name = *input.Name
	}

	if input.PrimaryMuscleGroups != nil {
		exercise.PrimaryMuscleGroups = input.PrimaryMuscleGroups
	}

	if input.SecondaryMuscleGroups != nil {
		exercise.SecondaryMuscleGroups = input.SecondaryMuscleGroups
	}

	v := validator.New()

	if data.ValidateExercise(v, exercise); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.models.Exercises.Update(exercise)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"exercise": exercise}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) deleteExerciseHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the exercise ID from the URL.
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	// Delete the exercise from the database, sending a 404 Not Found response to the
	// client if there isn't a matching record.
	err = app.models.Exercises.Delete(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Return a 200 OK status code along with a success message.
	err = app.writeJSON(w, http.StatusOK, envelope{"message": "exercise successfully deleted"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) listExerciseHandle(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name                  string
		PrimaryMuscleGroups   []string
		SecondaryMuscleGroups []string
		data.Filters
	}

	v := validator.New()

	qs := r.URL.Query()

	input.Name = app.readString(qs, "name", "")
	input.PrimaryMuscleGroups = app.readCSV(qs, "primaryMuscles", []string{})
	input.SecondaryMuscleGroups = app.readCSV(qs, "secondaryMuscles", []string{})

	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 20, v)

	input.Filters.Sort = app.readString(qs, "sort", "id")
	// Add the supported sort values for this endpoint to the sort safelist.
	input.Filters.SortSafelist = []string{"id", "name", "primarymuscles", "secondarymuscles", "-id", "-name"}

	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	exercises, err := app.models.Exercises.GetAll(input.Name, input.PrimaryMuscleGroups, input.SecondaryMuscleGroups, input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"exercises": exercises}, nil)

	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
