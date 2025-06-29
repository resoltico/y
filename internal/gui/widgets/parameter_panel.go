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
	currentAlgorithm       string

	// Common parameter widgets that get reused
	parameterWidgets map[string]fyne.CanvasObject
}

func NewParameterPanel() *ParameterPanel {
	panel := &ParameterPanel{
		parameterWidgets: make(map[string]fyne.CanvasObject),
	}
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
	// Use fyne.Do for thread-safe UI updates
	fyne.Do(func() {
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
	})
}

func (pp *ParameterPanel) updateValues(params map[string]interface{}) {
	// Update widget values without rebuilding the entire panel
	for paramName, value := range params {
		if widgetObj, exists := pp.parameterWidgets[paramName]; exists {
			switch widget := widgetObj.(type) {
			case *widget.Slider:
				if floatVal, ok := value.(float64); ok {
					widget.SetValue(floatVal)
				} else if intVal, ok := value.(int); ok {
					widget.SetValue(float64(intVal))
				}
			case *widget.Check:
				if boolVal, ok := value.(bool); ok {
					widget.SetChecked(boolVal)
				}
			case *widget.Select:
				if strVal, ok := value.(string); ok {
					widget.SetSelected(strVal)
				}
			}
		}
	}
}

func (pp *ParameterPanel) buildOtsu2DParameters(params map[string]interface{}) {
	// Window Size
	windowSizeSlider := widget.NewSlider(3, 21)
	windowSizeSlider.Step = 2
	windowSizeLabel := widget.NewLabel("Window Size: 7")
	windowSize := pp.getIntParam(params, "window_size", 7)
	windowSizeSlider.SetValue(float64(windowSize))
	windowSizeLabel.SetText("Window Size: " + strconv.Itoa(windowSize))
	windowSizeSlider.OnChanged = func(value float64) {
		intValue := int(value)
		if intValue%2 == 0 {
			intValue++
		}
		windowSizeLabel.SetText("Window Size: " + strconv.Itoa(intValue))
		if pp.parameterChangeHandler != nil {
			pp.parameterChangeHandler("window_size", intValue)
		}
	}

	// Histogram Bins
	histBinsSlider := widget.NewSlider(0, 256)
	histBinsLabel := widget.NewLabel("Histogram Bins: Auto")
	histBins := pp.getIntParam(params, "histogram_bins", 0)
	histBinsSlider.SetValue(float64(histBins))
	if histBins == 0 {
		histBinsLabel.SetText("Histogram Bins: Auto")
	} else {
		histBinsLabel.SetText("Histogram Bins: " + strconv.Itoa(histBins))
	}
	histBinsSlider.OnChanged = func(value float64) {
		intValue := int(value)
		if intValue == 0 {
			histBinsLabel.SetText("Histogram Bins: Auto")
		} else {
			histBinsLabel.SetText("Histogram Bins: " + strconv.Itoa(intValue))
		}
		if pp.parameterChangeHandler != nil {
			pp.parameterChangeHandler("histogram_bins", intValue)
		}
	}

	// Smoothing Strength
	smoothingSlider := widget.NewSlider(0.0, 5.0)
	smoothingLabel := widget.NewLabel("Smoothing Strength: 1.0")
	smoothing := pp.getFloatParam(params, "smoothing_strength", 1.0)
	smoothingSlider.SetValue(smoothing)
	smoothingLabel.SetText("Smoothing Strength: " + strconv.FormatFloat(smoothing, 'f', 1, 64))
	smoothingSlider.OnChanged = func(value float64) {
		smoothingLabel.SetText("Smoothing Strength: " + strconv.FormatFloat(value, 'f', 1, 64))
		if pp.parameterChangeHandler != nil {
			pp.parameterChangeHandler("smoothing_strength", value)
		}
	}

	// Boolean parameters
	noiseRobustnessCheck := widget.NewCheck("MAOTSU Noise Robustness", func(checked bool) {
		if pp.parameterChangeHandler != nil {
			pp.parameterChangeHandler("noise_robustness", checked)
		}
	})
	noiseRobustnessCheck.SetChecked(pp.getBoolParam(params, "noise_robustness", true))

	gaussianPreprocessCheck := widget.NewCheck("Gaussian Preprocessing", func(checked bool) {
		if pp.parameterChangeHandler != nil {
			pp.parameterChangeHandler("gaussian_preprocessing", checked)
		}
	})
	gaussianPreprocessCheck.SetChecked(pp.getBoolParam(params, "gaussian_preprocessing", true))

	useClaheCheck := widget.NewCheck("CLAHE Contrast Enhancement", func(checked bool) {
		if pp.parameterChangeHandler != nil {
			pp.parameterChangeHandler("use_clahe", checked)
		}
	})
	useClaheCheck.SetChecked(pp.getBoolParam(params, "use_clahe", false))

	guidedFilteringCheck := widget.NewCheck("Guided Filtering", func(checked bool) {
		if pp.parameterChangeHandler != nil {
			pp.parameterChangeHandler("guided_filtering", checked)
		}
	})
	guidedFilteringCheck.SetChecked(pp.getBoolParam(params, "guided_filtering", false))

	parallelProcessingCheck := widget.NewCheck("Parallel Processing", func(checked bool) {
		if pp.parameterChangeHandler != nil {
			pp.parameterChangeHandler("parallel_processing", checked)
		}
	})
	parallelProcessingCheck.SetChecked(pp.getBoolParam(params, "parallel_processing", true))

	// Store widgets for updates
	pp.parameterWidgets["window_size"] = windowSizeSlider
	pp.parameterWidgets["histogram_bins"] = histBinsSlider
	pp.parameterWidgets["smoothing_strength"] = smoothingSlider
	pp.parameterWidgets["noise_robustness"] = noiseRobustnessCheck
	pp.parameterWidgets["gaussian_preprocessing"] = gaussianPreprocessCheck
	pp.parameterWidgets["use_clahe"] = useClaheCheck
	pp.parameterWidgets["guided_filtering"] = guidedFilteringCheck
	pp.parameterWidgets["parallel_processing"] = parallelProcessingCheck

	// Layout parameters
	basicGroup := container.NewVBox(
		widget.NewCard("Basic Parameters", "",
			container.NewVBox(
				container.NewVBox(windowSizeLabel, windowSizeSlider),
				container.NewVBox(histBinsLabel, histBinsSlider),
				container.NewVBox(smoothingLabel, smoothingSlider),
			),
		),
	)

	preprocessingGroup := container.NewVBox(
		widget.NewCard("Preprocessing Options", "",
			container.NewVBox(
				noiseRobustnessCheck,
				gaussianPreprocessCheck,
				useClaheCheck,
				guidedFilteringCheck,
			),
		),
	)

	performanceGroup := container.NewVBox(
		widget.NewCard("Performance", "",
			container.NewVBox(parallelProcessingCheck),
		),
	)

	pp.parametersContent.Add(basicGroup)
	pp.parametersContent.Add(preprocessingGroup)
	pp.parametersContent.Add(performanceGroup)
}

