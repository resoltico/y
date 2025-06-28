package widgets

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type ParameterPanel struct {
	container              *fyne.Container
	parametersContent      *fyne.Container
	parameterChangeHandler func(string, interface{})

	// Reusable widgets for 2D Otsu
	windowSizeSlider        *widget.Slider
	windowSizeLabel         *widget.Label
	histBinsSlider          *widget.Slider
	histBinsLabel           *widget.Label
	smoothingStrengthSlider *widget.Slider
	smoothingStrengthLabel  *widget.Label
	noiseRobustnessCheck    *widget.Check
	gaussianPreprocessCheck *widget.Check
	useClaheCheck           *widget.Check
	claheClipLimitSlider    *widget.Slider
	claheClipLimitLabel     *widget.Label
	claheTileSizeSlider     *widget.Slider
	claheTileSizeLabel      *widget.Label
	guidedFilteringCheck    *widget.Check
	guidedRadiusSlider      *widget.Slider
	guidedRadiusLabel       *widget.Label
	guidedEpsilonSlider     *widget.Slider
	guidedEpsilonLabel      *widget.Label
	parallelProcessingCheck *widget.Check

	// Reusable widgets for Iterative Triclass
	initialMethod                *widget.Select
	maxIterSlider                *widget.Slider
	maxIterLabel                 *widget.Label
	convergencePrecisionSlider   *widget.Slider
	convergencePrecisionLabel    *widget.Label
	minTBDSlider                 *widget.Slider
	minTBDLabel                  *widget.Label
	classSeparationSlider        *widget.Slider
	classSeparationLabel         *widget.Label
	preprocessingCheck           *widget.Check
	cleanupCheck                 *widget.Check
	bordersCheck                 *widget.Check
	triclassNoiseRobustnessCheck *widget.Check
	triclassGuidedFilteringCheck *widget.Check
	triclassGuidedRadiusSlider   *widget.Slider
	triclassGuidedRadiusLabel    *widget.Label
	triclassGuidedEpsilonSlider  *widget.Slider
	triclassGuidedEpsilonLabel   *widget.Label
	triclassParallelCheck        *widget.Check

	currentAlgorithm string
}

func NewParameterPanel() *ParameterPanel {
	panel := &ParameterPanel{}
	panel.setupPanel()
	panel.createWidgets()
	return panel
}

func (pp *ParameterPanel) setupPanel() {
	pp.parametersContent = container.NewVBox(
		widget.NewLabel("Parameters:"),
	)
	pp.container = container.NewVBox(pp.parametersContent)
}

