package logger

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/fsoria-ttec/bne-converter/internal/config"
	"github.com/sirupsen/logrus"
)

type CustomFormatter struct {
	TimestampFormat string
	ColorEnabled    bool
}

const (
	colorRed    = 31
	colorYellow = 33
	colorPurple = 35
	colorBlue   = 36
)

func NewCustomFormatter(config config.LoggingConfig, enableColors bool) *CustomFormatter {
	return &CustomFormatter{
		TimestampFormat: config.TimestampFormat,
		ColorEnabled:    enableColors,
	}
}

func (f *CustomFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var b *bytes.Buffer

	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	timestamp := entry.Time.Format(f.TimestampFormat)
	var levelColor int

	switch entry.Level {
	case logrus.DebugLevel:
		levelColor = colorPurple
	case logrus.InfoLevel:
		levelColor = colorBlue
	case logrus.WarnLevel:
		levelColor = colorYellow
	case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
		levelColor = colorRed
	}

	levelText := strings.ToUpper(entry.Level.String())

	if f.ColorEnabled {
		fmt.Fprintf(b, "\x1b[%dm", levelColor)
	}

	// Formato base: [timestamp] [level] message
	fmt.Fprintf(b, "[%s] [%s] %s", timestamp, levelText, entry.Message)

	// Añadir campos adicionales si existen
	if len(entry.Data) > 0 {
		fmt.Fprintf(b, " |")
		for key, value := range entry.Data {
			fmt.Fprintf(b, " %s=%v", key, value)
		}
	}

	if f.ColorEnabled {
		b.WriteString("\x1b[0m")
	}

	b.WriteByte('\n')
	return b.Bytes(), nil
}
