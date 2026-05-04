package gtk

type WidgetFeature string

const (
	CreateLabel     WidgetFeature = "create-label"
	CreateButton    WidgetFeature = "create-button"
	CreateTextInput WidgetFeature = "create-text-input"
	CreateContainer WidgetFeature = "create-container"
	SetProperty     WidgetFeature = "set-property"
	AddChild        WidgetFeature = "add-child"
	ShowWidget      WidgetFeature = "show-widget"
	HideWidget      WidgetFeature = "hide-widget"
)

func WidgetFeatures() []WidgetFeature {
	return []WidgetFeature{
		CreateLabel,
		CreateButton,
		CreateTextInput,
		CreateContainer,
		SetProperty,
		AddChild,
		ShowWidget,
		HideWidget,
	}
}
