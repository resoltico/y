package components

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type ParameterPanel struct {
	container           *fyne.Container
	parametersContainer *fyne.Container
	currentWidgets      map[string]fyne.CanvasObject
	onParameterChange   func(string, interface{})
}

func NewParameterPanel() *ParameterPanel {
	panel := &ParameterPanel{
		currentWidgets: make(map[string]fyne.CanvasObject),
	}
	panel.setupPanel()
	return panel
}

func (pp *ParameterPanel) setupPanel() {
	parametersLabel := widget.NewLabel("Parameters")
	pp.parametersContainer = container.NewVBox()
	
	pp.container = container.NewVBox(
		parametersLabel,
		pp.parametersContainer,
	)
}

func (pp *ParameterPanel) GetContainer() *fyne.Container {
	return pp.container
}

func (pp *ParameterPanel) SetParameterChangeHandler(handler func(string, interface{})) {
	pp.onParameterChange = handler
}

func (pp *ParameterPanel) UpdateParameters(algorithm string, params map[string]interface{}) {
	pp.parametersContainer.RemoveAll()
	pp.currentWidgets = make(map[string]fyne.CanvasObject)

	switch algorithm {
	case "2D Otsu":
		pp.create2DOtsuParameters(params)
	case "Iterative Triclass":
		pp.createIterativeTriclassParameters(params)
	}

	pp.parametersContainer.Refresh()
}

func (pp *ParameterPanel) create2DOtsuParameters(params map[string]interface{}) {
	qualityRadio := widget.NewRadioGroup([]string{"Fast", "Best"}, func(value string) {
		pp.onParameterChange("quality", value)
	})
	qualityRadio.SetSelected(pp.getStringParam(params, "quality", "Fast"))
	pp.addParameter("Quality", qualityRadio)

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
		pp.onParameterChange("window_size", intValue)
	}
	pp.addParameterWithLabel("Window Size", windowSizeSlider, windowSizeLabel)

	histBins := pp.getIntParam(params, "histogram_bins", 64)
	histBinsSlider := widget.NewSlider(16, 256)
	histBinsSlider.SetValue(float64(histBins))
	histBinsLabel := widget.NewLabel("Histogram Bins: " + strconv.Itoa(histBins))
	histBinsSlider.OnChanged = func(value float64) {
		intValue := int(value)
		histBinsLabel.SetText("Histogram Bins: " + strconv.Itoa(intValue))
		pp.onParameterChange("histogram_bins", intValue)
	}
	pp.addParameterWithLabel("Histogram Bins", histBinsSlider, histBinsLabel)

	neighMetric := widget.NewSelect([]string{"mean", "median", "gaussian"}, func(value string) {
		pp.onParameterChange("neighbourhood_metric", value)
	})
	neighMetric.SetSelected(pp.getStringParam(params, "neighbourhood_metric", "mean"))
	pp.addParameter("Neighbourhood Metric", neighMetric)

	pixelWeight := pp.getFloatParam(params, "pixel_weight_factor", 0.5)
	pixelWeightSlider := widget.NewSlider(0.0, 1.0)
	pixelWeightSlider.SetValue(pixelWeight)
	pixelWeightLabel := widget.NewLabel("Pixel Weight: " + strconv.FormatFloat(pixelWeight, 'f', 2, 64))
	pixelWeightSlider.OnChanged = func(value float64) {
		pixelWeightLabel.SetText("Pixel Weight: " + strconv.FormatFloat(value, 'f', 2, 64))
		pp.onParameterChange("pixel_weight_factor", value)
	}
	pp.addParameterWithLabel("Pixel Weight Factor", pixelWeightSlider, pixelWeightLabel)

	pp.addCheckbox("Use Log Histogram", "use_log_histogram", params)
	pp.addCheckbox("Normalize Histogram", "normalize_histogram", params)
	pp.addCheckbox("Apply Contrast Enhancement", "apply_contrast_enhancement", params)
}

func (pp *ParameterPanel) createIterativeTriclassParameters(params map[string]interface{}) {
	qualityRadio := widget.NewRadioGroup([]string{"Fast", "Best"}, func(value string) {
		pp.onParameterChange("quality", value)
	})
	qualityRadio.SetSelected(pp.getStringParam(params, "quality", "Fast"))
	pp.addParameter("Quality", qualityRadio)

	initialMethod := widget.NewSelect([]string{"otsu", "mean", "median"}, func(value string) {
		pp.onParameterChange("initial_threshold_method", value)
	})
	initialMethod.SetSelected(pp.getStringParam(params, "initial_threshold_method", "otsu"))
	pp.addParameter("Initial Threshold Method", initialMethod)

	convEpsilon := pp.getFloatParam(params, "convergence_epsilon", 1.0)
	convEpsilonSlider := widget.NewSlider(0.1, 10.0)
	convEpsilonSlider.SetValue(convEpsilon)
	convEpsilonLabel := widget.NewLabel("Convergence Epsilon: " + strconv.FormatFloat(convEpsilon, 'f', 1, 64))
	convEpsilonSlider.OnChanged = func(value float64) {
		convEpsilonLabel.SetText("Convergence Epsilon: " + strconv.FormatFloat(value, 'f', 1, 64))
		pp.onParameterChange("convergence_epsilon", value)
	}
	pp.addParameterWithLabel("Convergence Epsilon", convEpsilonSlider, convEpsilonLabel)

	maxIter := pp.getIntParam(params, "max_iterations", 10)
	maxIterSlider := widget.NewSlider(1, 20)
	maxIterSlider.SetValue(float64(maxIter))
	maxIterLabel := widget.NewLabel("Max Iterations: " + strconv.Itoa(maxIter))
	maxIterSlider.OnChanged = func(value float64) {
		intValue := int(value)
		maxIterLabel.SetText("Max Iterations: " + strconv.Itoa(intValue))
		pp.onParameterChange("max_iterations", intValue)
	}
	pp.addParameterWithLabel("Max Iterations", maxIterSlider, maxIterLabel)

	pp.addCheckbox("Apply Preprocessing", "apply_preprocessing", params)
	pp.addCheckbox("Apply Cleanup", "apply_cleanup", params)
	pp.addCheckbox("Preserve Borders", "preserve_borders", params)
}

func (pp *ParameterPanel) addParameter(label string, obj fyne.CanvasObject) {
	paramLabel := widget.NewLabel(label)
	paramContainer := container.NewVBox(paramLabel, obj)
	pp.parametersContainer.Add(paramContainer)
	pp.currentWidgets[label] = obj
}

func (pp *ParameterPanel) addParameterWithLabel(label string, slider *widget.Slider, valueLabel *widget.Label) {
	paramLabel := widget.NewLabel(label)
	paramContainer := container.NewVBox(paramLabel, valueLabel, slider)
	pp.parametersContainer.Add(paramContainer)
	pp.currentWidgets[label] = slider
}

func (pp *ParameterPanel) addCheckbox(label, paramName string, params map[string]interface{}) {
	checkbox := widget.NewCheck(label, func(checked bool) {
		pp.onParameterChange(paramName, checked)
	})

	if value, ok := params[paramName].(bool); ok {
		checkbox.SetChecked(value)
	}

	pp.parametersContainer.Add(checkbox)
	pp.currentWidgets[label] = checkbox
}

func (pp *ParameterPanel) getIntParam(params map[string]interface{}, key string, defaultValue int) int {
	if value, ok := params[key].(int); ok {
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

func (pp *ParameterPanel) getStringParam(params map[string]interface{}, key string, defaultValue string) string {
	if value, ok := params[key].(string); ok {
		return value
	}
	return defaultValue
}