func (pp *ParameterPanel) createWidgets() {
	// Create 2D Otsu widgets
	pp.windowSizeSlider = widget.NewSlider(3, 21)
	pp.windowSizeSlider.Step = 2
	pp.windowSizeLabel = widget.NewLabel("Window Size: 7")

	pp.histBinsSlider = widget.NewSlider(0, 256)
	pp.histBinsLabel = widget.NewLabel("Histogram Bins: Auto")

	pp.smoothingStrengthSlider = widget.NewSlider(0.0, 5.0)
	pp.smoothingStrengthLabel = widget.NewLabel("Smoothing Strength: 1.0")

	pp.noiseRobustnessCheck = widget.NewCheck("MAOTSU Noise Robustness", nil)
	pp.gaussianPreprocessCheck = widget.NewCheck("Gaussian Preprocessing", nil)
	pp.useClaheCheck = widget.NewCheck("CLAHE Contrast Enhancement", nil)

	pp.claheClipLimitSlider = widget.NewSlider(1.0, 8.0)
	pp.claheClipLimitLabel = widget.NewLabel("CLAHE Clip Limit: 3.0")

	pp.claheTileSizeSlider = widget.NewSlider(4, 16)
	pp.claheTileSizeLabel = widget.NewLabel("CLAHE Tile Size: 8")

	pp.guidedFilteringCheck = widget.NewCheck("Guided Filtering", nil)

	pp.guidedRadiusSlider = widget.NewSlider(1, 8)
	pp.guidedRadiusLabel = widget.NewLabel("Guided Radius: 4")

	pp.guidedEpsilonSlider = widget.NewSlider(0.001, 0.5)
	pp.guidedEpsilonLabel = widget.NewLabel("Guided Epsilon: 0.05")

	pp.parallelProcessingCheck = widget.NewCheck("Parallel Processing", nil)

	// Create Iterative Triclass widgets
	pp.initialMethod = widget.NewSelect([]string{"otsu", "mean", "median", "triangle"}, nil)

	pp.maxIterSlider = widget.NewSlider(3, 15)
	pp.maxIterLabel = widget.NewLabel("Max Iterations: 8")

	pp.convergencePrecisionSlider = widget.NewSlider(0.5, 2.0)
	pp.convergencePrecisionLabel = widget.NewLabel("Convergence Precision: 1.0")

	pp.minTBDSlider = widget.NewSlider(0.001, 0.2)
	pp.minTBDLabel = widget.NewLabel("Min TBD Fraction: 0.010")

	pp.classSeparationSlider = widget.NewSlider(0.1, 0.8)
	pp.classSeparationLabel = widget.NewLabel("Class Separation: 0.50")

	pp.preprocessingCheck = widget.NewCheck("Advanced Preprocessing", nil)
	pp.cleanupCheck = widget.NewCheck("Result Cleanup", nil)
	pp.bordersCheck = widget.NewCheck("Preserve Borders", nil)
	pp.triclassNoiseRobustnessCheck = widget.NewCheck("Non-Local Means Denoising", nil)
	pp.triclassGuidedFilteringCheck = widget.NewCheck("Guided Filtering", nil)

	pp.triclassGuidedRadiusSlider = widget.NewSlider(1, 8)
	pp.triclassGuidedRadiusLabel = widget.NewLabel("Guided Radius: 6")

	pp.triclassGuidedEpsilonSlider = widget.NewSlider(0.01, 0.5)
	pp.triclassGuidedEpsilonLabel = widget.NewLabel("Guided Epsilon: 0.15")

	pp.triclassParallelCheck = widget.NewCheck("Parallel Processing", nil)
}

func (pp *ParameterPanel) GetContainer() *fyne.Container {
	return pp.container
}

func (pp *ParameterPanel) SetParameterChangeHandler(handler func(string, interface{})) {
	pp.parameterChangeHandler = handler
	pp.setupEventHandlers()
}

