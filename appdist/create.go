package appdist

import (
	"io"
	"os"
	"path/filepath"
)

type Result struct {
	OutputDir        string
	ExecutablePath   string
	CopiedAssetPaths []string
	Diagnostics      []string
}

func Create(plan Plan) (Result, error) {
	result := Result{OutputDir: plan.OutputDir, Diagnostics: append([]string{}, plan.Diagnostics...)}
	if err := os.RemoveAll(plan.OutputDir); err != nil {
		return result, err
	}
	if err := os.MkdirAll(plan.OutputDir, 0o755); err != nil {
		return result, err
	}
	executableOutput := filepath.Join(plan.OutputDir, filepath.Base(plan.ExecutablePath))
	if err := copyFile(plan.ExecutablePath, executableOutput, 0o755); err != nil {
		return result, err
	}
	result.ExecutablePath = executableOutput
	for _, asset := range plan.Assets {
		outputPath := filepath.Join(plan.OutputDir, asset.OutputName)
		if err := copyFile(asset.SourcePath, outputPath, 0o755); err != nil {
			return result, err
		}
		result.CopiedAssetPaths = append(result.CopiedAssetPaths, outputPath)
	}
	return result, nil
}

func copyFile(sourcePath string, outputPath string, mode os.FileMode) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer source.Close()
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return err
	}
	output, err := os.OpenFile(outputPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(output, source); err != nil {
		output.Close()
		return err
	}
	return output.Close()
}
