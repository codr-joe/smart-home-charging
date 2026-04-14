package energy

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository provides access to energy readings in the database.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new Repository backed by the given connection pool.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Save persists a single energy reading.
func (r *Repository) Save(ctx context.Context, reading Reading) error {
	const q = `INSERT INTO energy_readings (time, power_w, solar_w, tariff) VALUES ($1, $2, $3, $4)`
	if _, err := r.pool.Exec(ctx, q, reading.Time, reading.PowerW, reading.SolarW, reading.Tariff); err != nil {
		return fmt.Errorf("insert energy reading: %w", err)
	}
	return nil
}

// Latest returns the most recent energy reading.
func (r *Repository) Latest(ctx context.Context) (Reading, error) {
	const q = `SELECT time, power_w, solar_w, tariff FROM energy_readings ORDER BY time DESC LIMIT 1`
	var rd Reading
	row := r.pool.QueryRow(ctx, q)
	if err := row.Scan(&rd.Time, &rd.PowerW, &rd.SolarW, &rd.Tariff); err != nil {
		return Reading{}, fmt.Errorf("query latest reading: %w", err)
	}
	return rd, nil
}

// History returns energy readings within the given time range, newest first.
func (r *Repository) History(ctx context.Context, from, to time.Time, limit int) ([]Reading, error) {
	const q = `SELECT time, power_w, solar_w, tariff FROM energy_readings WHERE time >= $1 AND time <= $2 ORDER BY time DESC LIMIT $3`
	rows, err := r.pool.Query(ctx, q, from, to, limit)
	if err != nil {
		return nil, fmt.Errorf("query history: %w", err)
	}
	defer rows.Close()
	var readings []Reading
	for rows.Next() {
		var rd Reading
		if err := rows.Scan(&rd.Time, &rd.PowerW, &rd.SolarW, &rd.Tariff); err != nil {
			return nil, fmt.Errorf("scan reading: %w", err)
		}
		readings = append(readings, rd)
	}
	return readings, rows.Err()
}

// BucketedHistory returns downsampled energy readings grouped into fixed-width time buckets,
// newest first. bucketSecs controls the bucket width in seconds (e.g. 120 for 2-minute buckets).
// power_w and solar_w are averaged within each bucket; tariff is the most recent value.
func (r *Repository) BucketedHistory(ctx context.Context, from, to time.Time, bucketSecs, limit int) ([]Reading, error) {
	const q = `
		SELECT
			to_timestamp(floor(extract(epoch from time) / $4) * $4) AS bucket_time,
			avg(power_w)::float8                                     AS power_w,
			avg(solar_w)::float8                                     AS solar_w,
			max(tariff)                                              AS tariff
		FROM energy_readings
		WHERE time >= $1 AND time <= $2
		GROUP BY bucket_time
		ORDER BY bucket_time DESC
		LIMIT $3`
	rows, err := r.pool.Query(ctx, q, from, to, limit, bucketSecs)
	if err != nil {
		return nil, fmt.Errorf("query bucketed history: %w", err)
	}
	defer rows.Close()
	var readings []Reading
	for rows.Next() {
		var rd Reading
		if err := rows.Scan(&rd.Time, &rd.PowerW, &rd.SolarW, &rd.Tariff); err != nil {
			return nil, fmt.Errorf("scan bucketed reading: %w", err)
		}
		readings = append(readings, rd)
	}
	return readings, rows.Err()
}
