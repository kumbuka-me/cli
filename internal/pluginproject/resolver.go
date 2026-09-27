package pluginproject

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/kumbuka-me/sdk/pluginpackage"
)

var errPackageMismatch = errors.New("plugin package does not match declared dependency")

// Resolved contains validated package bytes for one project dependency.
type Resolved struct {
	// Dependency is the normalized declaration used to resolve the package.
	Dependency Dependency
	// Manifest is the validated manifest read from Archive.
	Manifest pluginpackage.Manifest
	// Archive contains the complete validated .kumbukaplugin package.
	Archive []byte
}

// Resolver loads project dependencies from embedded packages, a user cache, or GitHub Releases.
type Resolver struct {
	// Bundled optionally contains packaged .kumbukaplugin files keyed by asset name.
	Bundled fs.FS
	// CacheDir stores validated downloaded plugin packages.
	CacheDir string
	// Client performs release and checksum downloads.
	Client *http.Client
	// baseURL is the release host and is overridden only by package tests.
	baseURL string
}

// NewResolver constructs the default project resolver.
func NewResolver(bundled fs.FS) (*Resolver, error) {
	cache, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("resolve plugin cache: %w", err)
	}
	return &Resolver{
		Bundled:  bundled,
		CacheDir: filepath.Join(cache, "kumbuka", "plugins"),
		Client:   &http.Client{Timeout: 30 * time.Second},
		baseURL:  "https://github.com",
	}, nil
}

// Resolve validates and materializes every declared dependency without loading its WASM runtime.
func (r *Resolver) Resolve(ctx context.Context, dependencies []Dependency) ([]Resolved, error) {
	result := make([]Resolved, 0, len(dependencies))
	for _, dependency := range dependencies {
		var err error
		dependency, err = NormalizeDependency(dependency)
		if err != nil {
			return nil, err
		}

		item, found, err := resolveBundled(r.Bundled, dependency)
		if err != nil {
			return nil, err
		}
		if found {
			result = append(result, item)
			continue
		}

		item, err = r.resolveRemote(ctx, dependency)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}

	slices.SortFunc(result, func(left, right Resolved) int {
		return strings.Compare(left.Manifest.ID, right.Manifest.ID)
	})
	return result, nil
}

// resolveBundled returns a matching embedded package while surfacing malformed package data.
func resolveBundled(bundled fs.FS, dependency Dependency) (Resolved, bool, error) {
	if bundled == nil {
		return Resolved{}, false, nil
	}

	archive, err := fs.ReadFile(bundled, dependency.Asset+".kumbukaplugin")
	if errors.Is(err, fs.ErrNotExist) {
		return Resolved{}, false, nil
	}
	if err != nil {
		return Resolved{}, false, fmt.Errorf("read bundled plugin %s: %w", dependency.ID, err)
	}

	item, err := resolvedPackage(dependency, archive)
	if errors.Is(err, errPackageMismatch) {
		return Resolved{}, false, nil
	}
	if err != nil {
		return Resolved{}, false, fmt.Errorf("validate bundled plugin %s: %w", dependency.ID, err)
	}
	return item, true, nil
}

// resolveRemote loads a validated cache entry or downloads and caches the pinned release.
func (r *Resolver) resolveRemote(ctx context.Context, dependency Dependency) (Resolved, error) {
	cacheFile := r.cacheFilename(dependency)
	if archive, err := os.ReadFile(cacheFile); err == nil {
		item, validationErr := resolvedPackage(dependency, archive)
		if validationErr == nil {
			return item, nil
		}
		if removeErr := os.Remove(cacheFile); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			return Resolved{}, fmt.Errorf("remove invalid cached plugin %s: %w", dependency.ID, removeErr)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return Resolved{}, fmt.Errorf("read cached plugin %s: %w", dependency.ID, err)
	}

	archive, err := r.download(ctx, dependency)
	if err != nil {
		return Resolved{}, err
	}
	item, err := resolvedPackage(dependency, archive)
	if err != nil {
		return Resolved{}, fmt.Errorf("validate downloaded plugin %s: %w", dependency.ID, err)
	}
	if err := writeCacheFile(cacheFile, archive); err != nil {
		return Resolved{}, fmt.Errorf("cache plugin %s: %w", dependency.ID, err)
	}
	return item, nil
}

