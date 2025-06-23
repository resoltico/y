package gui

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type ParameterPanel struct {
	container           *fyne.Container
	parametersContainer *fyne.Container

	onAlgorithmChange func(string)
	onParameterChange func(string, interface{})
	onGenerate        func()

	currentWidgets map[string]fyne.CanvasObject
}

func NewParameterPanel(onAlgorithmChange func(string), onParameterChange func(string, interface{}), onGenerate func()) *ParameterPanel {
	panel := &ParameterPanel{
		onAlgorithmChange: onAlgorithmChange,
		onParameterChange: onParameterChange,
		onGenerate:        onGenerate,
		currentWidgets:    make(map[string]fyne.CanvasObject),
	}

	panel.setupPanel()
	return panel
}

func (panel *ParameterPanel) setupPanel() {
	parametersLabel := widget.NewLabel("Parameters")
	panel.parametersContainer = container.NewVBox()

	panel.container = container.NewVBox(
		parametersLabel,
		panel.parametersContainer,
	)
}

func (panel *ParameterPanel) Initialize() {
	// Ready for parameter updates
}

func (panel *ParameterPanel) UpdateParameters(algorithm string, params map[string]interface{}) {
	panel.parametersContainer.RemoveAll()
	panel.currentWidgets = make(map[string]fyne.CanvasObject)

	switch algorithm {
	case "2D Otsu":
		panel.create2DOtsuParameters(params)
	case "Iterative Triclass":
		panel.createIterativeTriclassParameters(params)
	}

	panel.parametersContainer.Refresh()
}

func (panel *ParameterPanel) create2DOtsuParameters(params map[string]interface{}) {
	// Quality selector
	qualityRadio := widget.NewRadioGroup([]string{"Fast", "Best"}, func(value string) {
		panel.onParameterChange("quality", value)
	})
	if quality, ok := params["quality"].(string); ok {
		qualityRadio.SetSelected(quality)
	} else {
		qualityRadio.SetSelected("Fast")
	}
	panel.addParameter("Quality", qualityRadio)

	// Window size (3-21, odd only)
	windowSize := panel.getIntParam(params, "window_size", 7)
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
		panel.onParameterChange("window_size", intValue)
	}
	panel.addParameterWithLabel("Window Size", windowSizeSlider, windowSizeLabel)

	// Histogram bins (16-256)
	histBins := panel.getIntParam(params, "histogram_bins", 64)
	histBinsSlider := widget.NewSlider(16, 256)
	histBinsSlider.SetValue(float64(histBins))
	histBinsLabel := widget.NewLabel("Histogram Bins: " + strconv.Itoa(histBins))
	histBinsSlider.OnChanged = func(value float64) {
		intValue := int(value)
		histBinsLabel.SetText("Histogram Bins: " + strconv.Itoa(intValue))
		panel.onParameterChange("histogram_bins", intValue)
	}
	panel.addParameterWithLabel("Histogram Bins", histBinsSlider, histBinsLabel)

	// Neighbourhood metric
	neighMetric := widget.NewSelect([]string{"mean", "median", "gaussian"}, func(value string) {
		panel.onParameterChange("neighbourhood_metric", value)
	})
	if metric, ok := params["neighbourhood_metric"].(string); ok {
		neighMetric.SetSelected(metric)
	} else {
		neighMetric.SetSelected("mean")
	}
	panel.addParameter("Neighbourhood Metric", neighMetric)

	// Pixel weight factor (0.0-1.0)
	pixelWeight := panel.getFloatParam(params, "pixel_weight_factor", 0.5)
	pixelWeightSlider := widget.NewSlider(0.0, 1.0)
	pixelWeightSlider.SetValue(pixelWeight)
	pixelWeightLabel := widget.NewLabel("Pixel Weight: " + strconv.FormatFloat(pixelWeight, 'f', 2, 64))
	pixelWeightSlider.OnChanged = func(value float64) {
		pixelWeightLabel.SetText("Pixel Weight: " + strconv.FormatFloat(value, 'f', 2, 64))
		panel.onParameterChange("pixel_weight_factor", value)
	}
	panel.addParameterWithLabel("Pixel Weight Factor", pixelWeightSlider, pixelWeightLabel)

	// Checkboxes
	panel.addCheckbox("Use Log Histogram", "use_log_histogram", params)
	panel.addCheckbox("Normalize Histogram", "normalize_histogram", params)
	panel.addCheckbox("Apply Contrast Enhancement", "apply_contrast_enhancement", params)
}

