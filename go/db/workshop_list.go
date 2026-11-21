// package: db / database access and repository layer
// type:    data
// job:     the names of an institution's workshops, for v1 code; listing workshops is the core's (core/workshop.go).
// limits:  does not create or mutate individual workshops (-> workshop.go).
package db

import (
	"context"

	"github.com/google/uuid"
)

// ListWorkshopsForInstitution returns a lightweight list of workshops for an institution (no permission checks).
func ListWorkshopsForInstitution(ctx context.Context, institutionID uuid.UUID) ([]WorkshopSummary, error) {
	rows, err := queries().ListWorkshopsByInstitution(ctx, institutionID)
	if err != nil {
		return nil, err
	}
	result := make([]WorkshopSummary, len(rows))
	for i, r := range rows {
		result[i] = WorkshopSummary{ID: r.ID, Name: r.Name}
	}
	return result, nil
}
