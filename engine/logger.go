package engine

import (
	"context"

	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

type vinceLogger struct {
	*zap.Logger
	file string
}

func init() {
	zap.RegisterEncoder("vince", func(cfg zapcore.EncoderConfig) (zapcore.Encoder, error) {
		enc := zapcore.NewJSONEncoder(cfg)
		return &MsgEncoder{ObjectEncoder: enc, enc: enc}, nil
	})
}

type loggerKey struct{}

const errorLogFile = "error_log_file"

func log(ctx context.Context) *vinceLogger {
	lg := ctx.Value(loggerKey{}).(*vinceLogger)
	if v := ctx.Value(errLogInfo{}); v != nil {
		i := v.(*ErrorLog)
		return &vinceLogger{Logger: lg.WithOptions(wrapCore(i.Name, i.Level)), file: i.Name}
	}
	return lg
}

func wrapCore(file, level string) zap.Option {
	return zap.WrapCore(func(c zapcore.Core) zapcore.Core {
		return &MsgCore{Core: c, Level: stringLevel(level)}
	})
}

func withLog(ctx context.Context, lg *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, &vinceLogger{Logger: lg})
}

func newLogger() (*zap.Logger, error) {
	c := zap.NewProductionConfig()
	c.Encoding = "vince"
	c.DisableCaller = true
	c.DisableStacktrace = true
	return c.Build()
}

type MsgCore struct {
	zapcore.Core
	Level zapcore.Level
}

func stringLevel(s string) zapcore.Level {
	switch s {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "notice", "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "crit":
		return zapcore.DPanicLevel
	case "alert":
		return zapcore.PanicLevel
	case "emerg":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

func (msg MsgCore) Enabled(lvl zapcore.Level) bool {
	return msg.Level.Enabled(lvl)
}

type MsgEncoder struct {
	zapcore.ObjectEncoder
	enc zapcore.Encoder
}

func (msg MsgEncoder) Clone() zapcore.Encoder {
	enc := msg.enc.Clone()
	return &MsgEncoder{enc: enc, ObjectEncoder: enc}
}

func (msg MsgEncoder) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	a := fields[:0]
	var f string
	for i := 0; i < len(fields); i++ {
		if fields[i].Key == errorLogFile {
			f = fields[i].String
		} else {
			a = append(a, fields[i])
		}
	}
	if f != "" {
		// encode as log message
	}
	return msg.enc.EncodeEntry(entry, a)
}
