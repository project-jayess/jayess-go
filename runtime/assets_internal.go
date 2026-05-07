package runtime

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

var ErrAssetNotFound = errors.New("asset not found")

type AssetManifest struct {
	Assets []AssetEntry `json:"assets"`
}

type AssetEntry struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	ContentType string `json:"contentType,omitempty"`
}

func AssetManifestFromJSON(data []byte) (AssetManifest, error) {
	var manifest AssetManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return AssetManifest{}, err
	}
	return manifest, nil
}

func AssetLookup(manifest AssetManifest, name string) (AssetEntry, bool) {
	for _, asset := range manifest.Assets {
		if asset.Name == name {
			return asset, true
		}
	}
	return AssetEntry{}, false
}

func AssetLoad(root string, manifest AssetManifest, name string) ([]byte, error) {
	asset, ok := AssetLookup(manifest, name)
	if !ok {
		return nil, ErrAssetNotFound
	}
	return os.ReadFile(filepath.Join(root, filepath.Clean(asset.Path)))
}
