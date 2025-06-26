package components

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type ParametersSection struct {
	container         *fyne.Container
	parametersContent *fyne.Container
	onParameterChange func(string, interface{})
}

func NewParametersSection() *ParametersSection {
	section := &ParametersSection{}
	section.setupSection()
	return section
}

func (ps *ParametersSection) setupSection() {
	ps.parametersContent = container.NewVBox(
		widget.NewLabel("Parameters:"),
	)

	ps.container = container.NewVBox(ps.parametersContent)
}

func (ps *ParametersSection) GetContainer() *fyne.Container {
	return ps.container
}

func (ps *ParametersSection) SetParameterChangeHandler(handler func(string, interface{})) {
	ps.onParameterChange = handler
}

func (ps *ParametersSection) UpdateParameters(algorithm string, params map[string]interface{}) {
	ps.parametersContent.RemoveAll()
	ps.parametersContent.Add(widget.NewLabel("Parameters:"))

	switch algorithm {
	case "2D Otsu":
		ps.create2DOtsuParameters(params)
	case "Iterative Triclass":
		ps.createIterativeTriclassParameters(params)
	}

	ps.container.Refresh()
}

func (ps *ParametersSection) create2DOtsuParameters(params map[string]interface{}) {
	// Row 1: Window Size, Histogram Bins, Neighbourhood Metric
	windowSize := ps.getIntParam(params, "window_size", 7)
	windowSizeSlider := widget.NewSlider(3, 21)
	windowSizeSlider.Step = 2
	windowSizeSlider.SetValue(float64(windowSize))
	windowSizeLabel := widget.NewLabel("Window Size: " + strconv.Itoa(windowSize))
	windowSizeSlider.OnChanged = func(value float64) {
		intValue := int(value)
		if intValue%2 == 0 {
			intValue++
		}
		windowSizeLabel.SetText("Window Size: " + strconv.Itoa(intValue))
		if ps.onParameterChange != nil {
			ps.onParameterChange("window_size", intValue)
		}
	}

	histBins := ps.getIntParam(params, "histogram_bins", 64)
	histBinsSlider := widget.NewSlider(16, 256)
	histBinsSlider.SetValue(float64(histBins))
	histBinsLabel := widget.NewLabel("Histogram Bins: " + strconv.Itoa(histBins))
	histBinsSlider.OnChanged = func(value float64) {
		intValue := int(value)
		histBinsLabel.SetText("Histogram Bins: " + strconv.Itoa(intValue))
		if ps.onParameterChange != nil {
			ps.onParameterChange("histogram_bins", intValue)
		}
	}

	neighMetric := widget.NewSelect([]string{"mean", "median", "gaussian"}, func(value string) {
		if ps.onParameterChange != nil {
			ps.onParameterChange("neighbourhood_metric", value)
		}
	})
	neighMetric.SetSelected(ps.getStringParam(params, "neighbourhood_metric", "mean"))

	// Row 2: Pixel Weight, Smoothing Sigma
	pixelWeight := ps.getFloatParam(params, "pixel_weight_factor", 0.5)
	pixelWeightSlider := widget.NewSlider(0.0, 1.0)
	pixelWeightSlider.SetValue(pixelWeight)
	pixelWeightLabel := widget.NewLabel("Pixel Weight: " + strconv.FormatFloat(pixelWeight, 'f', 2, 64))
	pixelWeightSlider.OnChanged = func(value float64) {
		pixelWeightLabel.SetText("Pixel Weight: " + strconv.FormatFloat(value, 'f', 2, 64))
		if ps.onParameterChange != nil {
			ps.onParameterChange("pixel_weight_factor", value)
		}
	}

	smoothingSigma := ps.getFloatParam(params, "smoothing_sigma", 1.0)
	smoothingSigmaSlider := widget.NewSlider(0.0, 5.0)
	smoothingSigmaSlider.SetValue(smoothingSigma)
	smoothingSigmaLabel := widget.NewLabel("Smoothing Sigma: " + strconv.FormatFloat(smoothingSigma, 'f', 1, 64))
	smoothingSigmaSlider.OnChanged = func(value float64) {
		smoothingSigmaLabel.SetText("Smoothing Sigma: " + strconv.FormatFloat(value, 'f', 1, 64))
		if ps.onParameterChange != nil {
			ps.onParameterChange("smoothing_sigma", value)
		}
	}

	// Row 3: Checkboxes
	useLogCheck := widget.NewCheck("Use Log Histogram", func(checked bool) {
		if ps.onParameterChange != nil {
			ps.onParameterChange("use_log_histogram", checked)
		}
	})
	useLogCheck.SetChecked(ps.getBoolParam(params, "use_log_histogram", false))

	normalizeCheck := widget.NewCheck("Normalize Histogram", func(checked bool) {
		if ps.onParameterChange != nil {
			ps.onParameterChange("normalize_histogram", checked)
		}
	})
	normalizeCheck.SetChecked(ps.getBoolParam(params, "normalize_histogram", true))

	contrastCheck := widget.NewCheck("Apply Contrast Enhancement", func(checked bool) {
		if ps.onParameterChange != nil {
			ps.onParameterChange("apply_contrast_enhancement", checked)
		}
	})
	contrastCheck.SetChecked(ps.getBoolParam(params, "apply_contrast_enhancement", false))

	ps.parametersContent.Add(container.NewVBox(
		container.NewHBox(
			container.NewVBox(windowSizeLabel, windowSizeSlider),
			container.NewVBox(histBinsLabel, histBinsSlider),
			container.NewVBox(widget.NewLabel("Neighbourhood Metric"), neighMetric),
		),
		container.NewHBox(
			container.NewVBox(pixelWeightLabel, pixelWeightSlider),
			container.NewVBox(smoothingSigmaLabel, smoothingSigmaSlider),
		),
		container.NewHBox(useLogCheck, normalizeCheck, contrastCheck),
	))
}

