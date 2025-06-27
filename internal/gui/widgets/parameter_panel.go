package widgets

import (
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type ParameterPanel struct {
	container              *fyne.Container
	parametersContent      *fyne.Container
	parameterChangeHandler func(string, interface{})
}

func NewParameterPanel() *ParameterPanel {
	panel := &ParameterPanel{}
	panel.setupPanel()
	return panel
}

func (pp *ParameterPanel) setupPanel() {
	pp.parametersContent = container.NewVBox(
		widget.NewLabel("Parameters:"),
	)
	pp.container = container.NewVBox(pp.parametersContent)
}

func (pp *ParameterPanel) GetContainer() *fyne.Container {
	return pp.container
}

func (pp *ParameterPanel) SetParameterChangeHandler(handler func(string, interface{})) {
	pp.parameterChangeHandler = handler
}

func (pp *ParameterPanel) UpdateParameters(algorithm string, params map[string]interface{}) {
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

func (pp *ParameterPanel) buildOtsu2DParameters(params map[string]interface{}) {
	var builder strings.Builder

	windowSize := pp.getIntParam(params, "window_size", 7)
	windowSizeSlider := widget.NewSlider(3, 21)
	windowSizeSlider.Step = 2
	windowSizeSlider.SetValue(float64(windowSize))

	builder.Reset()
	builder.WriteString("Window Size: ")
	builder.WriteString(strconv.Itoa(windowSize))
	windowSizeLabel := widget.NewLabel(builder.String())

	windowSizeSlider.OnChanged = func(value float64) {
		intValue := int(value)
		if intValue%2 == 0 {
			intValue++
		}
		builder.Reset()
		builder.WriteString("Window Size: ")
		builder.WriteString(strconv.Itoa(intValue))
		windowSizeLabel.SetText(builder.String())

		if pp.parameterChangeHandler != nil {
			pp.parameterChangeHandler("window_size", intValue)
		}
	}

	histBins := pp.getIntParam(params, "histogram_bins", 64)
	histBinsSlider := widget.NewSlider(16, 256)
	histBinsSlider.SetValue(float64(histBins))

	builder.Reset()
	builder.WriteString("Histogram Bins: ")
	builder.WriteString(strconv.Itoa(histBins))
	histBinsLabel := widget.NewLabel(builder.String())

	histBinsSlider.OnChanged = func(value float64) {
		intValue := int(value)
		builder.Reset()
		builder.WriteString("Histogram Bins: ")
		builder.WriteString(strconv.Itoa(intValue))
		histBinsLabel.SetText(builder.String())

		if pp.parameterChangeHandler != nil {
			pp.parameterChangeHandler("histogram_bins", intValue)
		}
	}

	neighMetric := widget.NewSelect([]string{"mean", "median", "gaussian"}, func(value string) {
		if pp.parameterChangeHandler != nil {
			pp.parameterChangeHandler("neighbourhood_metric", value)
		}
	})
	neighMetric.SetSelected(pp.getStringParam(params, "neighbourhood_metric", "mean"))

	pixelWeight := pp.getFloatParam(params, "pixel_weight_factor", 0.5)
	pixelWeightSlider := widget.NewSlider(0.0, 1.0)
	pixelWeightSlider.SetValue(pixelWeight)

	builder.Reset()
	builder.WriteString("Pixel Weight: ")
	builder.WriteString(strconv.FormatFloat(pixelWeight, 'f', 2, 64))
	pixelWeightLabel := widget.NewLabel(builder.String())

	pixelWeightSlider.OnChanged = func(value float64) {
		builder.Reset()
		builder.WriteString("Pixel Weight: ")
		builder.WriteString(strconv.FormatFloat(value, 'f', 2, 64))
		pixelWeightLabel.SetText(builder.String())

		if pp.parameterChangeHandler != nil {
			pp.parameterChangeHandler("pixel_weight_factor", value)
		}
	}

	smoothingSigma := pp.getFloatParam(params, "smoothing_sigma", 1.0)
	smoothingSigmaSlider := widget.NewSlider(0.0, 5.0)
	smoothingSigmaSlider.SetValue(smoothingSigma)

	builder.Reset()
	builder.WriteString("Smoothing Sigma: ")
	builder.WriteString(strconv.FormatFloat(smoothingSigma, 'f', 1, 64))
	smoothingSigmaLabel := widget.NewLabel(builder.String())

	smoothingSigmaSlider.OnChanged = func(value float64) {
		builder.Reset()
		builder.WriteString("Smoothing Sigma: ")
		builder.WriteString(strconv.FormatFloat(value, 'f', 1, 64))
		smoothingSigmaLabel.SetText(builder.String())

		if pp.parameterChangeHandler != nil {
			pp.parameterChangeHandler("smoothing_sigma", value)
		}
	}

	useLogCheck := widget.NewCheck("Use Log Histogram", func(checked bool) {
		if pp.parameterChangeHandler != nil {
			pp.parameterChangeHandler("use_log_histogram", checked)
		}
	})
	useLogCheck.SetChecked(pp.getBoolParam(params, "use_log_histogram", false))

	normalizeCheck := widget.NewCheck("Normalize Histogram", func(checked bool) {
		if pp.parameterChangeHandler != nil {
			pp.parameterChangeHandler("normalize_histogram", checked)
		}
	})
	normalizeCheck.SetChecked(pp.getBoolParam(params, "normalize_histogram", true))

	contrastCheck := widget.NewCheck("Apply Contrast Enhancement", func(checked bool) {
		if pp.parameterChangeHandler != nil {
			pp.parameterChangeHandler("apply_contrast_enhancement", checked)
		}
	})
	contrastCheck.SetChecked(pp.getBoolParam(params, "apply_contrast_enhancement", false))

	pp.parametersContent.Add(container.NewVBox(
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

func (pp *ParameterPanel) buildTriclassParameters(params map[string]interface{}) {
	var builder strings.Builder

	initialMethod := widget.NewSelect([]string{"otsu", "mean", "median"}, func(value string) {
		if pp.parameterChangeHandler != nil {
			pp.parameterChangeHandler("initial_threshold_method", value)
		}
	})
	initialMethod.SetSelected(pp.getStringParam(params, "initial_threshold_method", "otsu"))

	maxIter := pp.getIntParam(params, "max_iterations", 10)
	maxIterSlider := widget.NewSlider(1, 20)
	maxIterSlider.SetValue(float64(maxIter))

	builder.Reset()
	builder.WriteString("Max Iterations: ")
	builder.WriteString(strconv.Itoa(maxIter))
	maxIterLabel := widget.NewLabel(builder.String())

	maxIterSlider.OnChanged = func(value float64) {
		intValue := int(value)
		builder.Reset()
		builder.WriteString("Max Iterations: ")
		builder.WriteString(strconv.Itoa(intValue))
		maxIterLabel.SetText(builder.String())

		if pp.parameterChangeHandler != nil {
			pp.parameterChangeHandler("max_iterations", intValue)
		}
	}

	convEpsilon := pp.getFloatParam(params, "convergence_epsilon", 1.0)
	convEpsilonSlider := widget.NewSlider(0.1, 10.0)
	convEpsilonSlider.SetValue(convEpsilon)

	builder.Reset()
	builder.WriteString("Convergence Epsilon: ")
	builder.WriteString(strconv.FormatFloat(convEpsilon, 'f', 1, 64))
	convEpsilonLabel := widget.NewLabel(builder.String())

	convEpsilonSlider.OnChanged = func(value float64) {
		builder.Reset()
		builder.WriteString("Convergence Epsilon: ")
		builder.WriteString(strconv.FormatFloat(value, 'f', 1, 64))
		convEpsilonLabel.SetText(builder.String())

		if pp.parameterChangeHandler != nil {
			pp.parameterChangeHandler("convergence_epsilon", value)
		}
	}

	minTBD := pp.getFloatParam(params, "minimum_tbd_fraction", 0.01)
	minTBDSlider := widget.NewSlider(0.001, 0.2)
	minTBDSlider.SetValue(minTBD)

	builder.Reset()
	builder.WriteString("Min TBD Fraction: ")
	builder.WriteString(strconv.FormatFloat(minTBD, 'f', 3, 64))
	minTBDLabel := widget.NewLabel(builder.String())

	minTBDSlider.OnChanged = func(value float64) {
		builder.Reset()
		builder.WriteString("Min TBD Fraction: ")
		builder.WriteString(strconv.FormatFloat(value, 'f', 3, 64))
		minTBDLabel.SetText(builder.String())

		if pp.parameterChangeHandler != nil {
			pp.parameterChangeHandler("minimum_tbd_fraction", value)
		}
	}

	gapFactor := pp.getFloatParam(params, "lower_upper_gap_factor", 0.5)
	gapFactorSlider := widget.NewSlider(0.0, 1.0)
	gapFactorSlider.SetValue(gapFactor)

	builder.Reset()
	builder.WriteString("Gap Factor: ")
	builder.WriteString(strconv.FormatFloat(gapFactor, 'f', 2, 64))
	gapFactorLabel := widget.NewLabel(builder.String())

	gapFactorSlider.OnChanged = func(value float64) {
		builder.Reset()
		builder.WriteString("Gap Factor: ")
		builder.WriteString(strconv.FormatFloat(value, 'f', 2, 64))
		gapFactorLabel.SetText(builder.String())

		if pp.parameterChangeHandler != nil {
			pp.parameterChangeHandler("lower_upper_gap_factor", value)
		}
	}

	preprocessCheck := widget.NewCheck("Apply Preprocessing", func(checked bool) {
		if pp.parameterChangeHandler != nil {
			pp.parameterChangeHandler("apply_preprocessing", checked)
		}
	})
	preprocessCheck.SetChecked(pp.getBoolParam(params, "apply_preprocessing", false))

	cleanupCheck := widget.NewCheck("Apply Cleanup", func(checked bool) {
		if pp.parameterChangeHandler != nil {
			pp.parameterChangeHandler("apply_cleanup", checked)
		}
	})
	cleanupCheck.SetChecked(pp.getBoolParam(params, "apply_cleanup", true))

	bordersCheck := widget.NewCheck("Preserve Borders", func(checked bool) {
		if pp.parameterChangeHandler != nil {
			pp.parameterChangeHandler("preserve_borders", checked)
		}
	})
	bordersCheck.SetChecked(pp.getBoolParam(params, "preserve_borders", false))

	pp.parametersContent.Add(container.NewVBox(
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
