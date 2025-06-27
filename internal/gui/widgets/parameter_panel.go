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
	windowSizeSlider     *widget.Slider
	windowSizeLabel      *widget.Label
	histBinsSlider       *widget.Slider
	histBinsLabel        *widget.Label
	neighMetric          *widget.Select
	pixelWeightSlider    *widget.Slider
	pixelWeightLabel     *widget.Label
	smoothingSigmaSlider *widget.Slider
	smoothingSigmaLabel  *widget.Label
	useLogCheck          *widget.Check
	normalizeCheck       *widget.Check
	contrastCheck        *widget.Check

	// Reusable widgets for Iterative Triclass
	initialMethod     *widget.Select
	maxIterSlider     *widget.Slider
	maxIterLabel      *widget.Label
	convEpsilonSlider *widget.Slider
	convEpsilonLabel  *widget.Label
	minTBDSlider      *widget.Slider
	minTBDLabel       *widget.Label
	gapFactorSlider   *widget.Slider
	gapFactorLabel    *widget.Label
	preprocessCheck   *widget.Check
	cleanupCheck      *widget.Check
	bordersCheck      *widget.Check

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

	pp.histBinsSlider = widget.NewSlider(16, 256)
	pp.histBinsLabel = widget.NewLabel("Histogram Bins: 64")

	pp.neighMetric = widget.NewSelect([]string{"mean", "median", "gaussian"}, nil)

	pp.pixelWeightSlider = widget.NewSlider(0.0, 1.0)
	pp.pixelWeightLabel = widget.NewLabel("Pixel Weight: 0.50")

	pp.smoothingSigmaSlider = widget.NewSlider(0.0, 5.0)
	pp.smoothingSigmaLabel = widget.NewLabel("Smoothing Sigma: 1.0")

	pp.useLogCheck = widget.NewCheck("Use Log Histogram", nil)
	pp.normalizeCheck = widget.NewCheck("Normalize Histogram", nil)
	pp.contrastCheck = widget.NewCheck("Apply Contrast Enhancement", nil)

	// Create Iterative Triclass widgets
	pp.initialMethod = widget.NewSelect([]string{"otsu", "mean", "median"}, nil)

	pp.maxIterSlider = widget.NewSlider(1, 20)
	pp.maxIterLabel = widget.NewLabel("Max Iterations: 10")

	pp.convEpsilonSlider = widget.NewSlider(0.1, 10.0)
	pp.convEpsilonLabel = widget.NewLabel("Convergence Epsilon: 1.0")

	pp.minTBDSlider = widget.NewSlider(0.001, 0.2)
	pp.minTBDLabel = widget.NewLabel("Min TBD Fraction: 0.010")

	pp.gapFactorSlider = widget.NewSlider(0.0, 1.0)
	pp.gapFactorLabel = widget.NewLabel("Gap Factor: 0.50")

	pp.preprocessCheck = widget.NewCheck("Apply Preprocessing", nil)
	pp.cleanupCheck = widget.NewCheck("Apply Cleanup", nil)
	pp.bordersCheck = widget.NewCheck("Preserve Borders", nil)
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
		pp.histBinsLabel.SetText("Histogram Bins: " + strconv.Itoa(intValue))
		pp.parameterChangeHandler("histogram_bins", intValue)
	}

	pp.neighMetric.OnChanged = func(value string) {
		pp.parameterChangeHandler("neighbourhood_metric", value)
	}

	pp.pixelWeightSlider.OnChanged = func(value float64) {
		pp.pixelWeightLabel.SetText("Pixel Weight: " + strconv.FormatFloat(value, 'f', 2, 64))
		pp.parameterChangeHandler("pixel_weight_factor", value)
	}

	pp.smoothingSigmaSlider.OnChanged = func(value float64) {
		pp.smoothingSigmaLabel.SetText("Smoothing Sigma: " + strconv.FormatFloat(value, 'f', 1, 64))
		pp.parameterChangeHandler("smoothing_sigma", value)
	}

	pp.useLogCheck.OnChanged = func(checked bool) {
		pp.parameterChangeHandler("use_log_histogram", checked)
	}

	pp.normalizeCheck.OnChanged = func(checked bool) {
		pp.parameterChangeHandler("normalize_histogram", checked)
	}

	pp.contrastCheck.OnChanged = func(checked bool) {
		pp.parameterChangeHandler("apply_contrast_enhancement", checked)
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

	pp.convEpsilonSlider.OnChanged = func(value float64) {
		pp.convEpsilonLabel.SetText("Convergence Epsilon: " + strconv.FormatFloat(value, 'f', 1, 64))
		pp.parameterChangeHandler("convergence_epsilon", value)
	}

	pp.minTBDSlider.OnChanged = func(value float64) {
		pp.minTBDLabel.SetText("Min TBD Fraction: " + strconv.FormatFloat(value, 'f', 3, 64))
		pp.parameterChangeHandler("minimum_tbd_fraction", value)
	}

	pp.gapFactorSlider.OnChanged = func(value float64) {
		pp.gapFactorLabel.SetText("Gap Factor: " + strconv.FormatFloat(value, 'f', 2, 64))
		pp.parameterChangeHandler("lower_upper_gap_factor", value)
	}

	pp.preprocessCheck.OnChanged = func(checked bool) {
		pp.parameterChangeHandler("apply_preprocessing", checked)
	}

	pp.cleanupCheck.OnChanged = func(checked bool) {
		pp.parameterChangeHandler("apply_cleanup", checked)
	}

	pp.bordersCheck.OnChanged = func(checked bool) {
		pp.parameterChangeHandler("preserve_borders", checked)
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
	if histBins := pp.getIntParam(params, "histogram_bins", 64); histBins != int(pp.histBinsSlider.Value) {
		pp.histBinsSlider.SetValue(float64(histBins))
	}
	if metric := pp.getStringParam(params, "neighbourhood_metric", "mean"); metric != pp.neighMetric.Selected {
		pp.neighMetric.SetSelected(metric)
	}
	if weight := pp.getFloatParam(params, "pixel_weight_factor", 0.5); weight != pp.pixelWeightSlider.Value {
		pp.pixelWeightSlider.SetValue(weight)
	}
	if sigma := pp.getFloatParam(params, "smoothing_sigma", 1.0); sigma != pp.smoothingSigmaSlider.Value {
		pp.smoothingSigmaSlider.SetValue(sigma)
	}
	if useLog := pp.getBoolParam(params, "use_log_histogram", false); useLog != pp.useLogCheck.Checked {
		pp.useLogCheck.SetChecked(useLog)
	}
	if normalize := pp.getBoolParam(params, "normalize_histogram", true); normalize != pp.normalizeCheck.Checked {
		pp.normalizeCheck.SetChecked(normalize)
	}
	if contrast := pp.getBoolParam(params, "apply_contrast_enhancement", false); contrast != pp.contrastCheck.Checked {
		pp.contrastCheck.SetChecked(contrast)
	}
}

func (pp *ParameterPanel) updateTriclassValues(params map[string]interface{}) {
	if method := pp.getStringParam(params, "initial_threshold_method", "otsu"); method != pp.initialMethod.Selected {
		pp.initialMethod.SetSelected(method)
	}
	if maxIter := pp.getIntParam(params, "max_iterations", 10); maxIter != int(pp.maxIterSlider.Value) {
		pp.maxIterSlider.SetValue(float64(maxIter))
	}
	if epsilon := pp.getFloatParam(params, "convergence_epsilon", 1.0); epsilon != pp.convEpsilonSlider.Value {
		pp.convEpsilonSlider.SetValue(epsilon)
	}
	if minTBD := pp.getFloatParam(params, "minimum_tbd_fraction", 0.01); minTBD != pp.minTBDSlider.Value {
		pp.minTBDSlider.SetValue(minTBD)
	}
	if gap := pp.getFloatParam(params, "lower_upper_gap_factor", 0.5); gap != pp.gapFactorSlider.Value {
		pp.gapFactorSlider.SetValue(gap)
	}
	if preprocess := pp.getBoolParam(params, "apply_preprocessing", false); preprocess != pp.preprocessCheck.Checked {
		pp.preprocessCheck.SetChecked(preprocess)
	}
	if cleanup := pp.getBoolParam(params, "apply_cleanup", true); cleanup != pp.cleanupCheck.Checked {
		pp.cleanupCheck.SetChecked(cleanup)
	}
	if borders := pp.getBoolParam(params, "preserve_borders", false); borders != pp.bordersCheck.Checked {
		pp.bordersCheck.SetChecked(borders)
	}
}

func (pp *ParameterPanel) buildOtsu2DParameters(params map[string]interface{}) {
	windowSize := pp.getIntParam(params, "window_size", 7)
	pp.windowSizeSlider.SetValue(float64(windowSize))
	pp.windowSizeLabel.SetText("Window Size: " + strconv.Itoa(windowSize))

	histBins := pp.getIntParam(params, "histogram_bins", 64)
	pp.histBinsSlider.SetValue(float64(histBins))
	pp.histBinsLabel.SetText("Histogram Bins: " + strconv.Itoa(histBins))

	pp.neighMetric.SetSelected(pp.getStringParam(params, "neighbourhood_metric", "mean"))

	pixelWeight := pp.getFloatParam(params, "pixel_weight_factor", 0.5)
	pp.pixelWeightSlider.SetValue(pixelWeight)
	pp.pixelWeightLabel.SetText("Pixel Weight: " + strconv.FormatFloat(pixelWeight, 'f', 2, 64))

	smoothingSigma := pp.getFloatParam(params, "smoothing_sigma", 1.0)
	pp.smoothingSigmaSlider.SetValue(smoothingSigma)
	pp.smoothingSigmaLabel.SetText("Smoothing Sigma: " + strconv.FormatFloat(smoothingSigma, 'f', 1, 64))

	pp.useLogCheck.SetChecked(pp.getBoolParam(params, "use_log_histogram", false))
	pp.normalizeCheck.SetChecked(pp.getBoolParam(params, "normalize_histogram", true))
	pp.contrastCheck.SetChecked(pp.getBoolParam(params, "apply_contrast_enhancement", false))

	pp.parametersContent.Add(container.NewVBox(
		container.NewHBox(
			container.NewVBox(pp.windowSizeLabel, pp.windowSizeSlider),
			container.NewVBox(pp.histBinsLabel, pp.histBinsSlider),
			container.NewVBox(widget.NewLabel("Neighbourhood Metric"), pp.neighMetric),
		),
		container.NewHBox(
			container.NewVBox(pp.pixelWeightLabel, pp.pixelWeightSlider),
			container.NewVBox(pp.smoothingSigmaLabel, pp.smoothingSigmaSlider),
		),
		container.NewHBox(pp.useLogCheck, pp.normalizeCheck, pp.contrastCheck),
	))
}

func (pp *ParameterPanel) buildTriclassParameters(params map[string]interface{}) {
	pp.initialMethod.SetSelected(pp.getStringParam(params, "initial_threshold_method", "otsu"))

	maxIter := pp.getIntParam(params, "max_iterations", 10)
	pp.maxIterSlider.SetValue(float64(maxIter))
	pp.maxIterLabel.SetText("Max Iterations: " + strconv.Itoa(maxIter))

	convEpsilon := pp.getFloatParam(params, "convergence_epsilon", 1.0)
	pp.convEpsilonSlider.SetValue(convEpsilon)
	pp.convEpsilonLabel.SetText("Convergence Epsilon: " + strconv.FormatFloat(convEpsilon, 'f', 1, 64))

	minTBD := pp.getFloatParam(params, "minimum_tbd_fraction", 0.01)
	pp.minTBDSlider.SetValue(minTBD)
	pp.minTBDLabel.SetText("Min TBD Fraction: " + strconv.FormatFloat(minTBD, 'f', 3, 64))

	gapFactor := pp.getFloatParam(params, "lower_upper_gap_factor", 0.5)
	pp.gapFactorSlider.SetValue(gapFactor)
	pp.gapFactorLabel.SetText("Gap Factor: " + strconv.FormatFloat(gapFactor, 'f', 2, 64))

	pp.preprocessCheck.SetChecked(pp.getBoolParam(params, "apply_preprocessing", false))
	pp.cleanupCheck.SetChecked(pp.getBoolParam(params, "apply_cleanup", true))
	pp.bordersCheck.SetChecked(pp.getBoolParam(params, "preserve_borders", false))

	pp.parametersContent.Add(container.NewVBox(
		container.NewHBox(
			container.NewVBox(widget.NewLabel("Initial Method"), pp.initialMethod),
			container.NewVBox(pp.maxIterLabel, pp.maxIterSlider),
			container.NewVBox(pp.convEpsilonLabel, pp.convEpsilonSlider),
		),
		container.NewHBox(
			container.NewVBox(pp.minTBDLabel, pp.minTBDSlider),
			container.NewVBox(pp.gapFactorLabel, pp.gapFactorSlider),
		),
		container.NewHBox(pp.preprocessCheck, pp.cleanupCheck, pp.bordersCheck),
	))
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