func (pp *ParameterPanel) setupEventHandlers() {
	if pp.parameterChangeHandler == nil {
		return
	}

	// 2D Otsu handlers
	pp.windowSizeSlider.OnChanged = func(value float64) {
		intValue := int(value)
		if intValue%2 == 0 {
			intValue++
		}
		pp.windowSizeLabel.SetText("Window Size: " + strconv.Itoa(intValue))
		pp.parameterChangeHandler("window_size", intValue)
	}

	pp.histBinsSlider.OnChanged = func(value float64) {
		intValue := int(value)
		if intValue == 0 {
			pp.histBinsLabel.SetText("Histogram Bins: Auto")
		} else {
			pp.histBinsLabel.SetText("Histogram Bins: " + strconv.Itoa(intValue))
		}
		pp.parameterChangeHandler("histogram_bins", intValue)
	}

	pp.smoothingStrengthSlider.OnChanged = func(value float64) {
		pp.smoothingStrengthLabel.SetText("Smoothing Strength: " + strconv.FormatFloat(value, 'f', 1, 64))
		pp.parameterChangeHandler("smoothing_strength", value)
	}

	pp.noiseRobustnessCheck.OnChanged = func(checked bool) {
		pp.parameterChangeHandler("noise_robustness", checked)
	}

	pp.gaussianPreprocessCheck.OnChanged = func(checked bool) {
		pp.parameterChangeHandler("gaussian_preprocessing", checked)
	}

	pp.useClaheCheck.OnChanged = func(checked bool) {
		pp.parameterChangeHandler("use_clahe", checked)
	}

	pp.claheClipLimitSlider.OnChanged = func(value float64) {
		pp.claheClipLimitLabel.SetText("CLAHE Clip Limit: " + strconv.FormatFloat(value, 'f', 1, 64))
		pp.parameterChangeHandler("clahe_clip_limit", value)
	}

	pp.claheTileSizeSlider.OnChanged = func(value float64) {
		intValue := int(value)
		pp.claheTileSizeLabel.SetText("CLAHE Tile Size: " + strconv.Itoa(intValue))
		pp.parameterChangeHandler("clahe_tile_size", intValue)
	}

	pp.guidedFilteringCheck.OnChanged = func(checked bool) {
		pp.parameterChangeHandler("guided_filtering", checked)
	}

	pp.guidedRadiusSlider.OnChanged = func(value float64) {
		intValue := int(value)
		pp.guidedRadiusLabel.SetText("Guided Radius: " + strconv.Itoa(intValue))
		pp.parameterChangeHandler("guided_radius", intValue)
	}

	pp.guidedEpsilonSlider.OnChanged = func(value float64) {
		pp.guidedEpsilonLabel.SetText("Guided Epsilon: " + strconv.FormatFloat(value, 'f', 3, 64))
		pp.parameterChangeHandler("guided_epsilon", value)
	}

	pp.parallelProcessingCheck.OnChanged = func(checked bool) {
		pp.parameterChangeHandler("parallel_processing", checked)
	}

	// Iterative Triclass handlers
	pp.initialMethod.OnChanged = func(value string) {
		pp.parameterChangeHandler("initial_threshold_method", value)
	}

	pp.maxIterSlider.OnChanged = func(value float64) {
		intValue := int(value)
		pp.maxIterLabel.SetText("Max Iterations: " + strconv.Itoa(intValue))
		pp.parameterChangeHandler("max_iterations", intValue)
	}

	pp.convergencePrecisionSlider.OnChanged = func(value float64) {
		pp.convergencePrecisionLabel.SetText("Convergence Precision: " + strconv.FormatFloat(value, 'f', 1, 64))
		pp.parameterChangeHandler("convergence_precision", value)
	}

	pp.minTBDSlider.OnChanged = func(value float64) {
		pp.minTBDLabel.SetText("Min TBD Fraction: " + strconv.FormatFloat(value, 'f', 3, 64))
		pp.parameterChangeHandler("minimum_tbd_fraction", value)
	}

	pp.classSeparationSlider.OnChanged = func(value float64) {
		pp.classSeparationLabel.SetText("Class Separation: " + strconv.FormatFloat(value, 'f', 2, 64))
		pp.parameterChangeHandler("class_separation", value)
	}

	pp.preprocessingCheck.OnChanged = func(checked bool) {
		pp.parameterChangeHandler("preprocessing", checked)
	}

	pp.cleanupCheck.OnChanged = func(checked bool) {
		pp.parameterChangeHandler("result_cleanup", checked)
	}

	pp.bordersCheck.OnChanged = func(checked bool) {
		pp.parameterChangeHandler("preserve_borders", checked)
	}

	pp.triclassNoiseRobustnessCheck.OnChanged = func(checked bool) {
		pp.parameterChangeHandler("noise_robustness", checked)
	}

	pp.triclassGuidedFilteringCheck.OnChanged = func(checked bool) {
		pp.parameterChangeHandler("guided_filtering", checked)
	}

	pp.triclassGuidedRadiusSlider.OnChanged = func(value float64) {
		intValue := int(value)
		pp.triclassGuidedRadiusLabel.SetText("Guided Radius: " + strconv.Itoa(intValue))
		pp.parameterChangeHandler("guided_radius", intValue)
	}

	pp.triclassGuidedEpsilonSlider.OnChanged = func(value float64) {
		pp.triclassGuidedEpsilonLabel.SetText("Guided Epsilon: " + strconv.FormatFloat(value, 'f', 3, 64))
		pp.parameterChangeHandler("guided_epsilon", value)
	}

	pp.triclassParallelCheck.OnChanged = func(checked bool) {
		pp.parameterChangeHandler("parallel_processing", checked)
	}
}

