package binding

import (
	"path/filepath"
	"strings"
)

func sharedLibraryLinkArg(modulePath string, library string) string {
	if strings.HasPrefix(library, "-l") {
		return library
	}
	if looksLikeLibraryPath(library) {
		return normalizeSourceKey(modulePath, library)
	}
	return "-l" + library
}

func looksLikeLibraryPath(library string) bool {
	return strings.Contains(library, "/") || filepath.Ext(library) != ""
}
