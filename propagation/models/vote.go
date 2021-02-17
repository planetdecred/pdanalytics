// Code generated by SQLBoiler 3.7.1 (https://github.com/volatiletech/sqlboiler). DO NOT EDIT.
// This file is meant to be re-generated in place and/or deleted at any time.

package models

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/friendsofgo/errors"
	"github.com/volatiletech/null"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries"
	"github.com/volatiletech/sqlboiler/queries/qm"
	"github.com/volatiletech/sqlboiler/queries/qmhelper"
	"github.com/volatiletech/sqlboiler/strmangle"
)

// Vote is an object representing the database table.
type Vote struct {
	Hash              string      `boil:"hash" json:"hash" toml:"hash" yaml:"hash"`
	VotingOn          null.Int64  `boil:"voting_on" json:"voting_on,omitempty" toml:"voting_on" yaml:"voting_on,omitempty"`
	BlockHash         null.String `boil:"block_hash" json:"block_hash,omitempty" toml:"block_hash" yaml:"block_hash,omitempty"`
	ReceiveTime       null.Time   `boil:"receive_time" json:"receive_time,omitempty" toml:"receive_time" yaml:"receive_time,omitempty"`
	BlockReceiveTime  null.Time   `boil:"block_receive_time" json:"block_receive_time,omitempty" toml:"block_receive_time" yaml:"block_receive_time,omitempty"`
	TargetedBlockTime null.Time   `boil:"targeted_block_time" json:"targeted_block_time,omitempty" toml:"targeted_block_time" yaml:"targeted_block_time,omitempty"`
	ValidatorID       null.Int    `boil:"validator_id" json:"validator_id,omitempty" toml:"validator_id" yaml:"validator_id,omitempty"`
	Validity          null.String `boil:"validity" json:"validity,omitempty" toml:"validity" yaml:"validity,omitempty"`

	R *voteR `boil:"-" json:"-" toml:"-" yaml:"-"`
	L voteL  `boil:"-" json:"-" toml:"-" yaml:"-"`
}

var VoteColumns = struct {
	Hash              string
	VotingOn          string
	BlockHash         string
	ReceiveTime       string
	BlockReceiveTime  string
	TargetedBlockTime string
	ValidatorID       string
	Validity          string
}{
	Hash:              "hash",
	VotingOn:          "voting_on",
	BlockHash:         "block_hash",
	ReceiveTime:       "receive_time",
	BlockReceiveTime:  "block_receive_time",
	TargetedBlockTime: "targeted_block_time",
	ValidatorID:       "validator_id",
	Validity:          "validity",
}

// Generated where

type whereHelpernull_Int64 struct{ field string }

func (w whereHelpernull_Int64) EQ(x null.Int64) qm.QueryMod {
	return qmhelper.WhereNullEQ(w.field, false, x)
}
func (w whereHelpernull_Int64) NEQ(x null.Int64) qm.QueryMod {
	return qmhelper.WhereNullEQ(w.field, true, x)
}
func (w whereHelpernull_Int64) IsNull() qm.QueryMod    { return qmhelper.WhereIsNull(w.field) }
func (w whereHelpernull_Int64) IsNotNull() qm.QueryMod { return qmhelper.WhereIsNotNull(w.field) }
func (w whereHelpernull_Int64) LT(x null.Int64) qm.QueryMod {
	return qmhelper.Where(w.field, qmhelper.LT, x)
}
func (w whereHelpernull_Int64) LTE(x null.Int64) qm.QueryMod {
	return qmhelper.Where(w.field, qmhelper.LTE, x)
}
func (w whereHelpernull_Int64) GT(x null.Int64) qm.QueryMod {
	return qmhelper.Where(w.field, qmhelper.GT, x)
}
func (w whereHelpernull_Int64) GTE(x null.Int64) qm.QueryMod {
	return qmhelper.Where(w.field, qmhelper.GTE, x)
}

type whereHelpernull_Int struct{ field string }

