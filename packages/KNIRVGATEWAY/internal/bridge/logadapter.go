package bridge

import (
	"github.com/rs/zerolog"
	waLog "go.mau.fi/whatsmeow/util/log"
	"go.uber.org/zap"
)

// zapAsZerolog gives mautrix-go's appservice package (which logs through a
// concrete zerolog.Logger, not an interface) a logger that forwards every
// line into the gateway's own structured zap logger instead of opening a
// second, differently-formatted log stream.
func zapAsZerolog(logger *zap.Logger) zerolog.Logger {
	return zerolog.New(&zapWriter{logger: logger, level: "info"}).With().Timestamp().Logger()
}

// zapWaLogger adapts a *zap.Logger to whatsmeow's waLog.Logger interface so
// whatsmeow's own (fairly chatty) internal logging flows through the same
// structured logger as the rest of KNIRVGATEWAY instead of a second,
// differently-formatted log stream.
type zapWaLogger struct {
	logger *zap.Logger
	module string
}

func newZapWaLogger(logger *zap.Logger, module string) waLog.Logger {
	return &zapWaLogger{logger: logger, module: module}
}

func (l *zapWaLogger) field() zap.Field { return zap.String("module", l.module) }

func (l *zapWaLogger) Errorf(msg string, args ...any) {
	l.logger.Sugar().With(l.field()).Errorf(msg, args...)
}

func (l *zapWaLogger) Warnf(msg string, args ...any) {
	l.logger.Sugar().With(l.field()).Warnf(msg, args...)
}

func (l *zapWaLogger) Infof(msg string, args ...any) {
	l.logger.Sugar().With(l.field()).Infof(msg, args...)
}

func (l *zapWaLogger) Debugf(msg string, args ...any) {
	l.logger.Sugar().With(l.field()).Debugf(msg, args...)
}

func (l *zapWaLogger) Sub(module string) waLog.Logger {
	return newZapWaLogger(l.logger, l.module+"/"+module)
}
