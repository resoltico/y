package logger

import (
	"io"
	"os"

	"github.com/rs/zerolog"
)

type ZerologAdapter struct {
	logger zerolog.Logger
}

func NewZerolog(writer io.Writer, level zerolog.Level) *ZerologAdapter {
	logger := zerolog.New(writer).
		Level(level).
		With().
		Timestamp().
		Logger()

	return &ZerologAdapter{logger: logger}
}

func NewConsoleLogger(level zerolog.Level) *ZerologAdapter {
	consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout}
	return NewZerolog(consoleWriter, level)
}

func (z *ZerologAdapter) Info(component, message string, fields map[string]interface{}) {
	event := z.logger.Info().Str("component", component)
	for k, v := range fields {
		event = event.Interface(k, v)
	}
	event.Msg(message)
}

func (z *ZerologAdapter) Error(component string, err error, fields map[string]interface{}) {
	event := z.logger.Error().Str("component", component).Err(err)
	for k, v := range fields {
		event = event.Interface(k, v)
	}
	event.Msg("operation failed")
}

func (z *ZerologAdapter) Warning(component, message string, fields map[string]interface{}) {
	event := z.logger.Warn().Str("component", component)
	for k, v := range fields {
		event = event.Interface(k, v)
	}
	event.Msg(message)
}

func (z *ZerologAdapter) Debug(component, message string, fields map[string]interface{}) {
	event := z.logger.Debug().Str("component", component)
	for k, v := range fields {
		event = event.Interface(k, v)
	}
	event.Msg(message)
}
