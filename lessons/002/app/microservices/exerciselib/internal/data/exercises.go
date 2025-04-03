package data

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"exerciselib.jcroyoaun.io/internal/validator"
	"github.com/lib/pq"
)

type Exercise struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"-"`
	// MovementPatternID     int64     `json:"movementPatternId"` // Foreign key reference to MovementPattern
	Name                  string   `json:"name"`             // e.g., "Paused Front Squat"
	PrimaryMuscleGroups   []string `json:"primaryMuscles"`   // e.g., ["Quads", "Glutes", "Hamstrings"]
	SecondaryMuscleGroups []string `json:"secondaryMuscles"` // e.g., ["Scapular Retractors"]
	Version               int32    `json:"version,omitzero"`
}

type ExerciseModel struct {
	DB *sql.DB
}

func (e ExerciseModel) Insert(exercise *Exercise) error {
	query := `
        INSERT INTO exercises(name, primarymuscles, secondarymuscles)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, version`

	args := []any{exercise.Name, pq.Array(exercise.PrimaryMuscleGroups), pq.Array(exercise.SecondaryMuscleGroups)}

	// Create a context with a 3-second timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return e.DB.QueryRowContext(ctx, query, args...).Scan(&exercise.ID, &exercise.CreatedAt, &exercise.Version)
}

func (e ExerciseModel) Get(id int64) (*Exercise, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}

	query := `
		SELECT id, created_at, name, primarymuscles, secondarymuscles, version
		FROM exercises
		WHERE id = $1`

	var exercise Exercise

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)

	defer cancel()

	err := e.DB.QueryRowContext(ctx, query, id).Scan(
		&exercise.ID,
		&exercise.CreatedAt,
		&exercise.Name,
		pq.Array(&exercise.PrimaryMuscleGroups),
		pq.Array(&exercise.SecondaryMuscleGroups),
		&exercise.Version,
	)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &exercise, nil
}

func (e ExerciseModel) Update(exercise *Exercise) error {
	query := `
		UPDATE exercises
		SET name = $1, primarymuscles = $2, secondarymuscles = $3, version = version + 1
		WHERE id = $4 AND version = $5
		RETURNING version`

	args := []any{
		exercise.Name,
		pq.Array(exercise.PrimaryMuscleGroups),
		pq.Array(exercise.SecondaryMuscleGroups),
		exercise.ID,
		exercise.Version,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := e.DB.QueryRowContext(ctx, query, args...).Scan(&exercise.Version)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		default:
			return err
		}
	}

	return nil
}

func (e ExerciseModel) Delete(id int64) error {
	if id < 1 {
		return ErrRecordNotFound
	}

	query := `
		DELETE FROM exercises
		WHERE id = $1`

	// Create a context with a 3-second timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := e.DB.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrRecordNotFound
	}

	return nil
}

func (e ExerciseModel) GetAll(name string, primaryMuscles []string, secondaryMuscles []string, filters Filters) ([]*Exercise, error) {

	query := `
		SELECT id, created_at, name, primarymuscles, secondarymuscles, version
		FROM exercises
		WHERE (to_tsvector('simple', name) @@ plainto_tsquery('simple', $1) OR $1 = '') 
        AND (primarymuscles @> $2 OR $2 = '{}')
		AND (secondarymuscles @> $3 OR $3 = '{}')
		ORDER BY id`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := e.DB.QueryContext(ctx, query, name, pq.Array(primaryMuscles), pq.Array(secondaryMuscles))
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	exercises := []*Exercise{}

	for rows.Next() {
		var exercise Exercise

		err := rows.Scan(
			&exercise.ID,
			&exercise.CreatedAt,
			&exercise.Name,
			pq.Array(&exercise.PrimaryMuscleGroups),
			pq.Array(&exercise.SecondaryMuscleGroups),
			&exercise.Version,
		)

		if err != nil {
			return nil, err
		}

		exercises = append(exercises, &exercise)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return exercises, nil
}

func ValidateExercise(v *validator.Validator, exercise *Exercise) {
	v.Check(exercise.Name != "", "name", "must be provided")
	v.Check(len(exercise.Name) <= 500, "name", "must not be more than 500 bytes long")

	v.Check(exercise.PrimaryMuscleGroups != nil, "genres", "must be provided")
	v.Check(len(exercise.PrimaryMuscleGroups) >= 1, "genres", "must contain at least 1 genre")
	v.Check(len(exercise.PrimaryMuscleGroups) <= 5, "genres", "must not contain more than 5 genres")
	v.Check(validator.Unique(exercise.PrimaryMuscleGroups), "genres", "must not contain duplicate values")
}
