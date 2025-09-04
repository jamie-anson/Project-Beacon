package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// DiffsRepo provides persistence operations for cross-region diffs
type DiffsRepo struct {
	DB *sql.DB
}

func NewDiffsRepo(db *sql.DB) *DiffsRepo {
	return &DiffsRepo{DB: db}
}

// CreateDiff creates a new cross-region diff record
func (r *DiffsRepo) CreateDiff(ctx context.Context, diff *models.CrossRegionDiff) error {
	if r.DB == nil {
		return errors.New("database connection is nil")
	}

	// Serialize diff data
	diffDataJSON, err := json.Marshal(diff.DiffData)
	if err != nil {
		return fmt.Errorf("failed to marshal diff data: %w", err)
	}

	_, err = r.DB.ExecContext(ctx, `
		INSERT INTO diffs (job_id, region_a, region_b, similarity_score, diff_data, classification, created_at)
		VALUES ((SELECT id FROM jobs WHERE jobspec_id = $1), $2, $3, $4, $5, $6, $7)
	`, diff.JobSpecID, diff.RegionA, diff.RegionB, diff.SimilarityScore, 
		diffDataJSON, diff.Classification, diff.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to insert diff: %w", err)
	}

	return nil
}

// GetDiffsByJobSpecID returns all diffs for a specific JobSpec
func (r *DiffsRepo) GetDiffsByJobSpecID(ctx context.Context, jobspecID string) ([]*models.CrossRegionDiff, error) {
	if r.DB == nil {
		return nil, errors.New("database connection is nil")
	}

	rows, err := r.DB.QueryContext(ctx, `
		SELECT d.id, d.region_a, d.region_b, d.similarity_score, d.diff_data, d.classification, d.created_at
		FROM diffs d
		JOIN jobs j ON d.job_id = j.id
		WHERE j.jobspec_id = $1
		ORDER BY d.created_at DESC
	`, jobspecID)

	if err != nil {
		return nil, fmt.Errorf("failed to query diffs: %w", err)
	}
	defer rows.Close()

	var diffs []*models.CrossRegionDiff
	for rows.Next() {
		var diff models.CrossRegionDiff
		var diffDataJSON []byte
		var diffID int64

		err := rows.Scan(&diffID, &diff.RegionA, &diff.RegionB, &diff.SimilarityScore,
			&diffDataJSON, &diff.Classification, &diff.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan diff row: %w", err)
		}

		// Set the ID and JobSpecID
		diff.ID = fmt.Sprintf("%d", diffID)
		diff.JobSpecID = jobspecID

		// Unmarshal diff data
		if err := json.Unmarshal(diffDataJSON, &diff.DiffData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal diff data: %w", err)
		}

		diffs = append(diffs, &diff)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating diff rows: %w", err)
	}

	return diffs, nil
}

// GetDiffByRegions returns a specific diff between two regions for a JobSpec
func (r *DiffsRepo) GetDiffByRegions(ctx context.Context, jobspecID, regionA, regionB string) (*models.CrossRegionDiff, error) {
	if r.DB == nil {
		return nil, errors.New("database connection is nil")
	}

	var diff models.CrossRegionDiff
	var diffDataJSON []byte
	var diffID int64

	row := r.DB.QueryRowContext(ctx, `
		SELECT d.id, d.similarity_score, d.diff_data, d.classification, d.created_at
		FROM diffs d
		JOIN jobs j ON d.job_id = j.id
		WHERE j.jobspec_id = $1 AND d.region_a = $2 AND d.region_b = $3
		ORDER BY d.created_at DESC
		LIMIT 1
	`, jobspecID, regionA, regionB)

	err := row.Scan(&diffID, &diff.SimilarityScore, &diffDataJSON, &diff.Classification, &diff.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no diff found for job %s between regions %s and %s", jobspecID, regionA, regionB)
		}
		return nil, fmt.Errorf("failed to get diff: %w", err)
	}

	// Set the fields
	diff.ID = fmt.Sprintf("%d", diffID)
	diff.JobSpecID = jobspecID
	diff.RegionA = regionA
	diff.RegionB = regionB

	// Unmarshal diff data
	if err := json.Unmarshal(diffDataJSON, &diff.DiffData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal diff data: %w", err)
	}

	return &diff, nil
}