func (w whereHelpernull_Int) EQ(x null.Int) qm.QueryMod {
	return qmhelper.WhereNullEQ(w.field, false, x)
}
func (w whereHelpernull_Int) NEQ(x null.Int) qm.QueryMod {
	return qmhelper.WhereNullEQ(w.field, true, x)
}
func (w whereHelpernull_Int) IsNull() qm.QueryMod    { return qmhelper.WhereIsNull(w.field) }
func (w whereHelpernull_Int) IsNotNull() qm.QueryMod { return qmhelper.WhereIsNotNull(w.field) }
func (w whereHelpernull_Int) LT(x null.Int) qm.QueryMod {
	return qmhelper.Where(w.field, qmhelper.LT, x)
}
func (w whereHelpernull_Int) LTE(x null.Int) qm.QueryMod {
	return qmhelper.Where(w.field, qmhelper.LTE, x)
}
func (w whereHelpernull_Int) GT(x null.Int) qm.QueryMod {
	return qmhelper.Where(w.field, qmhelper.GT, x)
}
func (w whereHelpernull_Int) GTE(x null.Int) qm.QueryMod {
	return qmhelper.Where(w.field, qmhelper.GTE, x)
}

var VoteWhere = struct {
	Hash              whereHelperstring
	VotingOn          whereHelpernull_Int64
	BlockHash         whereHelpernull_String
	ReceiveTime       whereHelpernull_Time
	BlockReceiveTime  whereHelpernull_Time
	TargetedBlockTime whereHelpernull_Time
	ValidatorID       whereHelpernull_Int
	Validity          whereHelpernull_String
}{
	Hash:              whereHelperstring{field: "\"vote\".\"hash\""},
	VotingOn:          whereHelpernull_Int64{field: "\"vote\".\"voting_on\""},
	BlockHash:         whereHelpernull_String{field: "\"vote\".\"block_hash\""},
	ReceiveTime:       whereHelpernull_Time{field: "\"vote\".\"receive_time\""},
	BlockReceiveTime:  whereHelpernull_Time{field: "\"vote\".\"block_receive_time\""},
	TargetedBlockTime: whereHelpernull_Time{field: "\"vote\".\"targeted_block_time\""},
	ValidatorID:       whereHelpernull_Int{field: "\"vote\".\"validator_id\""},
	Validity:          whereHelpernull_String{field: "\"vote\".\"validity\""},
}

// VoteRels is where relationship names are stored.
var VoteRels = struct {
}{}

// voteR is where relationships are stored.
type voteR struct {
}

// NewStruct creates a new relationship struct
func (*voteR) NewStruct() *voteR {
	return &voteR{}
}

// voteL is where Load methods for each relationship are stored.
type voteL struct{}

var (
	voteAllColumns            = []string{"hash", "voting_on", "block_hash", "receive_time", "block_receive_time", "targeted_block_time", "validator_id", "validity"}
	voteColumnsWithoutDefault = []string{"hash", "voting_on", "block_hash", "receive_time", "block_receive_time", "targeted_block_time", "validator_id", "validity"}
	voteColumnsWithDefault    = []string{}
	votePrimaryKeyColumns     = []string{"hash"}
)

type (
	// VoteSlice is an alias for a slice of pointers to Vote.
	// This should generally be used opposed to []Vote.
	VoteSlice []*Vote

	voteQuery struct {
		*queries.Query
	}
)

// Cache for insert, update and upsert
var (
	voteType                 = reflect.TypeOf(&Vote{})
	voteMapping              = queries.MakeStructMapping(voteType)
	votePrimaryKeyMapping, _ = queries.BindMapping(voteType, voteMapping, votePrimaryKeyColumns)
	voteInsertCacheMut       sync.RWMutex
	voteInsertCache          = make(map[string]insertCache)
	voteUpdateCacheMut       sync.RWMutex
	voteUpdateCache          = make(map[string]updateCache)
	voteUpsertCacheMut       sync.RWMutex
	voteUpsertCache          = make(map[string]insertCache)
)

var (
	// Force time package dependency for automated UpdatedAt/CreatedAt.
	_ = time.Second
	// Force qmhelper dependency for where clause generation (which doesn't
	// always happen)
	_ = qmhelper.Where
)

// One returns a single vote record from the query.
func (q voteQuery) One(ctx context.Context, exec boil.ContextExecutor) (*Vote, error) {
	o := &Vote{}

	queries.SetLimit(q.Query, 1)

	err := q.Bind(ctx, exec, o)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, errors.Wrap(err, "models: failed to execute a one query for vote")
	}

	return o, nil
}

