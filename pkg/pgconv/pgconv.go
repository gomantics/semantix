// Package pgconv provides utilities for converting between PostgreSQL types and Go types.
package pgconv

import (
	"database/sql"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// ToText converts a *string to pgtype.Text.
// Returns an invalid Text if s is nil.
func ToText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

// FromText converts pgtype.Text to *string.
// Returns nil if the Text is not valid.
func FromText(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}

// ToInt4 converts an *int32 to pgtype.Int4.
// Returns an invalid Int4 if i is nil.
func ToInt4(i *int32) pgtype.Int4 {
	if i == nil {
		return pgtype.Int4{Valid: false}
	}
	return pgtype.Int4{Int32: *i, Valid: true}
}

// FromInt4 converts pgtype.Int4 to *int32.
// Returns nil if the Int4 is not valid.
func FromInt4(i pgtype.Int4) *int32 {
	if !i.Valid {
		return nil
	}
	return &i.Int32
}

// ToInt8 converts an *int64 to pgtype.Int8.
// Returns an invalid Int8 if i is nil.
func ToInt8(i *int64) pgtype.Int8 {
	if i == nil {
		return pgtype.Int8{Valid: false}
	}
	return pgtype.Int8{Int64: *i, Valid: true}
}

// FromInt8 converts pgtype.Int8 to *int64.
// Returns nil if the Int8 is not valid.
func FromInt8(i pgtype.Int8) *int64 {
	if !i.Valid {
		return nil
	}
	return &i.Int64
}

// ToBool converts a *bool to pgtype.Bool.
// Returns an invalid Bool if b is nil.
func ToBool(b *bool) pgtype.Bool {
	if b == nil {
		return pgtype.Bool{Valid: false}
	}
	return pgtype.Bool{Bool: *b, Valid: true}
}

// FromBool converts pgtype.Bool to *bool.
// Returns nil if the Bool is not valid.
func FromBool(b pgtype.Bool) *bool {
	if !b.Valid {
		return nil
	}
	return &b.Bool
}

// ToFloat8 converts a *float64 to pgtype.Float8.
// Returns an invalid Float8 if f is nil.
func ToFloat8(f *float64) pgtype.Float8 {
	if f == nil {
		return pgtype.Float8{Valid: false}
	}
	return pgtype.Float8{Float64: *f, Valid: true}
}

// FromFloat8 converts pgtype.Float8 to *float64.
// Returns nil if the Float8 is not valid.
func FromFloat8(f pgtype.Float8) *float64 {
	if !f.Valid {
		return nil
	}
	return &f.Float64
}

// ToTimestamp converts a *time.Time to pgtype.Timestamp.
// Returns an invalid Timestamp if t is nil.
func ToTimestamp(t *time.Time) pgtype.Timestamp {
	if t == nil {
		return pgtype.Timestamp{Valid: false}
	}
	return pgtype.Timestamp{Time: *t, Valid: true}
}

// FromTimestamp converts pgtype.Timestamp to *time.Time.
// Returns nil if the Timestamp is not valid.
func FromTimestamp(t pgtype.Timestamp) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

// ToTimestamptz converts a *time.Time to pgtype.Timestamptz.
// Returns an invalid Timestamptz if t is nil.
func ToTimestamptz(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

// FromTimestamptz converts pgtype.Timestamptz to *time.Time.
// Returns nil if the Timestamptz is not valid.
func FromTimestamptz(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

// ToDate converts a *time.Time to pgtype.Date.
// Returns an invalid Date if t is nil.
func ToDate(t *time.Time) pgtype.Date {
	if t == nil {
		return pgtype.Date{Valid: false}
	}
	return pgtype.Date{Time: *t, Valid: true}
}

// FromDate converts pgtype.Date to *time.Time.
// Returns nil if the Date is not valid.
func FromDate(d pgtype.Date) *time.Time {
	if !d.Valid {
		return nil
	}
	return &d.Time
}

// ToNullString converts a *string to sql.NullString.
// Returns an invalid NullString if s is nil.
func ToNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: *s, Valid: true}
}

// FromNullString converts sql.NullString to *string.
// Returns nil if the NullString is not valid.
func FromNullString(s sql.NullString) *string {
	if !s.Valid {
		return nil
	}
	return &s.String
}

// ToNullInt64 converts an *int64 to sql.NullInt64.
// Returns an invalid NullInt64 if i is nil.
func ToNullInt64(i *int64) sql.NullInt64 {
	if i == nil {
		return sql.NullInt64{Valid: false}
	}
	return sql.NullInt64{Int64: *i, Valid: true}
}

// FromNullInt64 converts sql.NullInt64 to *int64.
// Returns nil if the NullInt64 is not valid.
func FromNullInt64(i sql.NullInt64) *int64 {
	if !i.Valid {
		return nil
	}
	return &i.Int64
}

// Ptr returns a pointer to the given value.
// Useful for creating inline pointers: pgconv.Ptr("hello")
func Ptr[T any](v T) *T {
	return &v
}

// Val returns the value from a pointer, or the zero value if nil.
func Val[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}

// ValOr returns the value from a pointer, or the default value if nil.
func ValOr[T any](p *T, def T) T {
	if p == nil {
		return def
	}
	return *p
}
