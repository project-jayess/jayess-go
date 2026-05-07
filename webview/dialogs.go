package webview

type DialogFeature string

const (
	OpenFileDialog DialogFeature = "open-file-dialog"
	SaveFileDialog DialogFeature = "save-file-dialog"
)

func DialogFeatures() []DialogFeature {
	return []DialogFeature{OpenFileDialog, SaveFileDialog}
}