func (pp *ParameterPanel) UpdateParameters(algorithm string, params map[string]interface{}) {
	if pp.currentAlgorithm == algorithm {
		pp.updateValues(params)
		return
	}

	pp.currentAlgorithm = algorithm
	pp.parametersContent.RemoveAll()
	pp.parametersContent.Add(widget.NewLabel("Parameters:"))

	switch algorithm {
	case "2D Otsu":
		pp.buildOtsu2DParameters(params)
	case "Iterative Triclass":
		pp.buildTriclassParameters(params)
	}

	pp.container.Refresh()
}

func (pp *ParameterPanel) updateValues(params map[string]interface{}) {
	switch pp.currentAlgorithm {
	case "2D Otsu":
		pp.updateOtsu2DValues(params)
	case "Iterative Triclass":
		pp.updateTriclassValues(params)
	}
}

func (pp *ParameterPanel) updateOtsu2DValues(params map[string]interface{}) {
	if windowSize := pp.getIntParam(params, "window_size", 7); windowSize != int(pp.windowSizeSlider.Value) {
		pp.windowSizeSlider.SetValue(float64(windowSize))
	}
	if histBins := pp.getIntParam(params, "histogram_bins", 0); histBins != int(pp.histBinsSlider.Value) {
		pp.histBinsSlider.SetValue(float64(histBins))
	}
	if smoothing := pp.getFloatParam(params, "smoothing_strength", 1.0); smoothing != pp.smoothingStrengthSlider.Value {
		pp.smoothingStrengthSlider.SetValue(smoothing)
	}
	if noiseRob := pp.getBoolParam(params, "noise_robustness", true); noiseRob != pp.noiseRobustnessCheck.Checked {
		pp.noiseRobustnessCheck.SetChecked(noiseRob)
	}
	if gaussian := pp.getBoolParam(params, "gaussian_preprocessing", true); gaussian != pp.gaussianPreprocessCheck.Checked {
		pp.gaussianPreprocessCheck.SetChecked(gaussian)
	}
	if clahe := pp.getBoolParam(params, "use_clahe", false); clahe != pp.useClaheCheck.Checked {
		pp.useClaheCheck.SetChecked(clahe)
	}
	if clipLimit := pp.getFloatParam(params, "clahe_clip_limit", 3.0); clipLimit != pp.claheClipLimitSlider.Value {
		pp.claheClipLimitSlider.SetValue(clipLimit)
	}
	if tileSize := pp.getIntParam(params, "clahe_tile_size", 8); tileSize != int(pp.claheTileSizeSlider.Value) {
		pp.claheTileSizeSlider.SetValue(float64(tileSize))
	}
	if guided := pp.getBoolParam(params, "guided_filtering", false); guided != pp.guidedFilteringCheck.Checked {
		pp.guidedFilteringCheck.SetChecked(guided)
	}
	if radius := pp.getIntParam(params, "guided_radius", 4); radius != int(pp.guidedRadiusSlider.Value) {
		pp.guidedRadiusSlider.SetValue(float64(radius))
	}
	if epsilon := pp.getFloatParam(params, "guided_epsilon", 0.05); epsilon != pp.guidedEpsilonSlider.Value {
		pp.guidedEpsilonSlider.SetValue(epsilon)
	}
	if parallel := pp.getBoolParam(params, "parallel_processing", true); parallel != pp.parallelProcessingCheck.Checked {
		pp.parallelProcessingCheck.SetChecked(parallel)
	}
}

