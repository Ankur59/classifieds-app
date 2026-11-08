package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ankur59/classifieds-app/internal/models"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	// ErrNotFound is returned when a lookup matches no rows.
	ErrNotFound = errors.New("not found")
	// ErrConflict is returned when a write violates a unique constraint.
	ErrConflict = errors.New("already exists")
)

// pgUniqueViolation is the SQLSTATE Postgres reports for duplicate keys.
const pgUniqueViolation = "23505"

// StudentRepository wraps all SQL for the students table.
type StudentRepository struct {
	db *sql.DB
}

func NewStudentRepository(db *sql.DB) *StudentRepository {
	return &StudentRepository{db: db}
}

func (r *StudentRepository) Create(ctx context.Context, s models.Student) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO students (id, name) VALUES ($1, $2)`, s.ID, s.Name)
	if isUniqueViolation(err) {
		return ErrConflict
	}
	if err != nil {
		return fmt.Errorf("students.create: %w", err)
	}
	return nil
}

func (r *StudentRepository) GetByID(ctx context.Context, id int) (models.Student, error) {
	var s models.Student
	err := r.db.QueryRowContext(ctx, `SELECT id, name FROM students WHERE id = $1`, id).Scan(&s.ID, &s.Name)
	if errors.Is(err, sql.ErrNoRows) {
		return s, ErrNotFound
	}
	if err != nil {
		return s, fmt.Errorf("students.getByID: %w", err)
	}
	return s, nil
}

// List returns at most limit students ordered by id, skipping the first offset.
func (r *StudentRepository) List(ctx context.Context, limit, offset int) ([]models.Student, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name FROM students ORDER BY id LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("students.list: %w", err)
	}
	defer rows.Close()

	students := []models.Student{}
	for rows.Next() {
		var s models.Student
		if err := rows.Scan(&s.ID, &s.Name); err != nil {
			return nil, fmt.Errorf("students.list.scan: %w", err)
		}
		students = append(students, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("students.list.rows: %w", err)
	}
	return students, nil
}

// Update replaces the name of the student with s.ID.
func (r *StudentRepository) Update(ctx context.Context, s models.Student) error {
	res, err := r.db.ExecContext(ctx, `UPDATE students SET name = $2 WHERE id = $1`, s.ID, s.Name)
	if err != nil {
		return fmt.Errorf("students.update: %w", err)
	}
	return requireOneRow(res, "students.update")
}

func (r *StudentRepository) Delete(ctx context.Context, id int) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM students WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("students.delete: %w", err)
	}
	return requireOneRow(res, "students.delete")
}

// requireOneRow maps "no rows affected" to ErrNotFound.
func requireOneRow(res sql.Result, op string) error {
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s.rowsAffected: %w", op, err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func isUniqueViolation(err error) bool {
	pgErr, ok := errors.AsType[*pgconn.PgError](err)
	return ok && pgErr.Code == pgUniqueViolation
}