func (panel *ParameterPanel) createIterativeTriclassParameters(params map[string]interface{}) {
	// Quality selector
	qualityRadio := widget.NewRadioGroup([]string{"Fast", "Best"}, func(value string) {
		panel.onParameterChange("quality", value)
	})
	if quality, ok := params["quality"].(string); ok {
		qualityRadio.SetSelected(quality)
	} else {
		qualityRadio.SetSelected("Fast")
	}
	panel.addParameter("Quality", qualityRadio)

	// Initial threshold method
	initialMethod := widget.NewSelect([]string{"otsu", "mean", "median"}, func(value string) {
		panel.onParameterChange("initial_threshold_method", value)
	})
	if method, ok := params["initial_threshold_method"].(string); ok {
		initialMethod.SetSelected(method)
	} else {
		initialMethod.SetSelected("otsu")
	}
	panel.addParameter("Initial Threshold Method", initialMethod)

	// Convergence epsilon (0.1-10.0)
	convEpsilon := panel.getFloatParam(params, "convergence_epsilon", 1.0)
	convEpsilonSlider := widget.NewSlider(0.1, 10.0)
	convEpsilonSlider.SetValue(convEpsilon)
	convEpsilonLabel := widget.NewLabel("Convergence Epsilon: " + strconv.FormatFloat(convEpsilon, 'f', 1, 64))
	convEpsilonSlider.OnChanged = func(value float64) {
		convEpsilonLabel.SetText("Convergence Epsilon: " + strconv.FormatFloat(value, 'f', 1, 64))
		panel.onParameterChange("convergence_epsilon", value)
	}
	panel.addParameterWithLabel("Convergence Epsilon", convEpsilonSlider, convEpsilonLabel)

	// Max iterations (1-20)
	maxIter := panel.getIntParam(params, "max_iterations", 10)
	maxIterSlider := widget.NewSlider(1, 20)
	maxIterSlider.SetValue(float64(maxIter))
	maxIterLabel := widget.NewLabel("Max Iterations: " + strconv.Itoa(maxIter))
	maxIterSlider.OnChanged = func(value float64) {
		intValue := int(value)
		maxIterLabel.SetText("Max Iterations: " + strconv.Itoa(intValue))
		panel.onParameterChange("max_iterations", intValue)
	}
	panel.addParameterWithLabel("Max Iterations", maxIterSlider, maxIterLabel)

	// Checkboxes
	panel.addCheckbox("Apply Preprocessing", "apply_preprocessing", params)
	panel.addCheckbox("Apply Cleanup", "apply_cleanup", params)
	panel.addCheckbox("Preserve Borders", "preserve_borders", params)
}

func (panel *ParameterPanel) addParameter(label string, obj fyne.CanvasObject) {
	paramLabel := widget.NewLabel(label)
	paramContainer := container.NewVBox(paramLabel, obj)
	panel.parametersContainer.Add(paramContainer)
	panel.currentWidgets[label] = obj
}

func (panel *ParameterPanel) addParameterWithLabel(label string, slider *widget.Slider, valueLabel *widget.Label) {
	paramLabel := widget.NewLabel(label)
	paramContainer := container.NewVBox(paramLabel, valueLabel, slider)
	panel.parametersContainer.Add(paramContainer)
	panel.currentWidgets[label] = slider
}

func (panel *ParameterPanel) addCheckbox(label, paramName string, params map[string]interface{}) {
	checkbox := widget.NewCheck(label, func(checked bool) {
		panel.onParameterChange(paramName, checked)
	})

	if value, ok := params[paramName].(bool); ok {
		checkbox.SetChecked(value)
	}

	panel.parametersContainer.Add(checkbox)
	panel.currentWidgets[label] = checkbox
}

func (panel *ParameterPanel) getIntParam(params map[string]interface{}, key string, defaultValue int) int {
	if value, ok := params[key].(int); ok {
		return value
	}
	return defaultValue
}

func (panel *ParameterPanel) getFloatParam(params map[string]interface{}, key string, defaultValue float64) float64 {
	if value, ok := params[key].(float64); ok {
		return value
	}
	return defaultValue
}

func (panel *ParameterPanel) GetContainer() *fyne.Container {
	return panel.container
}
