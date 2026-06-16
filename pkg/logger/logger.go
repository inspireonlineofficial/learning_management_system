package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

var log *slog.Logger

// secretFields contains field names that should never be logged
var secretFields = map[string]bool{
	"password":          true,
	"password_hash":     true,
	"otp":               true,
	"otp_hash":          true,
	"token":             true,
	"refresh_token":     true,
	"access_token":      true,
	"jwt":               true,
	"secret":            true,
	"api_key":           true,
	"private_key":       true,
	"client_secret":     true,
	"smtp_password":     true,
	"database_dsn":      true,
	"rustfs_secret_key": true,
}

func init() {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Redact secret fields
			if secretFields[strings.ToLower(a.Key)] {
				return slog.Attr{Key: a.Key, Value: slog.StringValue("[REDACTED]")}
			}
			return a
		},
	})
	log = slog.New(handler)
}

// Info logs an info-level message
func Info(ctx context.Context, msg string, args ...any) {
	args = appendRequestID(ctx, args)
	log.InfoContext(ctx, msg, args...)
}

// Error logs an error-level message
func Error(ctx context.Context, msg string, args ...any) {
	args = appendRequestID(ctx, args)
	log.ErrorContext(ctx, msg, args...)
}

// Warn logs a warning-level message
func Warn(ctx context.Context, msg string, args ...any) {
	args = appendRequestID(ctx, args)
	log.WarnContext(ctx, msg, args...)
}

// Debug logs a debug-level message
func Debug(ctx context.Context, msg string, args ...any) {
	args = appendRequestID(ctx, args)
	log.DebugContext(ctx, msg, args...)
}

// appendRequestID extracts request_id from context and appends it to args
func appendRequestID(ctx context.Context, args []any) []any {
	if requestID := ctx.Value("request_id"); requestID != nil {
		args = append(args, "request_id", requestID)
	}
	return args
}