// All returns all Vote records from the query.
func (q voteQuery) All(ctx context.Context, exec boil.ContextExecutor) (VoteSlice, error) {
	var o []*Vote

	err := q.Bind(ctx, exec, &o)
	if err != nil {
		return nil, errors.Wrap(err, "models: failed to assign all query results to Vote slice")
	}

	return o, nil
}

// Count returns the count of all Vote records in the query.
func (q voteQuery) Count(ctx context.Context, exec boil.ContextExecutor) (int64, error) {
	var count int64

	queries.SetSelect(q.Query, nil)
	queries.SetCount(q.Query)

	err := q.Query.QueryRowContext(ctx, exec).Scan(&count)
	if err != nil {
		return 0, errors.Wrap(err, "models: failed to count vote rows")
	}

	return count, nil
}

// Exists checks if the row exists in the table.
func (q voteQuery) Exists(ctx context.Context, exec boil.ContextExecutor) (bool, error) {
	var count int64

	queries.SetSelect(q.Query, nil)
	queries.SetCount(q.Query)
	queries.SetLimit(q.Query, 1)

	err := q.Query.QueryRowContext(ctx, exec).Scan(&count)
	if err != nil {
		return false, errors.Wrap(err, "models: failed to check if vote exists")
	}

	return count > 0, nil
}

// Votes retrieves all the records using an executor.
func Votes(mods ...qm.QueryMod) voteQuery {
	mods = append(mods, qm.From("\"vote\""))
	return voteQuery{NewQuery(mods...)}
}

// FindVote retrieves a single record by ID with an executor.
// If selectCols is empty Find will return all columns.
func FindVote(ctx context.Context, exec boil.ContextExecutor, hash string, selectCols ...string) (*Vote, error) {
	voteObj := &Vote{}

	sel := "*"
	if len(selectCols) > 0 {
		sel = strings.Join(strmangle.IdentQuoteSlice(dialect.LQ, dialect.RQ, selectCols), ",")
	}
	query := fmt.Sprintf(
		"select %s from \"vote\" where \"hash\"=$1", sel,
	)

	q := queries.Raw(query, hash)

	err := q.Bind(ctx, exec, voteObj)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, errors.Wrap(err, "models: unable to select from vote")
	}

	return voteObj, nil
}

// Insert a single record using an executor.
// See boil.Columns.InsertColumnSet documentation to understand column list inference for inserts.
func (o *Vote) Insert(ctx context.Context, exec boil.ContextExecutor, columns boil.Columns) error {
	if o == nil {
		return errors.New("models: no vote provided for insertion")
	}

	var err error

	nzDefaults := queries.NonZeroDefaultSet(voteColumnsWithDefault, o)

	key := makeCacheKey(columns, nzDefaults)
	voteInsertCacheMut.RLock()
	cache, cached := voteInsertCache[key]
	voteInsertCacheMut.RUnlock()

	if !cached {
		wl, returnColumns := columns.InsertColumnSet(
			voteAllColumns,
			voteColumnsWithDefault,
			voteColumnsWithoutDefault,
			nzDefaults,
		)

		cache.valueMapping, err = queries.BindMapping(voteType, voteMapping, wl)
		if err != nil {
			return err
		}
		cache.retMapping, err = queries.BindMapping(voteType, voteMapping, returnColumns)
		if err != nil {
			return err
		}
		if len(wl) != 0 {
			cache.query = fmt.Sprintf("INSERT INTO \"vote\" (\"%s\") %%sVALUES (%s)%%s", strings.Join(wl, "\",\""), strmangle.Placeholders(dialect.UseIndexPlaceholders, len(wl), 1, 1))
		} else {
			cache.query = "INSERT INTO \"vote\" %sDEFAULT VALUES%s"
		}

		var queryOutput, queryReturning string

		if len(cache.retMapping) != 0 {
			queryReturning = fmt.Sprintf(" RETURNING \"%s\"", strings.Join(returnColumns, "\",\""))
		}

		cache.query = fmt.Sprintf(cache.query, queryOutput, queryReturning)
	}

	value := reflect.Indirect(reflect.ValueOf(o))
	vals := queries.ValuesFromMapping(value, cache.valueMapping)

	if boil.IsDebug(ctx) {
		writer := boil.DebugWriterFrom(ctx)
		fmt.Fprintln(writer, cache.query)
		fmt.Fprintln(writer, vals)
	}

	if len(cache.retMapping) != 0 {
		err = exec.QueryRowContext(ctx, cache.query, vals...).Scan(queries.PtrsFromMapping(value, cache.retMapping)...)
	} else {
		_, err = exec.ExecContext(ctx, cache.query, vals...)
	}

	if err != nil {
		return errors.Wrap(err, "models: unable to insert into vote")
	}

	if !cached {
		voteInsertCacheMut.Lock()
		voteInsertCache[key] = cache
		voteInsertCacheMut.Unlock()
	}

	return nil
}