// ListSignificantDiffs returns diffs classified as significant
func (r *DiffsRepo) ListSignificantDiffs(ctx context.Context, limit int) ([]*models.CrossRegionDiff, error) {
	if r.DB == nil {
		return nil, errors.New("database connection is nil")
	}

	if limit <= 0 {
		limit = 50
	}

	rows, err := r.DB.QueryContext(ctx, `
		SELECT d.id, j.jobspec_id, d.region_a, d.region_b, d.similarity_score, d.diff_data, d.classification, d.created_at
		FROM diffs d
		JOIN jobs j ON d.job_id = j.id
		WHERE d.classification = 'significant'
		ORDER BY d.created_at DESC
		LIMIT $1
	`, limit)

	if err != nil {
		return nil, fmt.Errorf("failed to query significant diffs: %w", err)
	}
	defer rows.Close()

	var diffs []*models.CrossRegionDiff
	for rows.Next() {
		var diff models.CrossRegionDiff
		var diffDataJSON []byte
		var diffID int64

		err := rows.Scan(&diffID, &diff.JobSpecID, &diff.RegionA, &diff.RegionB,
			&diff.SimilarityScore, &diffDataJSON, &diff.Classification, &diff.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan diff row: %w", err)
		}

		diff.ID = fmt.Sprintf("%d", diffID)

		// Unmarshal diff data
		if err := json.Unmarshal(diffDataJSON, &diff.DiffData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal diff data: %w", err)
		}

		diffs = append(diffs, &diff)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating diff rows: %w", err)
	}

	return diffs, nil
}

// GetDiffStats returns statistics about diffs for a JobSpec
func (r *DiffsRepo) GetDiffStats(ctx context.Context, jobspecID string) (map[string]interface{}, error) {
	if r.DB == nil {
		return nil, errors.New("database connection is nil")
	}

	row := r.DB.QueryRowContext(ctx, `
		SELECT 
			COUNT(*) as total_diffs,
			AVG(similarity_score) as avg_similarity,
			MIN(similarity_score) as min_similarity,
			MAX(similarity_score) as max_similarity,
			COUNT(CASE WHEN classification = 'significant' THEN 1 END) as significant_count,
			COUNT(CASE WHEN classification = 'minor' THEN 1 END) as minor_count,
			COUNT(CASE WHEN classification = 'noise' THEN 1 END) as noise_count
		FROM diffs d
		JOIN jobs j ON d.job_id = j.id
		WHERE j.jobspec_id = $1
	`, jobspecID)

	var totalDiffs, significantCount, minorCount, noiseCount int64
	var avgSimilarity, minSimilarity, maxSimilarity sql.NullFloat64

	err := row.Scan(&totalDiffs, &avgSimilarity, &minSimilarity, &maxSimilarity,
		&significantCount, &minorCount, &noiseCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get diff stats: %w", err)
	}

	stats := map[string]interface{}{
		"total_diffs":        totalDiffs,
		"significant_count":  significantCount,
		"minor_count":        minorCount,
		"noise_count":        noiseCount,
	}

	if avgSimilarity.Valid {
		stats["avg_similarity"] = avgSimilarity.Float64
	}
	if minSimilarity.Valid {
		stats["min_similarity"] = minSimilarity.Float64
	}
	if maxSimilarity.Valid {
		stats["max_similarity"] = maxSimilarity.Float64
	}

	return stats, nil
}

// DeleteDiffsByJobSpecID deletes all diffs for a JobSpec
func (r *DiffsRepo) DeleteDiffsByJobSpecID(ctx context.Context, jobspecID string) error {
	if r.DB == nil {
		return errors.New("database connection is nil")
	}

	_, err := r.DB.ExecContext(ctx, `
		DELETE FROM diffs 
		WHERE job_id = (SELECT id FROM jobs WHERE jobspec_id = $1)
	`, jobspecID)

	return err
}
