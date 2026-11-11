package commands_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"fearless/internal/app/bus"
	"fearless/internal/app/commands"
	"fearless/internal/core/events"
	"fearless/internal/core/model"
)

func stringRef(s string) *string { return &s }

func TestSelectEnvironment(t *testing.T) {
	dir, b, store, svc := createTestWorkspace(t)
	ctx := context.Background()

	colUID, err := svc.LoadCollection(ctx, commands.LoadCollectionCmd{Path: dir})
	if err != nil {
		t.Fatalf("LoadCollection failed: %v", err)
	}

	// Create environment "Staging"
	stagingEnv, err := svc.NewEnvironment(ctx, commands.NewEnvironmentCmd{
		CollectionUID: colUID,
		Name:          "Staging",
		Variables: []model.Variable{
			{Name: stringRef("host"), Value: "https://staging.api.test", Enabled: true},
		},
	})
	if err != nil {
		t.Fatalf("NewEnvironment failed: %v", err)
	}

	eventCh, cancel := b.Subscribe()
	defer cancel()

	// 1. Select Staging
	err = svc.SelectEnvironment(ctx, commands.SelectEnvironmentCmd{
		CollectionUID:   colUID,
		EnvironmentName: "Staging",
	})
	if err != nil {
		t.Fatalf("SelectEnvironment failed: %v", err)
	}

	snap := store.Snapshot()
	col := snap.FindCollection(colUID)
	if col.ActiveEnvironment != "Staging" {
		t.Errorf("col.ActiveEnvironment = %q, want Staging", col.ActiveEnvironment)
	}

	// 2. Verify event
	evs := collectEvents(t, eventCh, 50*time.Millisecond)
	var foundModified bool
	for _, ev := range evs {
		if cEv, ok := ev.(events.CollectionChanged); ok {
			for _, m := range cEv.Change.Modified {
				if m == stagingEnv.UID {
					foundModified = true
				}
			}
		}
	}
	if !foundModified {
		t.Errorf("expected CollectionChanged Modified event for selected environment")
	}

	// 3. Idempotency test: selecting already active environment is a clean no-op
	if err := svc.SelectEnvironment(ctx, commands.SelectEnvironmentCmd{CollectionUID: colUID, EnvironmentName: "Staging"}); err != nil {
		t.Errorf("idempotent SelectEnvironment failed: %v", err)
	}

	// 4. Clear active environment ("No Environment")
	if err := svc.SelectEnvironment(ctx, commands.SelectEnvironmentCmd{CollectionUID: colUID, EnvironmentName: ""}); err != nil {
		t.Fatalf("SelectEnvironment with empty name failed: %v", err)
	}
	if store.Snapshot().FindCollection(colUID).ActiveEnvironment != "" {
		t.Errorf("expected ActiveEnvironment to be cleared")
	}

	// 5. Select non-existent environment -> error
	if err := svc.SelectEnvironment(ctx, commands.SelectEnvironmentCmd{CollectionUID: colUID, EnvironmentName: "NonExistent"}); err == nil {
		t.Errorf("expected error when selecting non-existent environment")
	}
}

func TestNewAndSaveEnvironment(t *testing.T) {
	dir, b, store, svc := createTestWorkspace(t)
	ctx := context.Background()

	colUID, err := svc.LoadCollection(ctx, commands.LoadCollectionCmd{Path: dir})
	if err != nil {
		t.Fatalf("LoadCollection failed: %v", err)
	}

	eventCh, cancel := b.Subscribe()
	defer cancel()

	// 1. NewEnvironment
	env, err := svc.NewEnvironment(ctx, commands.NewEnvironmentCmd{
		CollectionUID: colUID,
		Name:          "Production",
		Color:         "#ef4444",
		Variables: []model.Variable{
			{Name: stringRef("apiKey"), Value: "prod-secret-key", Enabled: true, Secret: true},
			{Name: stringRef("retries"), Value: 3, Enabled: true},
		},
	})
	if err != nil {
		t.Fatalf("NewEnvironment failed: %v", err)
	}
	if env.Name != "Production" {
		t.Errorf("env.Name = %q, want Production", env.Name)
	}

	// Check file on disk
	envPath := filepath.Join(dir, "environments", "Production.bru")
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		t.Errorf("environment file was not created on disk: %s", envPath)
	}

	// Check event
	evs := collectEvents(t, eventCh, 50*time.Millisecond)
	var foundAdded bool
	for _, ev := range evs {
		if cEv, ok := ev.(events.CollectionChanged); ok {
			for _, a := range cEv.Change.Added {
				if a == env.UID {
					foundAdded = true
				}
			}
		}
	}
	if !foundAdded {
		t.Errorf("expected CollectionChanged Added event")
	}

	// 2. Modify and SaveEnvironment
	env.Variables = append(env.Variables, model.Variable{
		Name:    stringRef("timeout"),
		Value:   5000,
		Enabled: true,
	})

	if err := svc.SaveEnvironment(ctx, commands.SaveEnvironmentCmd{
		CollectionUID: colUID,
		Environment:   env,
	}); err != nil {
		t.Fatalf("SaveEnvironment failed: %v", err)
	}

	updatedSnap := store.Snapshot()
	updatedCol := updatedSnap.FindCollection(colUID)
	var foundEnv *model.Environment
	for _, e := range updatedCol.Environments {
		if e.UID == env.UID {
			foundEnv = e
			break
		}
	}
	if foundEnv == nil || len(foundEnv.Variables) != 3 {
		t.Fatalf("expected updated environment with 3 variables: %+v", foundEnv)
	}

	// Verify disk contents
	content, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("failed to read environment from disk: %v", err)
	}
	if !containsString(string(content), "timeout") {
		t.Errorf("disk environment did not contain updated variable 'timeout'")
	}
}