// Update uses an executor to update the Vote.
// See boil.Columns.UpdateColumnSet documentation to understand column list inference for updates.
// Update does not automatically update the record in case of default values. Use .Reload() to refresh the records.
func (o *Vote) Update(ctx context.Context, exec boil.ContextExecutor, columns boil.Columns) (int64, error) {
	var err error
	key := makeCacheKey(columns, nil)
	voteUpdateCacheMut.RLock()
	cache, cached := voteUpdateCache[key]
	voteUpdateCacheMut.RUnlock()

	if !cached {
		wl := columns.UpdateColumnSet(
			voteAllColumns,
			votePrimaryKeyColumns,
		)

		if len(wl) == 0 {
			return 0, errors.New("models: unable to update vote, could not build whitelist")
		}

		cache.query = fmt.Sprintf("UPDATE \"vote\" SET %s WHERE %s",
			strmangle.SetParamNames("\"", "\"", 1, wl),
			strmangle.WhereClause("\"", "\"", len(wl)+1, votePrimaryKeyColumns),
		)
		cache.valueMapping, err = queries.BindMapping(voteType, voteMapping, append(wl, votePrimaryKeyColumns...))
		if err != nil {
			return 0, err
		}
	}

	values := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(o)), cache.valueMapping)

	if boil.IsDebug(ctx) {
		writer := boil.DebugWriterFrom(ctx)
		fmt.Fprintln(writer, cache.query)
		fmt.Fprintln(writer, values)
	}
	var result sql.Result
	result, err = exec.ExecContext(ctx, cache.query, values...)
	if err != nil {
		return 0, errors.Wrap(err, "models: unable to update vote row")
	}

	rowsAff, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "models: failed to get rows affected by update for vote")
	}

	if !cached {
		voteUpdateCacheMut.Lock()
		voteUpdateCache[key] = cache
		voteUpdateCacheMut.Unlock()
	}

	return rowsAff, nil
}

// UpdateAll updates all rows with the specified column values.
func (q voteQuery) UpdateAll(ctx context.Context, exec boil.ContextExecutor, cols M) (int64, error) {
	queries.SetUpdate(q.Query, cols)

	result, err := q.Query.ExecContext(ctx, exec)
	if err != nil {
		return 0, errors.Wrap(err, "models: unable to update all for vote")
	}

	rowsAff, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "models: unable to retrieve rows affected for vote")
	}

	return rowsAff, nil
}

// UpdateAll updates all rows with the specified column values, using an executor.
func (o VoteSlice) UpdateAll(ctx context.Context, exec boil.ContextExecutor, cols M) (int64, error) {
	ln := int64(len(o))
	if ln == 0 {
		return 0, nil
	}

	if len(cols) == 0 {
		return 0, errors.New("models: update all requires at least one column argument")
	}

	colNames := make([]string, len(cols))
	args := make([]interface{}, len(cols))

	i := 0
	for name, value := range cols {
		colNames[i] = name
		args[i] = value
		i++
	}

	// Append all of the primary key values for each column
	for _, obj := range o {
		pkeyArgs := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(obj)), votePrimaryKeyMapping)
		args = append(args, pkeyArgs...)
	}

	sql := fmt.Sprintf("UPDATE \"vote\" SET %s WHERE %s",
		strmangle.SetParamNames("\"", "\"", 1, colNames),
		strmangle.WhereClauseRepeated(string(dialect.LQ), string(dialect.RQ), len(colNames)+1, votePrimaryKeyColumns, len(o)))

	if boil.IsDebug(ctx) {
		writer := boil.DebugWriterFrom(ctx)
		fmt.Fprintln(writer, sql)
		fmt.Fprintln(writer, args...)
	}
	result, err := exec.ExecContext(ctx, sql, args...)
	if err != nil {
		return 0, errors.Wrap(err, "models: unable to update all in vote slice")
	}

	rowsAff, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "models: unable to retrieve rows affected all in update all vote")
	}
	return rowsAff, nil
}