func (ps *ParametersSection) createIterativeTriclassParameters(params map[string]interface{}) {
	// Row 1: Initial Method, Max Iterations, Convergence Epsilon
	initialMethod := widget.NewSelect([]string{"otsu", "mean", "median"}, func(value string) {
		if ps.onParameterChange != nil {
			ps.onParameterChange("initial_threshold_method", value)
		}
	})
	initialMethod.SetSelected(ps.getStringParam(params, "initial_threshold_method", "otsu"))

	maxIter := ps.getIntParam(params, "max_iterations", 10)
	maxIterSlider := widget.NewSlider(1, 20)
	maxIterSlider.SetValue(float64(maxIter))
	maxIterLabel := widget.NewLabel("Max Iterations: " + strconv.Itoa(maxIter))
	maxIterSlider.OnChanged = func(value float64) {
		intValue := int(value)
		maxIterLabel.SetText("Max Iterations: " + strconv.Itoa(intValue))
		if ps.onParameterChange != nil {
			ps.onParameterChange("max_iterations", intValue)
		}
	}

	convEpsilon := ps.getFloatParam(params, "convergence_epsilon", 1.0)
	convEpsilonSlider := widget.NewSlider(0.1, 10.0)
	convEpsilonSlider.SetValue(convEpsilon)
	convEpsilonLabel := widget.NewLabel("Convergence Epsilon: " + strconv.FormatFloat(convEpsilon, 'f', 1, 64))
	convEpsilonSlider.OnChanged = func(value float64) {
		convEpsilonLabel.SetText("Convergence Epsilon: " + strconv.FormatFloat(value, 'f', 1, 64))
		if ps.onParameterChange != nil {
			ps.onParameterChange("convergence_epsilon", value)
		}
	}

	// Row 2: Min TBD Fraction, Gap Factor
	minTBD := ps.getFloatParam(params, "minimum_tbd_fraction", 0.01)
	minTBDSlider := widget.NewSlider(0.001, 0.2)
	minTBDSlider.SetValue(minTBD)
	minTBDLabel := widget.NewLabel("Min TBD Fraction: " + strconv.FormatFloat(minTBD, 'f', 3, 64))
	minTBDSlider.OnChanged = func(value float64) {
		minTBDLabel.SetText("Min TBD Fraction: " + strconv.FormatFloat(value, 'f', 3, 64))
		if ps.onParameterChange != nil {
			ps.onParameterChange("minimum_tbd_fraction", value)
		}
	}

	gapFactor := ps.getFloatParam(params, "lower_upper_gap_factor", 0.5)
	gapFactorSlider := widget.NewSlider(0.0, 1.0)
	gapFactorSlider.SetValue(gapFactor)
	gapFactorLabel := widget.NewLabel("Gap Factor: " + strconv.FormatFloat(gapFactor, 'f', 2, 64))
	gapFactorSlider.OnChanged = func(value float64) {
		gapFactorLabel.SetText("Gap Factor: " + strconv.FormatFloat(value, 'f', 2, 64))
		if ps.onParameterChange != nil {
			ps.onParameterChange("lower_upper_gap_factor", value)
		}
	}

	// Row 3: Checkboxes
	preprocessCheck := widget.NewCheck("Apply Preprocessing", func(checked bool) {
		if ps.onParameterChange != nil {
			ps.onParameterChange("apply_preprocessing", checked)
		}
	})
	preprocessCheck.SetChecked(ps.getBoolParam(params, "apply_preprocessing", false))

	cleanupCheck := widget.NewCheck("Apply Cleanup", func(checked bool) {
		if ps.onParameterChange != nil {
			ps.onParameterChange("apply_cleanup", checked)
		}
	})
	cleanupCheck.SetChecked(ps.getBoolParam(params, "apply_cleanup", true))

	bordersCheck := widget.NewCheck("Preserve Borders", func(checked bool) {
		if ps.onParameterChange != nil {
			ps.onParameterChange("preserve_borders", checked)
		}
	})
	bordersCheck.SetChecked(ps.getBoolParam(params, "preserve_borders", false))

	ps.parametersContent.Add(container.NewVBox(
		container.NewHBox(
			container.NewVBox(widget.NewLabel("Initial Method"), initialMethod),
			container.NewVBox(maxIterLabel, maxIterSlider),
			container.NewVBox(convEpsilonLabel, convEpsilonSlider),
		),
		container.NewHBox(
			container.NewVBox(minTBDLabel, minTBDSlider),
			container.NewVBox(gapFactorLabel, gapFactorSlider),
		),
		container.NewHBox(preprocessCheck, cleanupCheck, bordersCheck),
	))
}

func (ps *ParametersSection) getIntParam(params map[string]interface{}, key string, defaultValue int) int {
	if value, ok := params[key].(int); ok {
		return value
	}
	return defaultValue
}

func (ps *ParametersSection) getStringParam(params map[string]interface{}, key string, defaultValue string) string {
	if value, ok := params[key].(string); ok {
		return value
	}
	return defaultValue
}

func (ps *ParametersSection) getFloatParam(params map[string]interface{}, key string, defaultValue float64) float64 {
	if value, ok := params[key].(float64); ok {
		return value
	}
	return defaultValue
}

func (ps *ParametersSection) getBoolParam(params map[string]interface{}, key string, defaultValue bool) bool {
	if value, ok := params[key].(bool); ok {
		return value
	}
	return defaultValue
}