func TestDeleteEnvironment(t *testing.T) {
	dir, b, store, svc := createTestWorkspace(t)
	ctx := context.Background()

	colUID, err := svc.LoadCollection(ctx, commands.LoadCollectionCmd{Path: dir})
	if err != nil {
		t.Fatalf("LoadCollection failed: %v", err)
	}

	env, err := svc.NewEnvironment(ctx, commands.NewEnvironmentCmd{
		CollectionUID: colUID,
		Name:          "To Delete",
		Variables: []model.Variable{
			{Name: stringRef("key"), Value: "val", Enabled: true},
		},
	})
	if err != nil {
		t.Fatalf("NewEnvironment failed: %v", err)
	}

	// Select it as active environment
	if err := svc.SelectEnvironment(ctx, commands.SelectEnvironmentCmd{CollectionUID: colUID, EnvironmentName: env.Name}); err != nil {
		t.Fatalf("SelectEnvironment failed: %v", err)
	}

	eventCh, cancel := b.Subscribe()
	defer cancel()

	// Delete by UID
	err = svc.DeleteEnvironment(ctx, commands.DeleteEnvironmentCmd{
		CollectionUID:  colUID,
		EnvironmentUID: env.UID,
	})
	if err != nil {
		t.Fatalf("DeleteEnvironment failed: %v", err)
	}

	// 1. Assert file removed from disk
	envPath := filepath.Join(dir, "environments", env.Filename)
	if _, err := os.Stat(envPath); !os.IsNotExist(err) {
		t.Errorf("file still exists on disk after DeleteEnvironment: %s", envPath)
	}

	// 2. Assert removed from state
	updatedCol := store.Snapshot().FindCollection(colUID)
	for _, e := range updatedCol.Environments {
		if e.UID == env.UID {
			t.Errorf("environment still found in collection after deletion")
		}
	}

	// 3. Assert ActiveEnvironment unselected
	if updatedCol.ActiveEnvironment != "" {
		t.Errorf("ActiveEnvironment = %q, want empty after active env deletion", updatedCol.ActiveEnvironment)
	}

	// 4. Assert event
	evs := collectEvents(t, eventCh, 50*time.Millisecond)
	var foundRemoved bool
	for _, ev := range evs {
		if cEv, ok := ev.(events.CollectionChanged); ok {
			for _, r := range cEv.Change.Removed {
				if r == env.UID {
					foundRemoved = true
				}
			}
		}
	}
	if !foundRemoved {
		t.Errorf("expected CollectionChanged Removed event")
	}

	// 5. Idempotency: delete already removed environment returns nil
	if err := svc.DeleteEnvironment(ctx, commands.DeleteEnvironmentCmd{CollectionUID: colUID, EnvironmentUID: env.UID}); err != nil {
		t.Errorf("second DeleteEnvironment failed: %v", err)
	}
}