func (pp *ParameterPanel) buildTriclassParameters(params map[string]interface{}) {
	// Initial threshold method
	initialMethod := widget.NewSelect([]string{"otsu", "mean", "median", "triangle"}, func(value string) {
		if pp.parameterChangeHandler != nil {
			pp.parameterChangeHandler("initial_threshold_method", value)
		}
	})
	initialMethod.SetSelected(pp.getStringParam(params, "initial_threshold_method", "otsu"))

	// Max iterations
	maxIterSlider := widget.NewSlider(3, 15)
	maxIterLabel := widget.NewLabel("Max Iterations: 8")
	maxIter := pp.getIntParam(params, "max_iterations", 8)
	maxIterSlider.SetValue(float64(maxIter))
	maxIterLabel.SetText("Max Iterations: " + strconv.Itoa(maxIter))
	maxIterSlider.OnChanged = func(value float64) {
		intValue := int(value)
		maxIterLabel.SetText("Max Iterations: " + strconv.Itoa(intValue))
		if pp.parameterChangeHandler != nil {
			pp.parameterChangeHandler("max_iterations", intValue)
		}
	}

	// Convergence precision
	convergenceSlider := widget.NewSlider(0.5, 2.0)
	convergenceLabel := widget.NewLabel("Convergence Precision: 1.0")
	convergence := pp.getFloatParam(params, "convergence_precision", 1.0)
	convergenceSlider.SetValue(convergence)
	convergenceLabel.SetText("Convergence Precision: " + strconv.FormatFloat(convergence, 'f', 1, 64))
	convergenceSlider.OnChanged = func(value float64) {
		convergenceLabel.SetText("Convergence Precision: " + strconv.FormatFloat(value, 'f', 1, 64))
		if pp.parameterChangeHandler != nil {
			pp.parameterChangeHandler("convergence_precision", value)
		}
	}

	// Class separation
	classSeparationSlider := widget.NewSlider(0.1, 0.8)
	classSeparationLabel := widget.NewLabel("Class Separation: 0.50")
	classSeparation := pp.getFloatParam(params, "class_separation", 0.5)
	classSeparationSlider.SetValue(classSeparation)
	classSeparationLabel.SetText("Class Separation: " + strconv.FormatFloat(classSeparation, 'f', 2, 64))
	classSeparationSlider.OnChanged = func(value float64) {
		classSeparationLabel.SetText("Class Separation: " + strconv.FormatFloat(value, 'f', 2, 64))
		if pp.parameterChangeHandler != nil {
			pp.parameterChangeHandler("class_separation", value)
		}
	}

	// Boolean parameters
	preprocessingCheck := widget.NewCheck("Advanced Preprocessing", func(checked bool) {
		if pp.parameterChangeHandler != nil {
			pp.parameterChangeHandler("preprocessing", checked)
		}
	})
	preprocessingCheck.SetChecked(pp.getBoolParam(params, "preprocessing", true))

	cleanupCheck := widget.NewCheck("Result Cleanup", func(checked bool) {
		if pp.parameterChangeHandler != nil {
			pp.parameterChangeHandler("result_cleanup", checked)
		}
	})
	cleanupCheck.SetChecked(pp.getBoolParam(params, "result_cleanup", true))

	noiseRobustnessCheck := widget.NewCheck("Non-Local Means Denoising", func(checked bool) {
		if pp.parameterChangeHandler != nil {
			pp.parameterChangeHandler("noise_robustness", checked)
		}
	})
	noiseRobustnessCheck.SetChecked(pp.getBoolParam(params, "noise_robustness", true))

	guidedFilteringCheck := widget.NewCheck("Guided Filtering", func(checked bool) {
		if pp.parameterChangeHandler != nil {
			pp.parameterChangeHandler("guided_filtering", checked)
		}
	})
	guidedFilteringCheck.SetChecked(pp.getBoolParam(params, "guided_filtering", true))

	parallelCheck := widget.NewCheck("Parallel Processing", func(checked bool) {
		if pp.parameterChangeHandler != nil {
			pp.parameterChangeHandler("parallel_processing", checked)
		}
	})
	parallelCheck.SetChecked(pp.getBoolParam(params, "parallel_processing", true))

	// Store widgets for updates
	pp.parameterWidgets["initial_threshold_method"] = initialMethod
	pp.parameterWidgets["max_iterations"] = maxIterSlider
	pp.parameterWidgets["convergence_precision"] = convergenceSlider
	pp.parameterWidgets["class_separation"] = classSeparationSlider
	pp.parameterWidgets["preprocessing"] = preprocessingCheck
	pp.parameterWidgets["result_cleanup"] = cleanupCheck
	pp.parameterWidgets["noise_robustness"] = noiseRobustnessCheck
	pp.parameterWidgets["guided_filtering"] = guidedFilteringCheck
	pp.parameterWidgets["parallel_processing"] = parallelCheck

	// Layout parameters
	algorithmGroup := container.NewVBox(
		widget.NewCard("Algorithm Parameters", "",
			container.NewVBox(
				container.NewVBox(widget.NewLabel("Initial Method"), initialMethod),
				container.NewVBox(maxIterLabel, maxIterSlider),
				container.NewVBox(convergenceLabel, convergenceSlider),
				container.NewVBox(classSeparationLabel, classSeparationSlider),
			),
		),
	)

	processingGroup := container.NewVBox(
		widget.NewCard("Processing Options", "",
			container.NewVBox(
				preprocessingCheck,
				cleanupCheck,
				noiseRobustnessCheck,
				guidedFilteringCheck,
			),
		),
	)

	performanceGroup := container.NewVBox(
		widget.NewCard("Performance", "",
			container.NewVBox(parallelCheck),
		),
	)

	pp.parametersContent.Add(algorithmGroup)
	pp.parametersContent.Add(processingGroup)
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
