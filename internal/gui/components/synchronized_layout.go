package components

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

type SynchronizedLayout struct {
	imageDisplay  *ImageDisplay
	toolbar       *ResponsiveToolbar
	parameters    *ParametersSection
	mainContainer *fyne.Container
}

func NewSynchronizedLayout(imageDisplay *ImageDisplay, toolbar *ResponsiveToolbar, parameters *ParametersSection) *SynchronizedLayout {
	layout := &SynchronizedLayout{
		imageDisplay: imageDisplay,
		toolbar:      toolbar,
		parameters:   parameters,
	}

	layout.setupLayout()
	return layout
}

func (sl *SynchronizedLayout) setupLayout() {
	// Create responsive toolbar that positions elements based on image split
	responsiveToolbar := sl.createResponsiveToolbar()

	sl.mainContainer = container.NewVBox(
		sl.imageDisplay.GetContainer(),
		responsiveToolbar,
		sl.parameters.GetContainer(),
	)
}

func (sl *SynchronizedLayout) createResponsiveToolbar() *fyne.Container {
	// Left section: Load/Save (fixed position)
	leftSection := container.NewHBox(
		sl.toolbar.LoadButton,
		sl.toolbar.SaveButton,
	)

	// Center section with three zones aligned to image areas
	centerSection := container.NewHBox(
		// Algorithm group (aligned to Original image center)
		sl.toolbar.AlgorithmGroup,

		// Generate button (aligned to split divider)
		container.NewCenter(sl.toolbar.GenerateButton),

		// Status group (aligned to Preview image center)
		sl.toolbar.StatusGroup,
	)

	// Right section: Metrics (fixed position)
	rightSection := container.NewHBox(
		sl.toolbar.MetricsLabel,
	)

	// Use Border layout to maintain responsive positioning
	return container.NewBorder(
		nil, nil,
		leftSection,   // West: Load/Save
		rightSection,  // East: Metrics
		centerSection, // Center: Algorithm | Generate | Status
	)
}

func (sl *SynchronizedLayout) GetContainer() *fyne.Container {
	return sl.mainContainer
}

// UpdateSplitPosition updates toolbar element positions based on image split
func (sl *SynchronizedLayout) UpdateSplitPosition(offset float32) {
	// The toolbar will automatically adjust due to responsive layout structure
	// No manual positioning needed as Border layout handles distribution
}