func TestEnvironment_YAML(t *testing.T) {
	dir := t.TempDir()
	openCollectionYML := `opencollection: "1.0.0"
info:
  name: yaml-env-test
  version: "1.0.0"
`
	if err := os.WriteFile(filepath.Join(dir, "opencollection.yml"), []byte(openCollectionYML), 0o644); err != nil {
		t.Fatalf("failed to write opencollection.yml: %v", err)
	}

	b := bus.New()
	store := bus.NewStore(nil, b)
	svc := commands.NewCollectionService(store, b)
	defer func() {
		_ = svc.Close()
		store.Close()
		b.Close()
	}()

	ctx := context.Background()
	colUID, err := svc.LoadCollection(ctx, commands.LoadCollectionCmd{Path: dir})
	if err != nil {
		t.Fatalf("LoadCollection YAML failed: %v", err)
	}

	// Create environment in YAML collection
	env, err := svc.NewEnvironment(ctx, commands.NewEnvironmentCmd{
		CollectionUID: colUID,
		Name:          "Dev",
		Format:        model.FormatYML,
		Variables: []model.Variable{
			{Name: stringRef("apiHost"), Value: "https://dev.api.test", Enabled: true},
		},
	})
	if err != nil {
		t.Fatalf("NewEnvironment YAML failed: %v", err)
	}
	if filepath.Ext(env.Filename) != ".yml" {
		t.Errorf("env.Filename = %q, want .yml extension", env.Filename)
	}

	// Select it
	if err := svc.SelectEnvironment(ctx, commands.SelectEnvironmentCmd{CollectionUID: colUID, EnvironmentName: "Dev"}); err != nil {
		t.Fatalf("SelectEnvironment in YAML failed: %v", err)
	}
	if store.Snapshot().FindCollection(colUID).ActiveEnvironment != "Dev" {
		t.Errorf("active environment was not set in YAML collection")
	}

	// Delete by Name
	if err := svc.DeleteEnvironment(ctx, commands.DeleteEnvironmentCmd{CollectionUID: colUID, EnvironmentName: "Dev"}); err != nil {
		t.Fatalf("DeleteEnvironment by name failed: %v", err)
	}
}

func TestEnvironmentValidationAndErrors(t *testing.T) {
	dir, _, _, svc := createTestWorkspace(t)
	ctx := context.Background()

	colUID, err := svc.LoadCollection(ctx, commands.LoadCollectionCmd{Path: dir})
	if err != nil {
		t.Fatalf("LoadCollection failed: %v", err)
	}

	// NewEnvironment errors
	if _, err := svc.NewEnvironment(ctx, commands.NewEnvironmentCmd{}); err == nil {
		t.Errorf("expected error on empty NewEnvironmentCmd")
	}
	if _, err := svc.NewEnvironment(ctx, commands.NewEnvironmentCmd{CollectionUID: colUID, Name: ""}); err == nil {
		t.Errorf("expected error on empty Name")
	}
	if _, err := svc.NewEnvironment(ctx, commands.NewEnvironmentCmd{CollectionUID: "bad-col", Name: "Env"}); err == nil {
		t.Errorf("expected error on unknown collection")
	}

	// Duplicate environment name
	_, err = svc.NewEnvironment(ctx, commands.NewEnvironmentCmd{CollectionUID: colUID, Name: "Unique"})
	if err != nil {
		t.Fatalf("failed to create Unique env: %v", err)
	}
	if _, err := svc.NewEnvironment(ctx, commands.NewEnvironmentCmd{CollectionUID: colUID, Name: "Unique"}); err == nil {
		t.Errorf("expected error on duplicate environment name")
	}

	// SaveEnvironment errors
	if err := svc.SaveEnvironment(ctx, commands.SaveEnvironmentCmd{}); err == nil {
		t.Errorf("expected error on empty SaveEnvironmentCmd")
	}
	if err := svc.SaveEnvironment(ctx, commands.SaveEnvironmentCmd{CollectionUID: "bad-col", Environment: &model.Environment{Name: "E"}}); err == nil {
		t.Errorf("expected error on unknown collection")
	}
	if err := svc.SaveEnvironment(ctx, commands.SaveEnvironmentCmd{CollectionUID: colUID, Environment: &model.Environment{Name: ""}}); err == nil {
		t.Errorf("expected error on empty environment name")
	}

	// DeleteEnvironment errors
	if err := svc.DeleteEnvironment(ctx, commands.DeleteEnvironmentCmd{}); err == nil {
		t.Errorf("expected error on empty DeleteEnvironmentCmd")
	}
	if err := svc.DeleteEnvironment(ctx, commands.DeleteEnvironmentCmd{CollectionUID: "bad-col", EnvironmentName: "E"}); err == nil {
		t.Errorf("expected error on unknown collection")
	}
	if err := svc.DeleteEnvironment(ctx, commands.DeleteEnvironmentCmd{CollectionUID: colUID}); err == nil {
		t.Errorf("expected error when neither name nor UID specified")
	}

	// SelectEnvironment errors
	if err := svc.SelectEnvironment(ctx, commands.SelectEnvironmentCmd{}); err == nil {
		t.Errorf("expected error on empty SelectEnvironmentCmd")
	}
	if err := svc.SelectEnvironment(ctx, commands.SelectEnvironmentCmd{CollectionUID: "bad-col", EnvironmentName: "E"}); err == nil {
		t.Errorf("expected error on unknown collection")
	}
}
