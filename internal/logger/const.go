package logger

import (
	_ "embed"
	"log/slog"

	"github.com/google/uuid"
)

func WithString(key, val string) Field {
	return Field{a: slog.String(key, val)}
}

func WithUnt64(key string, val uint64) Field {
	return Field{a: slog.Uint64(key, val)}
}

func WithInt64(key string, val int64) Field {
	return Field{a: slog.Int64(key, val)}
}

func WithHTTPCode(val int) Field {
	return WithInt("http_code", val)
}

func WithFloat64(key string, val float64) Field {
	return Field{a: slog.Float64(key, val)}
}

func WithInt(key string, val int) Field {
	return Field{a: slog.Int(key, val)}
}

func WithEmail(val string) Field {
	return WithString("email", val)
}

func WithUserName(val string) Field {
	return WithString("user_name", val)
}

func WithPath(path string) Field {
	return WithString("path", path)
}

func WithArtifactID(artifactID uuid.UUID) Field {
	return WithString("artifact_id", artifactID.String())
}

func WithService(serviceName string) Field {
	return WithString("service", serviceName)
}
