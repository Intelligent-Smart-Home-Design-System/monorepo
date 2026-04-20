package logging

import (
	"fmt"

	"github.com/rs/zerolog"
	tlog "go.temporal.io/sdk/log"
)

type TemporalLogger struct {
	logger zerolog.Logger
}

func NewTemporalLogger(logger zerolog.Logger) *TemporalLogger {
	return &TemporalLogger{logger: logger}
}

func (l *TemporalLogger) Debug(msg string, keyvals ...interface{}) {
	l.withFields(keyvals...).Debug().Msg(msg)
}

func (l *TemporalLogger) Info(msg string, keyvals ...interface{}) {
	l.withFields(keyvals...).Info().Msg(msg)
}

func (l *TemporalLogger) Warn(msg string, keyvals ...interface{}) {
	l.withFields(keyvals...).Warn().Msg(msg)
}

func (l *TemporalLogger) Error(msg string, keyvals ...interface{}) {
	l.withFields(keyvals...).Error().Msg(msg)
}

func (l *TemporalLogger) With(keyvals ...interface{}) tlog.Logger {
	return &TemporalLogger{logger: *l.withFields(keyvals...)}
}

func (l *TemporalLogger) withFields(keyvals ...interface{}) *zerolog.Logger {
	if len(keyvals) == 0 {
		return &l.logger
	}

	ctx := l.logger.With()
	for i := 0; i < len(keyvals); i += 2 {
		key := fmt.Sprintf("arg_%d", i)
		if i < len(keyvals) {
			key = fmt.Sprint(keyvals[i])
		}
		value := interface{}(nil)
		if i+1 < len(keyvals) {
			value = keyvals[i+1]
		}
		ctx = ctx.Interface(key, value)
	}

	logger := ctx.Logger()
	return &logger
}
