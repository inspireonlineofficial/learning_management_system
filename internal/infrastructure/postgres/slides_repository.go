package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	domainslides "lms-backend/internal/domain/slides"

	"github.com/google/uuid"
)

type PromotionalSlideRepository struct {
	db *sql.DB
}

func NewPromotionalSlideRepository(db *sql.DB) *PromotionalSlideRepository {
	return &PromotionalSlideRepository{db: db}
}

func (r *PromotionalSlideRepository) Create(ctx context.Context, slide *domainslides.PromotionalSlide) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO promotional_slides
			(id, title, subtitle, link_url, media_key, media_type, duration_ms, position, is_active, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		slide.ID, slide.Title, slide.Subtitle, slide.LinkURL, slide.MediaKey, slide.MediaType,
		slide.DurationMS, slide.Position, slide.IsActive, slide.CreatedAt, slide.UpdatedAt)
	return err
}

func (r *PromotionalSlideRepository) FindByID(ctx context.Context, id uuid.UUID) (*domainslides.PromotionalSlide, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, title, subtitle, link_url, media_key, media_type, duration_ms, position, is_active, created_at, updated_at, deactivated_at
		FROM promotional_slides WHERE id=$1`, id)
	return scanPromotionalSlide(row)
}

func (r *PromotionalSlideRepository) Update(ctx context.Context, slide *domainslides.PromotionalSlide) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE promotional_slides SET
			title=$2, subtitle=$3, link_url=$4, media_key=$5, media_type=$6,
			duration_ms=$7, position=$8, is_active=$9, updated_at=$10, deactivated_at=$11
		WHERE id=$1`,
		slide.ID, slide.Title, slide.Subtitle, slide.LinkURL, slide.MediaKey, slide.MediaType,
		slide.DurationMS, slide.Position, slide.IsActive, slide.UpdatedAt, slide.DeactivatedAt)
	return err
}

func (r *PromotionalSlideRepository) List(ctx context.Context, activeOnly bool) ([]*domainslides.PromotionalSlide, error) {
	where := ""
	if activeOnly {
		where = "WHERE is_active = true"
	}
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT id, title, subtitle, link_url, media_key, media_type, duration_ms, position, is_active, created_at, updated_at, deactivated_at
		FROM promotional_slides %s
		ORDER BY position ASC, created_at DESC`, where))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	slides := []*domainslides.PromotionalSlide{}
	for rows.Next() {
		slide, err := scanPromotionalSlideRow(rows)
		if err != nil {
			return nil, err
		}
		slides = append(slides, slide)
	}
	return slides, rows.Err()
}

func (r *PromotionalSlideRepository) Reorder(ctx context.Context, positions map[uuid.UUID]int) error {
	if len(positions) == 0 {
		return nil
	}
	args := make([]interface{}, 0, len(positions)*2)
	cases := make([]string, 0, len(positions))
	ids := make([]string, 0, len(positions))
	i := 1
	for id, position := range positions {
		cases = append(cases, fmt.Sprintf("WHEN id = $%d THEN $%d", i, i+1))
		ids = append(ids, "$"+strconv.Itoa(i))
		args = append(args, id, position)
		i += 2
	}
	query := fmt.Sprintf("UPDATE promotional_slides SET position = CASE %s ELSE position END, updated_at = $%d WHERE id IN (%s)",
		strings.Join(cases, " "), i, strings.Join(ids, ","))
	args = append(args, time.Now().UTC())
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func scanPromotionalSlide(row *sql.Row) (*domainslides.PromotionalSlide, error) {
	var slide domainslides.PromotionalSlide
	err := row.Scan(&slide.ID, &slide.Title, &slide.Subtitle, &slide.LinkURL, &slide.MediaKey, &slide.MediaType,
		&slide.DurationMS, &slide.Position, &slide.IsActive, &slide.CreatedAt, &slide.UpdatedAt, &slide.DeactivatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &slide, err
}

func scanPromotionalSlideRow(rows *sql.Rows) (*domainslides.PromotionalSlide, error) {
	var slide domainslides.PromotionalSlide
	err := rows.Scan(&slide.ID, &slide.Title, &slide.Subtitle, &slide.LinkURL, &slide.MediaKey, &slide.MediaType,
		&slide.DurationMS, &slide.Position, &slide.IsActive, &slide.CreatedAt, &slide.UpdatedAt, &slide.DeactivatedAt)
	return &slide, err
}