func (pp *ParameterPanel) updateTriclassValues(params map[string]interface{}) {
	if method := pp.getStringParam(params, "initial_threshold_method", "otsu"); method != pp.initialMethod.Selected {
		pp.initialMethod.SetSelected(method)
	}
	if maxIter := pp.getIntParam(params, "max_iterations", 8); maxIter != int(pp.maxIterSlider.Value) {
		pp.maxIterSlider.SetValue(float64(maxIter))
	}
	if precision := pp.getFloatParam(params, "convergence_precision", 1.0); precision != pp.convergencePrecisionSlider.Value {
		pp.convergencePrecisionSlider.SetValue(precision)
	}
	if minTBD := pp.getFloatParam(params, "minimum_tbd_fraction", 0.01); minTBD != pp.minTBDSlider.Value {
		pp.minTBDSlider.SetValue(minTBD)
	}
	if separation := pp.getFloatParam(params, "class_separation", 0.5); separation != pp.classSeparationSlider.Value {
		pp.classSeparationSlider.SetValue(separation)
	}
	if preprocess := pp.getBoolParam(params, "preprocessing", true); preprocess != pp.preprocessingCheck.Checked {
		pp.preprocessingCheck.SetChecked(preprocess)
	}
	if cleanup := pp.getBoolParam(params, "result_cleanup", true); cleanup != pp.cleanupCheck.Checked {
		pp.cleanupCheck.SetChecked(cleanup)
	}
	if borders := pp.getBoolParam(params, "preserve_borders", false); borders != pp.bordersCheck.Checked {
		pp.bordersCheck.SetChecked(borders)
	}
	if noiseRob := pp.getBoolParam(params, "noise_robustness", true); noiseRob != pp.triclassNoiseRobustnessCheck.Checked {
		pp.triclassNoiseRobustnessCheck.SetChecked(noiseRob)
	}
	if guided := pp.getBoolParam(params, "guided_filtering", true); guided != pp.triclassGuidedFilteringCheck.Checked {
		pp.triclassGuidedFilteringCheck.SetChecked(guided)
	}
	if radius := pp.getIntParam(params, "guided_radius", 6); radius != int(pp.triclassGuidedRadiusSlider.Value) {
		pp.triclassGuidedRadiusSlider.SetValue(float64(radius))
	}
	if epsilon := pp.getFloatParam(params, "guided_epsilon", 0.15); epsilon != pp.triclassGuidedEpsilonSlider.Value {
		pp.triclassGuidedEpsilonSlider.SetValue(epsilon)
	}
	if parallel := pp.getBoolParam(params, "parallel_processing", true); parallel != pp.triclassParallelCheck.Checked {
		pp.triclassParallelCheck.SetChecked(parallel)
	}
}