// cacheFilename returns the repository-scoped cache path for one pinned dependency.
func (r *Resolver) cacheFilename(dependency Dependency) string {
	cacheKey := sha256.Sum256([]byte(dependency.Repository + "\n" + dependency.TagPrefix + "\n" + dependency.Asset))
	return filepath.Join(
		r.CacheDir,
		hex.EncodeToString(cacheKey[:]),
		dependency.ID,
		dependency.Version,
		"plugin.kumbukaplugin",
	)
}

// resolvedPackage verifies package structure and the declared identity/version.
func resolvedPackage(dependency Dependency, archive []byte) (Resolved, error) {
	pkg, err := pluginpackage.Read(archive)
	if err != nil {
		return Resolved{}, fmt.Errorf("read package: %w", err)
	}

	manifest := pkg.Manifest()
	if manifest.ID != dependency.ID {
		return Resolved{}, fmt.Errorf(
			"%w: package ID %s, declared ID %s",
			errPackageMismatch,
			manifest.ID,
			dependency.ID,
		)
	}
	if manifest.Version != dependency.Version {
		return Resolved{}, fmt.Errorf(
			"%w: plugin %s package version %s, declared version %s",
			errPackageMismatch,
			dependency.ID,
			manifest.Version,
			dependency.Version,
		)
	}

	return Resolved{Dependency: dependency, Manifest: manifest, Archive: slices.Clone(archive)}, nil
}

// download fetches a release package and verifies its published SHA-256 checksum.
func (r *Resolver) download(ctx context.Context, dependency Dependency) ([]byte, error) {
	filename := dependency.Asset + "-" + dependency.Version + ".kumbukaplugin"
	tag := dependency.TagPrefix + dependency.Version
	base := strings.TrimRight(r.baseURL, "/") + "/" + dependency.Repository + "/releases/download/" + tag + "/" + filename

	archive, err := r.get(ctx, base, pluginpackage.MaxArchiveBytes)
	if err != nil {
		return nil, fmt.Errorf("download plugin %s: %w", dependency.ID, err)
	}
	checksum, err := r.get(ctx, base+".sha256", 4096)
	if err != nil {
		return nil, fmt.Errorf("download checksum for %s: %w", dependency.ID, err)
	}

	fields := strings.Fields(string(checksum))
	if len(fields) == 0 {
		return nil, fmt.Errorf("empty checksum for plugin %s", dependency.ID)
	}
	expected, err := hex.DecodeString(fields[0])
	if err != nil || len(expected) != sha256.Size {
		return nil, fmt.Errorf("invalid checksum for plugin %s", dependency.ID)
	}
	actual := sha256.Sum256(archive)
	if !slices.Equal(expected, actual[:]) {
		return nil, fmt.Errorf("checksum mismatch for plugin %s", dependency.ID)
	}
	return archive, nil
}

// get downloads one bounded HTTP response body.
func (r *Resolver) get(ctx context.Context, url string, limit int) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	client := r.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return nil, fmt.Errorf("unexpected HTTP status %s", response.Status)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, int64(limit)+1))
	if err != nil {
		return nil, err
	}
	if len(data) > limit {
		return nil, fmt.Errorf("response exceeds %d-byte size limit", limit)
	}
	return data, nil
}

// writeCacheFile replaces one cache entry through a temporary sibling file.
func writeCacheFile(filename string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(filename), ".plugin-*")
	if err != nil {
		return err
	}
	name := temporary.Name()
	defer func() { _ = os.Remove(name) }()

	if err := temporary.Chmod(0o644); err != nil {
		_ = temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(name, filename)
}