// Upsert attempts an insert using an executor, and does an update or ignore on conflict.
// See boil.Columns documentation for how to properly use updateColumns and insertColumns.
func (o *Vote) Upsert(ctx context.Context, exec boil.ContextExecutor, updateOnConflict bool, conflictColumns []string, updateColumns, insertColumns boil.Columns) error {
	if o == nil {
		return errors.New("models: no vote provided for upsert")
	}

	nzDefaults := queries.NonZeroDefaultSet(voteColumnsWithDefault, o)

	// Build cache key in-line uglily - mysql vs psql problems
	buf := strmangle.GetBuffer()
	if updateOnConflict {
		buf.WriteByte('t')
	} else {
		buf.WriteByte('f')
	}
	buf.WriteByte('.')
	for _, c := range conflictColumns {
		buf.WriteString(c)
	}
	buf.WriteByte('.')
	buf.WriteString(strconv.Itoa(updateColumns.Kind))
	for _, c := range updateColumns.Cols {
		buf.WriteString(c)
	}
	buf.WriteByte('.')
	buf.WriteString(strconv.Itoa(insertColumns.Kind))
	for _, c := range insertColumns.Cols {
		buf.WriteString(c)
	}
	buf.WriteByte('.')
	for _, c := range nzDefaults {
		buf.WriteString(c)
	}
	key := buf.String()
	strmangle.PutBuffer(buf)

	voteUpsertCacheMut.RLock()
	cache, cached := voteUpsertCache[key]
	voteUpsertCacheMut.RUnlock()

	var err error

	if !cached {
		insert, ret := insertColumns.InsertColumnSet(
			voteAllColumns,
			voteColumnsWithDefault,
			voteColumnsWithoutDefault,
			nzDefaults,
		)
		update := updateColumns.UpdateColumnSet(
			voteAllColumns,
			votePrimaryKeyColumns,
		)

		if updateOnConflict && len(update) == 0 {
			return errors.New("models: unable to upsert vote, could not build update column list")
		}

		conflict := conflictColumns
		if len(conflict) == 0 {
			conflict = make([]string, len(votePrimaryKeyColumns))
			copy(conflict, votePrimaryKeyColumns)
		}
		cache.query = buildUpsertQueryPostgres(dialect, "\"vote\"", updateOnConflict, ret, update, conflict, insert)

		cache.valueMapping, err = queries.BindMapping(voteType, voteMapping, insert)
		if err != nil {
			return err
		}
		if len(ret) != 0 {
			cache.retMapping, err = queries.BindMapping(voteType, voteMapping, ret)
			if err != nil {
				return err
			}
		}
	}

	value := reflect.Indirect(reflect.ValueOf(o))
	vals := queries.ValuesFromMapping(value, cache.valueMapping)
	var returns []interface{}
	if len(cache.retMapping) != 0 {
		returns = queries.PtrsFromMapping(value, cache.retMapping)
	}

	if boil.IsDebug(ctx) {
		writer := boil.DebugWriterFrom(ctx)
		fmt.Fprintln(writer, cache.query)
		fmt.Fprintln(writer, vals)
	}
	if len(cache.retMapping) != 0 {
		err = exec.QueryRowContext(ctx, cache.query, vals...).Scan(returns...)
		if err == sql.ErrNoRows {
			err = nil // Postgres doesn't return anything when there's no update
		}
	} else {
		_, err = exec.ExecContext(ctx, cache.query, vals...)
	}
	if err != nil {
		return errors.Wrap(err, "models: unable to upsert vote")
	}

	if !cached {
		voteUpsertCacheMut.Lock()
		voteUpsertCache[key] = cache
		voteUpsertCacheMut.Unlock()
	}

	return nil
}