func (pp *ParameterPanel) buildOtsu2DParameters(params map[string]interface{}) {
	windowSize := pp.getIntParam(params, "window_size", 7)
	pp.windowSizeSlider.SetValue(float64(windowSize))
	pp.windowSizeLabel.SetText("Window Size: " + strconv.Itoa(windowSize))

	histBins := pp.getIntParam(params, "histogram_bins", 0)
	pp.histBinsSlider.SetValue(float64(histBins))
	if histBins == 0 {
		pp.histBinsLabel.SetText("Histogram Bins: Auto")
	} else {
		pp.histBinsLabel.SetText("Histogram Bins: " + strconv.Itoa(histBins))
	}

	smoothingStrength := pp.getFloatParam(params, "smoothing_strength", 1.0)
	pp.smoothingStrengthSlider.SetValue(smoothingStrength)
	pp.smoothingStrengthLabel.SetText("Smoothing Strength: " + strconv.FormatFloat(smoothingStrength, 'f', 1, 64))

	clipLimit := pp.getFloatParam(params, "clahe_clip_limit", 3.0)
	pp.claheClipLimitSlider.SetValue(clipLimit)
	pp.claheClipLimitLabel.SetText("CLAHE Clip Limit: " + strconv.FormatFloat(clipLimit, 'f', 1, 64))

	tileSize := pp.getIntParam(params, "clahe_tile_size", 8)
	pp.claheTileSizeSlider.SetValue(float64(tileSize))
	pp.claheTileSizeLabel.SetText("CLAHE Tile Size: " + strconv.Itoa(tileSize))

	guidedRadius := pp.getIntParam(params, "guided_radius", 4)
	pp.guidedRadiusSlider.SetValue(float64(guidedRadius))
	pp.guidedRadiusLabel.SetText("Guided Radius: " + strconv.Itoa(guidedRadius))

	guidedEpsilon := pp.getFloatParam(params, "guided_epsilon", 0.05)
	pp.guidedEpsilonSlider.SetValue(guidedEpsilon)
	pp.guidedEpsilonLabel.SetText("Guided Epsilon: " + strconv.FormatFloat(guidedEpsilon, 'f', 3, 64))

	pp.noiseRobustnessCheck.SetChecked(pp.getBoolParam(params, "noise_robustness", true))
	pp.gaussianPreprocessCheck.SetChecked(pp.getBoolParam(params, "gaussian_preprocessing", true))
	pp.useClaheCheck.SetChecked(pp.getBoolParam(params, "use_clahe", false))
	pp.guidedFilteringCheck.SetChecked(pp.getBoolParam(params, "guided_filtering", false))
	pp.parallelProcessingCheck.SetChecked(pp.getBoolParam(params, "parallel_processing", true))

	// Basic parameters
	basicGroup := container.NewVBox(
		container.NewHBox(
			container.NewVBox(pp.windowSizeLabel, pp.windowSizeSlider),
			container.NewVBox(pp.histBinsLabel, pp.histBinsSlider),
			container.NewVBox(pp.smoothingStrengthLabel, pp.smoothingStrengthSlider),
		),
	)

	// Preprocessing options
	preprocessingGroup := container.NewVBox(
		widget.NewCard("Preprocessing", "",
			container.NewVBox(
				container.NewHBox(pp.noiseRobustnessCheck, pp.gaussianPreprocessCheck),
				container.NewHBox(pp.useClaheCheck, pp.guidedFilteringCheck),
			),
		),
	)

	// CLAHE parameters (shown when CLAHE is enabled)
	claheGroup := container.NewVBox(
		widget.NewCard("CLAHE Parameters", "",
			container.NewHBox(
				container.NewVBox(pp.claheClipLimitLabel, pp.claheClipLimitSlider),
				container.NewVBox(pp.claheTileSizeLabel, pp.claheTileSizeSlider),
			),
		),
	)

	// Guided filtering parameters (shown when guided filtering is enabled)
	guidedGroup := container.NewVBox(
		widget.NewCard("Guided Filtering Parameters", "",
			container.NewHBox(
				container.NewVBox(pp.guidedRadiusLabel, pp.guidedRadiusSlider),
				container.NewVBox(pp.guidedEpsilonLabel, pp.guidedEpsilonSlider),
			),
		),
	)

	// Performance options
	performanceGroup := container.NewVBox(
		widget.NewCard("Performance", "",
			container.NewVBox(pp.parallelProcessingCheck),
		),
	)

	pp.parametersContent.Add(basicGroup)
	pp.parametersContent.Add(preprocessingGroup)
	pp.parametersContent.Add(claheGroup)
	pp.parametersContent.Add(guidedGroup)
	pp.parametersContent.Add(performanceGroup)
}

