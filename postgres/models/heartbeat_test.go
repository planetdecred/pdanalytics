// Code generated by SQLBoiler 4.5.0 (https://github.com/volatiletech/sqlboiler). DO NOT EDIT.
// This file is meant to be re-generated in place and/or deleted at any time.

package models

import (
	"bytes"
	"context"
	"reflect"
	"testing"

	"github.com/volatiletech/randomize"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries"
	"github.com/volatiletech/strmangle"
)

var (
	// Relationships sometimes use the reflection helper queries.Equal/queries.Assign
	// so force a package dependency in case they don't.
	_ = queries.Equal
)

func testHeartbeats(t *testing.T) {
	t.Parallel()

	query := Heartbeats()

	if query.Query == nil {
		t.Error("expected a query, got nothing")
	}
}

func testHeartbeatsDelete(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	o := &Heartbeat{}
	if err = randomize.Struct(seed, o, heartbeatDBTypes, true, heartbeatColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize Heartbeat struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	if rowsAff, err := o.Delete(ctx, tx); err != nil {
		t.Error(err)
	} else if rowsAff != 1 {
		t.Error("should only have deleted one row, but affected:", rowsAff)
	}

	count, err := Heartbeats().Count(ctx, tx)
	if err != nil {
		t.Error(err)
	}

	if count != 0 {
		t.Error("want zero records, got:", count)
	}
}

func testHeartbeatsQueryDeleteAll(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	o := &Heartbeat{}
	if err = randomize.Struct(seed, o, heartbeatDBTypes, true, heartbeatColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize Heartbeat struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	if rowsAff, err := Heartbeats().DeleteAll(ctx, tx); err != nil {
		t.Error(err)
	} else if rowsAff != 1 {
		t.Error("should only have deleted one row, but affected:", rowsAff)
	}

	count, err := Heartbeats().Count(ctx, tx)
	if err != nil {
		t.Error(err)
	}

	if count != 0 {
		t.Error("want zero records, got:", count)
	}
}

func testHeartbeatsSliceDeleteAll(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	o := &Heartbeat{}
	if err = randomize.Struct(seed, o, heartbeatDBTypes, true, heartbeatColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize Heartbeat struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	slice := HeartbeatSlice{o}

	if rowsAff, err := slice.DeleteAll(ctx, tx); err != nil {
		t.Error(err)
	} else if rowsAff != 1 {
		t.Error("should only have deleted one row, but affected:", rowsAff)
	}

	count, err := Heartbeats().Count(ctx, tx)
	if err != nil {
		t.Error(err)
	}

	if count != 0 {
		t.Error("want zero records, got:", count)
	}
}

func testHeartbeatsExists(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	o := &Heartbeat{}
	if err = randomize.Struct(seed, o, heartbeatDBTypes, true, heartbeatColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize Heartbeat struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	e, err := HeartbeatExists(ctx, tx, o.Timestamp, o.NodeID)
	if err != nil {
		t.Errorf("Unable to check if Heartbeat exists: %s", err)
	}
	if !e {
		t.Errorf("Expected HeartbeatExists to return true, but got false.")
	}
}

func testHeartbeatsFind(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	o := &Heartbeat{}
	if err = randomize.Struct(seed, o, heartbeatDBTypes, true, heartbeatColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize Heartbeat struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	heartbeatFound, err := FindHeartbeat(ctx, tx, o.Timestamp, o.NodeID)
	if err != nil {
		t.Error(err)
	}

	if heartbeatFound == nil {
		t.Error("want a record, got nil")
	}
}

func testHeartbeatsBind(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	o := &Heartbeat{}
	if err = randomize.Struct(seed, o, heartbeatDBTypes, true, heartbeatColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize Heartbeat struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	if err = Heartbeats().Bind(ctx, tx, o); err != nil {
		t.Error(err)
	}
}

func testHeartbeatsOne(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	o := &Heartbeat{}
	if err = randomize.Struct(seed, o, heartbeatDBTypes, true, heartbeatColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize Heartbeat struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	if x, err := Heartbeats().One(ctx, tx); err != nil {
		t.Error(err)
	} else if x == nil {
		t.Error("expected to get a non nil record")
	}
}

func testHeartbeatsAll(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	heartbeatOne := &Heartbeat{}
	heartbeatTwo := &Heartbeat{}
	if err = randomize.Struct(seed, heartbeatOne, heartbeatDBTypes, false, heartbeatColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize Heartbeat struct: %s", err)
	}
	if err = randomize.Struct(seed, heartbeatTwo, heartbeatDBTypes, false, heartbeatColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize Heartbeat struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = heartbeatOne.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}
	if err = heartbeatTwo.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	slice, err := Heartbeats().All(ctx, tx)
	if err != nil {
		t.Error(err)
	}

	if len(slice) != 2 {
		t.Error("want 2 records, got:", len(slice))
	}
}

func testHeartbeatsCount(t *testing.T) {
	t.Parallel()

	var err error
	seed := randomize.NewSeed()
	heartbeatOne := &Heartbeat{}
	heartbeatTwo := &Heartbeat{}
	if err = randomize.Struct(seed, heartbeatOne, heartbeatDBTypes, false, heartbeatColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize Heartbeat struct: %s", err)
	}
	if err = randomize.Struct(seed, heartbeatTwo, heartbeatDBTypes, false, heartbeatColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize Heartbeat struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = heartbeatOne.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}
	if err = heartbeatTwo.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	count, err := Heartbeats().Count(ctx, tx)
	if err != nil {
		t.Error(err)
	}

	if count != 2 {
		t.Error("want 2 records, got:", count)
	}
}

func testHeartbeatsInsert(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	o := &Heartbeat{}
	if err = randomize.Struct(seed, o, heartbeatDBTypes, true, heartbeatColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize Heartbeat struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	count, err := Heartbeats().Count(ctx, tx)
	if err != nil {
		t.Error(err)
	}

	if count != 1 {
		t.Error("want one record, got:", count)
	}
}

func testHeartbeatsInsertWhitelist(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	o := &Heartbeat{}
	if err = randomize.Struct(seed, o, heartbeatDBTypes, true); err != nil {
		t.Errorf("Unable to randomize Heartbeat struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Whitelist(heartbeatColumnsWithoutDefault...)); err != nil {
		t.Error(err)
	}

	count, err := Heartbeats().Count(ctx, tx)
	if err != nil {
		t.Error(err)
	}

	if count != 1 {
		t.Error("want one record, got:", count)
	}
}

func testHeartbeatToOneNodeUsingNode(t *testing.T) {
	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()

	var local Heartbeat
	var foreign Node

	seed := randomize.NewSeed()
	if err := randomize.Struct(seed, &local, heartbeatDBTypes, false, heartbeatColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize Heartbeat struct: %s", err)
	}
	if err := randomize.Struct(seed, &foreign, nodeDBTypes, false, nodeColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize Node struct: %s", err)
	}

	if err := foreign.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Fatal(err)
	}

	local.NodeID = foreign.Address
	if err := local.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Fatal(err)
	}

	check, err := local.Node().One(ctx, tx)
	if err != nil {
		t.Fatal(err)
	}

	if check.Address != foreign.Address {
		t.Errorf("want: %v, got %v", foreign.Address, check.Address)
	}

	slice := HeartbeatSlice{&local}
	if err = local.L.LoadNode(ctx, tx, false, (*[]*Heartbeat)(&slice), nil); err != nil {
		t.Fatal(err)
	}
	if local.R.Node == nil {
		t.Error("struct should have been eager loaded")
	}

	local.R.Node = nil
	if err = local.L.LoadNode(ctx, tx, true, &local, nil); err != nil {
		t.Fatal(err)
	}
	if local.R.Node == nil {
		t.Error("struct should have been eager loaded")
	}
}

func testHeartbeatToOneSetOpNodeUsingNode(t *testing.T) {
	var err error

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()

	var a Heartbeat
	var b, c Node

	seed := randomize.NewSeed()
	if err = randomize.Struct(seed, &a, heartbeatDBTypes, false, strmangle.SetComplement(heartbeatPrimaryKeyColumns, heartbeatColumnsWithoutDefault)...); err != nil {
		t.Fatal(err)
	}
	if err = randomize.Struct(seed, &b, nodeDBTypes, false, strmangle.SetComplement(nodePrimaryKeyColumns, nodeColumnsWithoutDefault)...); err != nil {
		t.Fatal(err)
	}
	if err = randomize.Struct(seed, &c, nodeDBTypes, false, strmangle.SetComplement(nodePrimaryKeyColumns, nodeColumnsWithoutDefault)...); err != nil {
		t.Fatal(err)
	}

	if err := a.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Fatal(err)
	}
	if err = b.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Fatal(err)
	}

	for i, x := range []*Node{&b, &c} {
		err = a.SetNode(ctx, tx, i != 0, x)
		if err != nil {
			t.Fatal(err)
		}

		if a.R.Node != x {
			t.Error("relationship struct not set to correct value")
		}

		if x.R.Heartbeats[0] != &a {
			t.Error("failed to append to foreign relationship struct")
		}
		if a.NodeID != x.Address {
			t.Error("foreign key was wrong value", a.NodeID)
		}

		if exists, err := HeartbeatExists(ctx, tx, a.Timestamp, a.NodeID); err != nil {
			t.Fatal(err)
		} else if !exists {
			t.Error("want 'a' to exist")
		}

	}
}

func testHeartbeatsReload(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	o := &Heartbeat{}
	if err = randomize.Struct(seed, o, heartbeatDBTypes, true, heartbeatColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize Heartbeat struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	if err = o.Reload(ctx, tx); err != nil {
		t.Error(err)
	}
}

func testHeartbeatsReloadAll(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	o := &Heartbeat{}
	if err = randomize.Struct(seed, o, heartbeatDBTypes, true, heartbeatColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize Heartbeat struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	slice := HeartbeatSlice{o}

	if err = slice.ReloadAll(ctx, tx); err != nil {
		t.Error(err)
	}
}

func testHeartbeatsSelect(t *testing.T) {
	t.Parallel()

	seed := randomize.NewSeed()
	var err error
	o := &Heartbeat{}
	if err = randomize.Struct(seed, o, heartbeatDBTypes, true, heartbeatColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize Heartbeat struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	slice, err := Heartbeats().All(ctx, tx)
	if err != nil {
		t.Error(err)
	}

	if len(slice) != 1 {
		t.Error("want one record, got:", len(slice))
	}
}

var (
	heartbeatDBTypes = map[string]string{`Timestamp`: `bigint`, `NodeID`: `character varying`, `LastSeen`: `bigint`, `Latency`: `integer`, `CurrentHeight`: `bigint`}
	_                = bytes.MinRead
)

func testHeartbeatsUpdate(t *testing.T) {
	t.Parallel()

	if 0 == len(heartbeatPrimaryKeyColumns) {
		t.Skip("Skipping table with no primary key columns")
	}
	if len(heartbeatAllColumns) == len(heartbeatPrimaryKeyColumns) {
		t.Skip("Skipping table with only primary key columns")
	}

	seed := randomize.NewSeed()
	var err error
	o := &Heartbeat{}
	if err = randomize.Struct(seed, o, heartbeatDBTypes, true, heartbeatColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize Heartbeat struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	count, err := Heartbeats().Count(ctx, tx)
	if err != nil {
		t.Error(err)
	}

	if count != 1 {
		t.Error("want one record, got:", count)
	}

	if err = randomize.Struct(seed, o, heartbeatDBTypes, true, heartbeatPrimaryKeyColumns...); err != nil {
		t.Errorf("Unable to randomize Heartbeat struct: %s", err)
	}

	if rowsAff, err := o.Update(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	} else if rowsAff != 1 {
		t.Error("should only affect one row but affected", rowsAff)
	}
}

func testHeartbeatsSliceUpdateAll(t *testing.T) {
	t.Parallel()

	if len(heartbeatAllColumns) == len(heartbeatPrimaryKeyColumns) {
		t.Skip("Skipping table with only primary key columns")
	}

	seed := randomize.NewSeed()
	var err error
	o := &Heartbeat{}
	if err = randomize.Struct(seed, o, heartbeatDBTypes, true, heartbeatColumnsWithDefault...); err != nil {
		t.Errorf("Unable to randomize Heartbeat struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Insert(ctx, tx, boil.Infer()); err != nil {
		t.Error(err)
	}

	count, err := Heartbeats().Count(ctx, tx)
	if err != nil {
		t.Error(err)
	}

	if count != 1 {
		t.Error("want one record, got:", count)
	}

	if err = randomize.Struct(seed, o, heartbeatDBTypes, true, heartbeatPrimaryKeyColumns...); err != nil {
		t.Errorf("Unable to randomize Heartbeat struct: %s", err)
	}

	// Remove Primary keys and unique columns from what we plan to update
	var fields []string
	if strmangle.StringSliceMatch(heartbeatAllColumns, heartbeatPrimaryKeyColumns) {
		fields = heartbeatAllColumns
	} else {
		fields = strmangle.SetComplement(
			heartbeatAllColumns,
			heartbeatPrimaryKeyColumns,
		)
	}

	value := reflect.Indirect(reflect.ValueOf(o))
	typ := reflect.TypeOf(o).Elem()
	n := typ.NumField()

	updateMap := M{}
	for _, col := range fields {
		for i := 0; i < n; i++ {
			f := typ.Field(i)
			if f.Tag.Get("boil") == col {
				updateMap[col] = value.Field(i).Interface()
			}
		}
	}

	slice := HeartbeatSlice{o}
	if rowsAff, err := slice.UpdateAll(ctx, tx, updateMap); err != nil {
		t.Error(err)
	} else if rowsAff != 1 {
		t.Error("wanted one record updated but got", rowsAff)
	}
}

func testHeartbeatsUpsert(t *testing.T) {
	t.Parallel()

	if len(heartbeatAllColumns) == len(heartbeatPrimaryKeyColumns) {
		t.Skip("Skipping table with only primary key columns")
	}

	seed := randomize.NewSeed()
	var err error
	// Attempt the INSERT side of an UPSERT
	o := Heartbeat{}
	if err = randomize.Struct(seed, &o, heartbeatDBTypes, true); err != nil {
		t.Errorf("Unable to randomize Heartbeat struct: %s", err)
	}

	ctx := context.Background()
	tx := MustTx(boil.BeginTx(ctx, nil))
	defer func() { _ = tx.Rollback() }()
	if err = o.Upsert(ctx, tx, false, nil, boil.Infer(), boil.Infer()); err != nil {
		t.Errorf("Unable to upsert Heartbeat: %s", err)
	}

	count, err := Heartbeats().Count(ctx, tx)
	if err != nil {
		t.Error(err)
	}
	if count != 1 {
		t.Error("want one record, got:", count)
	}

	// Attempt the UPDATE side of an UPSERT
	if err = randomize.Struct(seed, &o, heartbeatDBTypes, false, heartbeatPrimaryKeyColumns...); err != nil {
		t.Errorf("Unable to randomize Heartbeat struct: %s", err)
	}

	if err = o.Upsert(ctx, tx, true, nil, boil.Infer(), boil.Infer()); err != nil {
		t.Errorf("Unable to upsert Heartbeat: %s", err)
	}

	count, err = Heartbeats().Count(ctx, tx)
	if err != nil {
		t.Error(err)
	}
	if count != 1 {
		t.Error("want one record, got:", count)
	}
}
