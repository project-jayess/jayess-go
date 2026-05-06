package appdist

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const PackageMetadataFile = "jayess.package.json"

type PackageMetadata struct {
	RuntimeAssets      []PackageAsset     `json:"runtimeAssets"`
	HelperAssets       []PackageAsset     `json:"helperAssets"`
	LicenseFiles       []string           `json:"licenseFiles"`
	SystemDependencies []SystemDependency `json:"systemDependencies"`
}

type PackageAsset struct {
	Path            string `json:"path"`
	OutputName      string `json:"outputName"`
	RequiresLicense bool   `json:"requiresLicense"`
	BuildOnly       bool   `json:"buildOnly"`
}

type SystemDependency struct {
	Name      string `json:"name"`
	BuildOnly bool   `json:"buildOnly"`
}

func LoadPackageMetadata(packageRoot string) (PackageMetadata, error) {
	content, err := os.ReadFile(filepath.Join(packageRoot, PackageMetadataFile))
	if err != nil {
		return PackageMetadata{}, err
	}
	var metadata PackageMetadata
	if err := json.Unmarshal(content, &metadata); err != nil {
		return PackageMetadata{}, err
	}
	return metadata, nil
}
