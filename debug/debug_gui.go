package debug

import (
	"fmt"
)

// Global debug toggle for GUI events (set from main package)
var EnableGUIDebug = false

func (dm *Manager) LogGUIEvent(component, event, details string) {
	if !EnableGUIDebug {
		return
	}
	LogInfo("GUIDebug", fmt.Sprintf("%s - %s: %s", component, event, details))
}

func (dm *Manager) LogParameterChange(algorithm, parameter string, oldValue, newValue interface{}) {
	if !EnableGUIDebug {
		return
	}
	LogInfo("GUIDebug", fmt.Sprintf("Parameter changed - Algorithm: %s, Parameter: %s, %v -> %v",
		algorithm, parameter, oldValue, newValue))
}

func (dm *Manager) LogGUIAlgorithmSwitch(fromAlgorithm, toAlgorithm string) {
	if !EnableGUIDebug {
		return
	}
	LogInfo("GUIDebug", fmt.Sprintf("GUI Algorithm switched: %s -> %s", fromAlgorithm, toAlgorithm))
}

func (dm *Manager) LogImageDisplay(imageType string, dimensions string) {
	if !EnableGUIDebug {
		return
	}
	LogInfo("GUIDebug", fmt.Sprintf("Image displayed - Type: %s, Dimensions: %s", imageType, dimensions))
}
