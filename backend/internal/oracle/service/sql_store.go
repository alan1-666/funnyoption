package service

import (
	"context"
	"database/sql"
	"encoding/json"
)

type SQLStore struct {
	db *sql.DB
}

func NewSQLStore(db *sql.DB) *SQLStore {
	return &SQLStore{db: db}
}

func (s *SQLStore) ListEligibleMarkets(ctx context.Context, now int64, limit int) ([]EligibleMarket, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			m.market_id,
			m.resolve_at,
			m.status,
			c.category_key,
			m.metadata,
			COALESCE(o.option_schema, '[]'::jsonb),
			COALESCE(r.status, ''),
			COALESCE(r.resolved_outcome, ''),
			COALESCE(r.resolver_type, ''),
			COALESCE(r.resolver_ref, ''),
			COALESCE(r.evidence, '{}'::jsonb)
		FROM markets m
		INNER JOIN market_categories c ON c.category_id = m.category_id
		LEFT JOIN market_option_sets o ON o.market_id = m.market_id
		LEFT JOIN market_resolutions r ON r.market_id = m.market_id
		WHERE m.resolve_at > 0
		  AND m.resolve_at <= $1
		  AND m.status <> 'RESOLVED'
		  AND c.category_key = 'CRYPTO'
		  AND COALESCE(m.metadata->'resolution'->>'mode', '') = 'ORACLE_PRICE'
		  AND (r.market_id IS NULL OR r.status IN ('', 'PENDING', 'RETRYABLE_ERROR', 'TERMINAL_ERROR', 'OBSERVED'))
		ORDER BY m.resolve_at ASC, m.market_id ASC
		LIMIT $2
	`, now, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	markets := make([]EligibleMarket, 0)
	for rows.Next() {
		var item EligibleMarket
		if err := rows.Scan(
			&item.MarketID,
			&item.ResolveAt,
			&item.MarketStatus,
			&item.CategoryKey,
			&item.Metadata,
			&item.OptionSchema,
			&item.ResolutionStatus,
			&item.ResolvedOutcome,
			&item.ResolverType,
			&item.ResolverRef,
			&item.Evidence,
		); err != nil {
			return nil, err
		}
		item.Metadata = normalizeJSONRaw(item.Metadata)
		item.OptionSchema = normalizeJSONRaw(item.OptionSchema)
		item.Evidence = normalizeJSONRaw(item.Evidence)
		markets = append(markets, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return markets, nil
}

func (s *SQLStore) RollupFrozen(ctx context.Context) (bool, error) {
	var frozen bool
	err := s.db.QueryRowContext(ctx, `
		SELECT frozen
		FROM rollup_freeze_state
		ORDER BY id DESC
		LIMIT 1
	`).Scan(&frozen)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return frozen, nil
}

func (s *SQLStore) UpsertResolution(ctx context.Context, update ResolutionUpdate) error {
	evidence := normalizeResolutionEvidence(update.Evidence)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO market_resolutions (
			market_id, status, resolved_outcome, resolver_type, resolver_ref, evidence, created_at, updated_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6,
			EXTRACT(EPOCH FROM NOW())::BIGINT,
			EXTRACT(EPOCH FROM NOW())::BIGINT
		)
		ON CONFLICT (market_id) DO UPDATE
		SET status = EXCLUDED.status,
			resolved_outcome = EXCLUDED.resolved_outcome,
			resolver_type = EXCLUDED.resolver_type,
			resolver_ref = EXCLUDED.resolver_ref,
			evidence = EXCLUDED.evidence,
			updated_at = EXCLUDED.updated_at
	`,
		update.MarketID,
		update.Status,
		update.ResolvedOutcome,
		update.ResolverType,
		update.ResolverRef,
		evidence,
	)
	return err
}

func normalizeResolutionEvidence(raw json.RawMessage) []byte {
	if len(raw) == 0 {
		return []byte(`{}`)
	}
	return raw
}

func normalizeJSONRaw(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(`{}`)
	}
	return raw
}
