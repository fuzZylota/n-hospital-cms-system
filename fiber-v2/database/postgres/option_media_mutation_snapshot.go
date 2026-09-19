package postgres

import (
	"context"
	"database/sql"
	"strconv"

	"models/data"
)

const readOptionMediaMutationSnapshotSQL = `SELECT oid, option_set_is_active, option_set_is_testing_now, max_upload_size, site_logo_mid, site_light_logo_mid, site_favicon_mid, default_page_mid
FROM options
WHERE option_set_is_active = TRUE`

var _ data.OptionMediaMutationSnapshotReader = (*OptionsRepository)(nil)

// ReadOptionMediaMutationSnapshot reads all option-media mutation inputs from
// one active options row and one SQL statement.
func (r *OptionsRepository) ReadOptionMediaMutationSnapshot(ctx context.Context) (data.OptionMediaMutationSnapshot, bool, error) {
	var identity optionIdentityRow
	var maxBytes, siteLogoID, siteLightLogoID, faviconID, defaultPageMediaID sql.NullInt64
	found, err := r.readOne(ctx, readOptionMediaMutationSnapshot, readOptionMediaMutationSnapshotSQL, func(rows *sql.Rows) error {
		return rows.Scan(
			&identity.id, &identity.active, &identity.testing, &maxBytes,
			&siteLogoID, &siteLightLogoID, &faviconID, &defaultPageMediaID,
		)
	})
	if err != nil {
		return data.OptionMediaMutationSnapshot{}, false, strictOptionMediaMutationSnapshotError(ctx, err)
	}
	if !found {
		return data.OptionMediaMutationSnapshot{}, false, nil
	}
	set, err := activeOptionIdentity(identity, readOptionMediaMutationSnapshot)
	if err != nil {
		return data.OptionMediaMutationSnapshot{}, false, strictOptionMediaMutationSnapshotError(ctx, err)
	}

	snapshot := data.OptionMediaMutationSnapshot{Set: set, MaxBytes: maxBytes.Int64}
	mediaIDs := []struct {
		source sql.NullInt64
		target **string
	}{
		{source: siteLogoID, target: &snapshot.SiteLogoID},
		{source: siteLightLogoID, target: &snapshot.SiteLightLogoID},
		{source: faviconID, target: &snapshot.FaviconID},
		{source: defaultPageMediaID, target: &snapshot.DefaultPageMediaID},
	}
	for _, mediaID := range mediaIDs {
		if !mediaID.source.Valid {
			continue
		}
		if mediaID.source.Int64 <= 0 {
			return data.OptionMediaMutationSnapshot{}, false, newStrictOptionMediaMutationSnapshotError(ctx, readInvalidIdentifier)
		}
		value := strconv.FormatInt(mediaID.source.Int64, 10)
		*mediaID.target = &value
	}
	return snapshot, true, nil
}

func strictOptionMediaMutationSnapshotError(ctx context.Context, err error) error {
	repositoryErr, ok := err.(*repositoryReadError)
	if !ok || repositoryErr == nil {
		return newStrictOptionMediaMutationSnapshotError(ctx, readDependency)
	}
	return newStrictOptionMediaMutationSnapshotError(ctx, repositoryErr.stage)
}

func newStrictOptionMediaMutationSnapshotError(ctx context.Context, stage readStage) error {
	result := &repositoryReadError{operation: readOptionMediaMutationSnapshot, stage: stage}
	if ctx == nil {
		return result
	}
	switch ctx.Err() {
	case context.Canceled:
		result.contextErr = context.Canceled
	case context.DeadlineExceeded:
		result.contextErr = context.DeadlineExceeded
	}
	return result
}
