package components

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type ParametersPanel struct {
	container         *fyne.Container
	qualitySection    *fyne.Container
	parametersSection *fyne.Container
	onParameterChange func(string, interface{})
}

func NewParametersPanel() *ParametersPanel {
	panel := &ParametersPanel{}
	panel.setupPanel()
	return panel
}

func (pp *ParametersPanel) setupPanel() {
	pp.qualitySection = container.NewVBox()
	pp.parametersSection = container.NewVBox()

	pp.container = container.NewVBox(
		widget.NewCard("Quality", "", pp.qualitySection),
		widget.NewCard("Parameters", "", pp.parametersSection),
	)
}

func (pp *ParametersPanel) GetContainer() *fyne.Container {
	return pp.container
}

func (pp *ParametersPanel) SetParameterChangeHandler(handler func(string, interface{})) {
	pp.onParameterChange = handler
}

func (pp *ParametersPanel) UpdateParameters(algorithm string, params map[string]interface{}) {
	pp.qualitySection.RemoveAll()
	pp.parametersSection.RemoveAll()

	qualityRadio := widget.NewRadioGroup([]string{"Fast", "Best"}, func(value string) {
		if pp.onParameterChange != nil {
			pp.onParameterChange("quality", value)
		}
	})
	qualityRadio.SetSelected(pp.getStringParam(params, "quality", "Fast"))
	pp.qualitySection.Add(qualityRadio)

	switch algorithm {
	case "2D Otsu":
		pp.create2DOtsuParameters(params)
	case "Iterative Triclass":
		pp.createIterativeTriclassParameters(params)
	}

	pp.container.Refresh()
}

func (pp *ParametersPanel) create2DOtsuParameters(params map[string]interface{}) {
	windowSize := pp.getIntParam(params, "window_size", 7)
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
		if pp.onParameterChange != nil {
			pp.onParameterChange("window_size", intValue)
		}
	}

	histBins := pp.getIntParam(params, "histogram_bins", 64)
	histBinsSlider := widget.NewSlider(16, 256)
	histBinsSlider.SetValue(float64(histBins))
	histBinsLabel := widget.NewLabel("Histogram Bins: " + strconv.Itoa(histBins))
	histBinsSlider.OnChanged = func(value float64) {
		intValue := int(value)
		histBinsLabel.SetText("Histogram Bins: " + strconv.Itoa(intValue))
		if pp.onParameterChange != nil {
			pp.onParameterChange("histogram_bins", intValue)
		}
	}

	neighMetric := widget.NewSelect([]string{"mean", "median", "gaussian"}, func(value string) {
		if pp.onParameterChange != nil {
			pp.onParameterChange("neighbourhood_metric", value)
		}
	})
	neighMetric.SetSelected(pp.getStringParam(params, "neighbourhood_metric", "mean"))

	pixelWeight := pp.getFloatParam(params, "pixel_weight_factor", 0.5)
	pixelWeightSlider := widget.NewSlider(0.0, 1.0)
	pixelWeightSlider.SetValue(pixelWeight)
	pixelWeightLabel := widget.NewLabel("Pixel Weight: " + strconv.FormatFloat(pixelWeight, 'f', 2, 64))
	pixelWeightSlider.OnChanged = func(value float64) {
		pixelWeightLabel.SetText("Pixel Weight: " + strconv.FormatFloat(value, 'f', 2, 64))
		if pp.onParameterChange != nil {
			pp.onParameterChange("pixel_weight_factor", value)
		}
	}

	pp.parametersSection.Add(container.NewVBox(
		windowSizeLabel,
		windowSizeSlider,
		histBinsLabel,
		histBinsSlider,
		widget.NewLabel("Neighbourhood Metric"),
		neighMetric,
		pixelWeightLabel,
		pixelWeightSlider,
		pp.createCheckbox("Use Log Histogram", "use_log_histogram", params),
		pp.createCheckbox("Normalize Histogram", "normalize_histogram", params),
		pp.createCheckbox("Apply Contrast Enhancement", "apply_contrast_enhancement", params),
	))
}

func (pp *ParametersPanel) createIterativeTriclassParameters(params map[string]interface{}) {
	initialMethod := widget.NewSelect([]string{"otsu", "mean", "median"}, func(value string) {
		if pp.onParameterChange != nil {
			pp.onParameterChange("initial_threshold_method", value)
		}
	})
	initialMethod.SetSelected(pp.getStringParam(params, "initial_threshold_method", "otsu"))

	convEpsilon := pp.getFloatParam(params, "convergence_epsilon", 1.0)
	convEpsilonSlider := widget.NewSlider(0.1, 10.0)
	convEpsilonSlider.SetValue(convEpsilon)
	convEpsilonLabel := widget.NewLabel("Convergence Epsilon: " + strconv.FormatFloat(convEpsilon, 'f', 1, 64))
	convEpsilonSlider.OnChanged = func(value float64) {
		convEpsilonLabel.SetText("Convergence Epsilon: " + strconv.FormatFloat(value, 'f', 1, 64))
		if pp.onParameterChange != nil {
			pp.onParameterChange("convergence_epsilon", value)
		}
	}

	maxIter := pp.getIntParam(params, "max_iterations", 10)
	maxIterSlider := widget.NewSlider(1, 20)
	maxIterSlider.SetValue(float64(maxIter))
	maxIterLabel := widget.NewLabel("Max Iterations: " + strconv.Itoa(maxIter))
	maxIterSlider.OnChanged = func(value float64) {
		intValue := int(value)
		maxIterLabel.SetText("Max Iterations: " + strconv.Itoa(intValue))
		if pp.onParameterChange != nil {
			pp.onParameterChange("max_iterations", intValue)
		}
	}

	pp.parametersSection.Add(container.NewVBox(
		widget.NewLabel("Initial Threshold Method"),
		initialMethod,
		convEpsilonLabel,
		convEpsilonSlider,
		maxIterLabel,
		maxIterSlider,
		pp.createCheckbox("Apply Preprocessing", "apply_preprocessing", params),
		pp.createCheckbox("Apply Cleanup", "apply_cleanup", params),
		pp.createCheckbox("Preserve Borders", "preserve_borders", params),
	))
}

func (pp *ParametersPanel) createCheckbox(label, paramName string, params map[string]interface{}) *widget.Check {
	checkbox := widget.NewCheck(label, func(checked bool) {
		if pp.onParameterChange != nil {
			pp.onParameterChange(paramName, checked)
		}
	})

	if value, ok := params[paramName].(bool); ok {
		checkbox.SetChecked(value)
	}

	return checkbox
}

func (pp *ParametersPanel) getIntParam(params map[string]interface{}, key string, defaultValue int) int {
	if value, ok := params[key].(int); ok {
		return value
	}
	return defaultValue
}

func (pp *ParametersPanel) getFloatParam(params map[string]interface{}, key string, defaultValue float64) float64 {
	if value, ok := params[key].(float64); ok {
		return value
	}
	return defaultValue
}

func (pp *ParametersPanel) getStringParam(params map[string]interface{}, key string, defaultValue string) string {
	if value, ok := params[key].(string); ok {
		return value
	}
	return defaultValue
}