func (pp *ParameterPanel) buildTriclassParameters(params map[string]interface{}) {
	pp.initialMethod.SetSelected(pp.getStringParam(params, "initial_threshold_method", "otsu"))

	maxIter := pp.getIntParam(params, "max_iterations", 8)
	pp.maxIterSlider.SetValue(float64(maxIter))
	pp.maxIterLabel.SetText("Max Iterations: " + strconv.Itoa(maxIter))

	convergencePrecision := pp.getFloatParam(params, "convergence_precision", 1.0)
	pp.convergencePrecisionSlider.SetValue(convergencePrecision)
	pp.convergencePrecisionLabel.SetText("Convergence Precision: " + strconv.FormatFloat(convergencePrecision, 'f', 1, 64))

	minTBD := pp.getFloatParam(params, "minimum_tbd_fraction", 0.01)
	pp.minTBDSlider.SetValue(minTBD)
	pp.minTBDLabel.SetText("Min TBD Fraction: " + strconv.FormatFloat(minTBD, 'f', 3, 64))

	classSeparation := pp.getFloatParam(params, "class_separation", 0.5)
	pp.classSeparationSlider.SetValue(classSeparation)
	pp.classSeparationLabel.SetText("Class Separation: " + strconv.FormatFloat(classSeparation, 'f', 2, 64))

	guidedRadius := pp.getIntParam(params, "guided_radius", 6)
	pp.triclassGuidedRadiusSlider.SetValue(float64(guidedRadius))
	pp.triclassGuidedRadiusLabel.SetText("Guided Radius: " + strconv.Itoa(guidedRadius))

	guidedEpsilon := pp.getFloatParam(params, "guided_epsilon", 0.15)
	pp.triclassGuidedEpsilonSlider.SetValue(guidedEpsilon)
	pp.triclassGuidedEpsilonLabel.SetText("Guided Epsilon: " + strconv.FormatFloat(guidedEpsilon, 'f', 3, 64))

	pp.preprocessingCheck.SetChecked(pp.getBoolParam(params, "preprocessing", true))
	pp.cleanupCheck.SetChecked(pp.getBoolParam(params, "result_cleanup", true))
	pp.bordersCheck.SetChecked(pp.getBoolParam(params, "preserve_borders", false))
	pp.triclassNoiseRobustnessCheck.SetChecked(pp.getBoolParam(params, "noise_robustness", true))
	pp.triclassGuidedFilteringCheck.SetChecked(pp.getBoolParam(params, "guided_filtering", true))
	pp.triclassParallelCheck.SetChecked(pp.getBoolParam(params, "parallel_processing", true))

	// Algorithm parameters
	algorithmGroup := container.NewVBox(
		container.NewHBox(
			container.NewVBox(widget.NewLabel("Initial Method"), pp.initialMethod),
			container.NewVBox(pp.maxIterLabel, pp.maxIterSlider),
			container.NewVBox(pp.convergencePrecisionLabel, pp.convergencePrecisionSlider),
		),
		container.NewHBox(
			container.NewVBox(pp.minTBDLabel, pp.minTBDSlider),
			container.NewVBox(pp.classSeparationLabel, pp.classSeparationSlider),
		),
	)

	// Preprocessing options
	preprocessingGroup := container.NewVBox(
		widget.NewCard("Preprocessing", "",
			container.NewVBox(
				container.NewHBox(pp.preprocessingCheck, pp.triclassNoiseRobustnessCheck),
				container.NewHBox(pp.triclassGuidedFilteringCheck, pp.cleanupCheck),
				pp.bordersCheck,
			),
		),
	)

	// Guided filtering parameters
	guidedGroup := container.NewVBox(
		widget.NewCard("Guided Filtering Parameters", "",
			container.NewHBox(
				container.NewVBox(pp.triclassGuidedRadiusLabel, pp.triclassGuidedRadiusSlider),
				container.NewVBox(pp.triclassGuidedEpsilonLabel, pp.triclassGuidedEpsilonSlider),
			),
		),
	)

	// Performance options
	performanceGroup := container.NewVBox(
		widget.NewCard("Performance", "",
			container.NewVBox(pp.triclassParallelCheck),
		),
	)

	pp.parametersContent.Add(algorithmGroup)
	pp.parametersContent.Add(preprocessingGroup)
	pp.parametersContent.Add(guidedGroup)
	pp.parametersContent.Add(performanceGroup)
}

func (pp *ParameterPanel) getIntParam(params map[string]interface{}, key string, defaultValue int) int {
	if value, ok := params[key].(int); ok {
		return value
	}
	return defaultValue
}

func (pp *ParameterPanel) getStringParam(params map[string]interface{}, key string, defaultValue string) string {
	if value, ok := params[key].(string); ok {
		return value
	}
	return defaultValue
}

func (pp *ParameterPanel) getFloatParam(params map[string]interface{}, key string, defaultValue float64) float64 {
	if value, ok := params[key].(float64); ok {
		return value
	}
	return defaultValue
}

func (pp *ParameterPanel) getBoolParam(params map[string]interface{}, key string, defaultValue bool) bool {
	if value, ok := params[key].(bool); ok {
		return value
	}
	return defaultValue
}