// Delete deletes a single Vote record with an executor.
// Delete will match against the primary key column to find the record to delete.
func (o *Vote) Delete(ctx context.Context, exec boil.ContextExecutor) (int64, error) {
	if o == nil {
		return 0, errors.New("models: no Vote provided for delete")
	}

	args := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(o)), votePrimaryKeyMapping)
	sql := "DELETE FROM \"vote\" WHERE \"hash\"=$1"

	if boil.IsDebug(ctx) {
		writer := boil.DebugWriterFrom(ctx)
		fmt.Fprintln(writer, sql)
		fmt.Fprintln(writer, args...)
	}
	result, err := exec.ExecContext(ctx, sql, args...)
	if err != nil {
		return 0, errors.Wrap(err, "models: unable to delete from vote")
	}

	rowsAff, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "models: failed to get rows affected by delete for vote")
	}

	return rowsAff, nil
}

// DeleteAll deletes all matching rows.
func (q voteQuery) DeleteAll(ctx context.Context, exec boil.ContextExecutor) (int64, error) {
	if q.Query == nil {
		return 0, errors.New("models: no voteQuery provided for delete all")
	}

	queries.SetDelete(q.Query)

	result, err := q.Query.ExecContext(ctx, exec)
	if err != nil {
		return 0, errors.Wrap(err, "models: unable to delete all from vote")
	}

	rowsAff, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "models: failed to get rows affected by deleteall for vote")
	}

	return rowsAff, nil
}

// DeleteAll deletes all rows in the slice, using an executor.
func (o VoteSlice) DeleteAll(ctx context.Context, exec boil.ContextExecutor) (int64, error) {
	if len(o) == 0 {
		return 0, nil
	}

	var args []interface{}
	for _, obj := range o {
		pkeyArgs := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(obj)), votePrimaryKeyMapping)
		args = append(args, pkeyArgs...)
	}

	sql := "DELETE FROM \"vote\" WHERE " +
		strmangle.WhereClauseRepeated(string(dialect.LQ), string(dialect.RQ), 1, votePrimaryKeyColumns, len(o))

	if boil.IsDebug(ctx) {
		writer := boil.DebugWriterFrom(ctx)
		fmt.Fprintln(writer, sql)
		fmt.Fprintln(writer, args)
	}
	result, err := exec.ExecContext(ctx, sql, args...)
	if err != nil {
		return 0, errors.Wrap(err, "models: unable to delete all from vote slice")
	}

	rowsAff, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "models: failed to get rows affected by deleteall for vote")
	}

	return rowsAff, nil
}

// Reload refetches the object from the database
// using the primary keys with an executor.
func (o *Vote) Reload(ctx context.Context, exec boil.ContextExecutor) error {
	ret, err := FindVote(ctx, exec, o.Hash)
	if err != nil {
		return err
	}

	*o = *ret
	return nil
}

// ReloadAll refetches every row with matching primary key column values
// and overwrites the original object slice with the newly updated slice.
func (o *VoteSlice) ReloadAll(ctx context.Context, exec boil.ContextExecutor) error {
	if o == nil || len(*o) == 0 {
		return nil
	}

	slice := VoteSlice{}
	var args []interface{}
	for _, obj := range *o {
		pkeyArgs := queries.ValuesFromMapping(reflect.Indirect(reflect.ValueOf(obj)), votePrimaryKeyMapping)
		args = append(args, pkeyArgs...)
	}

	sql := "SELECT \"vote\".* FROM \"vote\" WHERE " +
		strmangle.WhereClauseRepeated(string(dialect.LQ), string(dialect.RQ), 1, votePrimaryKeyColumns, len(*o))

	q := queries.Raw(sql, args...)

	err := q.Bind(ctx, exec, &slice)
	if err != nil {
		return errors.Wrap(err, "models: unable to reload all in VoteSlice")
	}

	*o = slice

	return nil
}

// VoteExists checks if the Vote row exists.
func VoteExists(ctx context.Context, exec boil.ContextExecutor, hash string) (bool, error) {
	var exists bool
	sql := "select exists(select 1 from \"vote\" where \"hash\"=$1 limit 1)"

	if boil.IsDebug(ctx) {
		writer := boil.DebugWriterFrom(ctx)
		fmt.Fprintln(writer, sql)
		fmt.Fprintln(writer, hash)
	}
	row := exec.QueryRowContext(ctx, sql, hash)

	err := row.Scan(&exists)
	if err != nil {
		return false, errors.Wrap(err, "models: unable to check if vote exists")
	}

	return exists, nil
}
