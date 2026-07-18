package guide

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

const (
	OutputMarker        = ".opendart-spec-output"
	OutputMarkerContent = "opendart-spec-v1\n"
)

var managedOutputs = []string{"paths", "schemas", "components", "openapi.yaml", OutputMarker}

type publishHook func(phase, name string) error

func publishGenerated(staging, output, repositoryRoot string) error {
	return publishGeneratedWithHook(staging, output, repositoryRoot, nil)
}

func publishGeneratedWithHook(staging, output, repositoryRoot string, hook publishHook) error {
	if err := assertSafeOutput(output, repositoryRoot); err != nil {
		return err
	}
	if err := os.MkdirAll(output, 0o755); err != nil {
		return fmt.Errorf("prepare generated output: %w", err)
	}
	if err := assertSafePhysicalOutput(output, repositoryRoot); err != nil {
		return err
	}
	outputInfo, err := os.Lstat(output)
	if err != nil {
		return fmt.Errorf("inspect generated output: %w", err)
	}
	if !outputInfo.IsDir() || outputInfo.Mode()&os.ModeSymlink != 0 {
		return errors.New("refusing to publish through a non-directory or symlink")
	}
	generatedDirectory := filepath.Join(output, "generated")
	if info, err := os.Lstat(generatedDirectory); err == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("refusing to invalidate a bundle through an unsafe path")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect generated bundle directory: %w", err)
	}

	existing, err := existingManagedOutputs(output)
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		marker, markerErr := os.ReadFile(filepath.Join(output, OutputMarker))
		markerInfo, infoErr := os.Lstat(filepath.Join(output, OutputMarker))
		if markerErr != nil || infoErr != nil || !markerInfo.Mode().IsRegular() || markerInfo.Mode()&os.ModeSymlink != 0 || string(marker) != OutputMarkerContent {
			return fmt.Errorf("refusing to replace an unowned output directory containing %v", existing)
		}
	}

	backup, err := os.MkdirTemp(filepath.Dir(output), ".opendart-backup-")
	if err != nil {
		return fmt.Errorf("create publication backup: %w", err)
	}
	movedOld := make([]string, 0, len(managedOutputs))
	movedNew := make([]string, 0, len(managedOutputs))
	cleanupBackup := false
	defer func() {
		if cleanupBackup {
			_ = os.RemoveAll(backup)
		}
	}()

	rollback := func(publishErr error) error {
		var rollbackErrors []error
		for _, name := range slices.Backward(movedNew) {
			if err := os.RemoveAll(filepath.Join(output, name)); err != nil {
				rollbackErrors = append(rollbackErrors, err)
			}
		}
		for _, name := range slices.Backward(movedOld) {
			if err := os.Rename(filepath.Join(backup, name), filepath.Join(output, name)); err != nil {
				rollbackErrors = append(rollbackErrors, err)
			}
		}
		if len(rollbackErrors) > 0 {
			return fmt.Errorf("publishing failed and rollback is incomplete; backup retained at %s: %w", backup, errors.Join(append([]error{publishErr}, rollbackErrors...)...))
		}
		cleanupBackup = true
		return publishErr
	}

	for _, name := range managedOutputs {
		target := filepath.Join(output, name)
		if _, err := os.Lstat(target); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return rollback(fmt.Errorf("inspect managed output %s: %w", name, err))
		}
		if err := os.Rename(target, filepath.Join(backup, name)); err != nil {
			return rollback(fmt.Errorf("back up managed output %s: %w", name, err))
		}
		movedOld = append(movedOld, name)
	}
	for _, name := range managedOutputs {
		source := filepath.Join(staging, name)
		if _, err := os.Lstat(source); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return rollback(fmt.Errorf("inspect staged output %s: %w", name, err))
		}
		if hook != nil {
			if err := hook("before-new", name); err != nil {
				return rollback(err)
			}
		}
		if err := os.Rename(source, filepath.Join(output, name)); err != nil {
			return rollback(fmt.Errorf("publish managed output %s: %w", name, err))
		}
		movedNew = append(movedNew, name)
	}
	if err := os.Remove(filepath.Join(output, "generated", "openapi.bundle.yaml")); err != nil && !errors.Is(err, os.ErrNotExist) {
		return rollback(fmt.Errorf("invalidate generated bundle: %w", err))
	}
	cleanupBackup = true
	return nil
}

func existingManagedOutputs(output string) ([]string, error) {
	names := append(slices.Clone(managedOutputs), filepath.Join("generated", "openapi.bundle.yaml"))
	var existing []string
	for _, name := range names {
		if _, err := os.Lstat(filepath.Join(output, name)); err == nil {
			existing = append(existing, name)
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("inspect managed output %s: %w", name, err)
		}
	}
	return existing, nil
}

func assertSafeOutput(output, repositoryRoot string) error {
	absOutput, err := filepath.Abs(output)
	if err != nil {
		return fmt.Errorf("resolve generated output: %w", err)
	}
	blocked := []string{filepath.VolumeName(absOutput) + string(filepath.Separator), repositoryRoot}
	if home, err := os.UserHomeDir(); err == nil {
		blocked = append(blocked, home)
	}
	blocked = append(blocked, os.TempDir())
	for _, candidate := range blocked {
		absolute, err := filepath.Abs(candidate)
		if err == nil && filepath.Clean(absOutput) == filepath.Clean(absolute) {
			return errors.New("refusing to publish into a broad directory")
		}
	}
	return nil
}

func assertSafePhysicalOutput(output, repositoryRoot string) error {
	physicalOutput, err := filepath.EvalSymlinks(output)
	if err != nil {
		return fmt.Errorf("resolve physical generated output: %w", err)
	}
	for _, candidate := range []string{filepath.VolumeName(physicalOutput) + string(filepath.Separator), repositoryRoot, os.TempDir()} {
		physicalCandidate, err := filepath.EvalSymlinks(candidate)
		if err == nil && filepath.Clean(physicalOutput) == filepath.Clean(physicalCandidate) {
			return errors.New("refusing to publish into a broad physical directory")
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		if physicalHome, err := filepath.EvalSymlinks(home); err == nil && filepath.Clean(physicalOutput) == filepath.Clean(physicalHome) {
			return errors.New("refusing to publish into a broad physical directory")
		}
	}
	return nil
}
