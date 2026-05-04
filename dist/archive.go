package dist

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
)

func writeArchive(plan Plan) (string, string, error) {
	if err := os.MkdirAll(filepath.Dir(plan.ArchivePath), 0o755); err != nil {
		return "", "", err
	}
	var err error
	if plan.Platform.GOOS == "windows" {
		err = writeZip(plan.Root, plan.ArchivePath)
	} else {
		err = writeTarGz(plan.Root, plan.ArchivePath)
	}
	if err != nil {
		return "", "", err
	}
	if err := writeSHA256(plan.ArchivePath, plan.ChecksumPath); err != nil {
		return "", "", err
	}
	return plan.ArchivePath, plan.ChecksumPath, nil
}

func writeTarGz(root string, outputPath string) error {
	output, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer output.Close()
	gzipWriter := gzip.NewWriter(output)
	defer gzipWriter.Close()
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()
	base := filepath.Dir(root)
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() {
			return walkErr
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(relativeArchivePath(base, path))
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		return writeArchiveFile(tarWriter, path)
	})
}

func writeZip(root string, outputPath string) error {
	output, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer output.Close()
	zipWriter := zip.NewWriter(output)
	defer zipWriter.Close()
	base := filepath.Dir(root)
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() {
			return walkErr
		}
		writer, err := zipWriter.Create(filepath.ToSlash(relativeArchivePath(base, path)))
		if err != nil {
			return err
		}
		return writeArchiveFile(writer, path)
	})
}

func writeArchiveFile(writer io.Writer, path string) error {
	input, err := os.Open(path)
	if err != nil {
		return err
	}
	defer input.Close()
	_, err = io.Copy(writer, input)
	return err
}

func relativeArchivePath(base string, path string) string {
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return filepath.Base(path)
	}
	return rel
}

func writeSHA256(archivePath string, checksumPath string) error {
	input, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer input.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, input); err != nil {
		return err
	}
	sum := hex.EncodeToString(hash.Sum(nil)) + "  " + filepath.Base(archivePath) + "\n"
	return os.WriteFile(checksumPath, []byte(sum), 0o644)
}
