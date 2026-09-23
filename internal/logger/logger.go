package logger

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/ORG_PLACEHOLDER/EnterpriseRag/internal/types"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"
	"gopkg.in/natefinch/lumberjack.v2"
)

// appLogger is a private instance, so external code rewriting logrus global
// state cannot make our logs disappear.
var appLogger = logrus.New()

var (
	loggerMu      sync.Mutex
	activeLogFile io.WriteCloser
	ansiEscapeRE  = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
)

// ansiStripWriter removes ANSI color/style sequences so file logs stay plain text
// while stdout can still render colors in a terminal.
type ansiStripWriter struct {
	w io.Writer
}

func (s *ansiStripWriter) Write(p []byte) (int, error) {
	_, err := s.w.Write(ansiEscapeRE.ReplaceAll(p, nil))
	return len(p), err
}

// LogLevel is the log level type.
type LogLevel string

const (
	LevelDebug LogLevel = "debug"
	LevelInfo  LogLevel = "info"
	LevelWarn  LogLevel = "warn"
	LevelError LogLevel = "error"
	LevelFatal LogLevel = "fatal"
)

// ANSI colour codes
const (
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorWhite  = "\033[37m"
	colorGray   = "\033[90m"
	colorBold   = "\033[1m"
	colorReset  = "\033[0m"
)

type CustomFormatter struct {
	ForceColor bool
	Template   string
	// Template placeholders: %d=time %level=level %thread=goroutine %logger=caller %traceId=request ID %msg=message plus structured fields

	// threadNeeded caches whether the template references %thread, so that
	// runtime.Stack is not called for every log line.
	threadNeeded bool
}

// levelColorFor returns the ANSI colour code for a log level, or an empty string
// when colour is disabled.
func levelColorFor(level logrus.Level) string {
	switch level {
	case logrus.DebugLevel:
		return colorCyan
	case logrus.InfoLevel:
		return colorGreen
	case logrus.WarnLevel:
		return colorYellow
	case logrus.ErrorLevel:
		return colorRed
	case logrus.FatalLevel:
		return colorPurple
	}
	return ""
}

func (f *CustomFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	timestamp := entry.Time.Format("2006-01-02 15:04:05.000")
	level := strings.ToUpper(entry.Level.String())

	caller, _ := entry.Data["caller"].(string)
	traceID, _ := entry.Data["request_id"].(string)

	keys := make([]string, 0, len(entry.Data))
	for k := range entry.Data {
		if k != "caller" && k != "request_id" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	if f.Template != "" {
		msg := entry.Message
		for _, k := range keys {
			msg += fmt.Sprintf(" %s=%v", k, entry.Data[k])
		}
		shortCaller := caller
		if len(shortCaller) > 50 {
			shortCaller = shortCaller[len(shortCaller)-50:]
		}
		// Only read the goroutine ID when the template references %thread, so that
		// runtime.Stack is not executed for every log line.
		thread := ""
		if f.threadNeeded {
			thread = getGoroutineID()
		}
		// Level colouring happens during placeholder substitution, so that a later
		// whole-line ReplaceAll cannot accidentally colour literal "INFO"/"ERROR"
		// strings that appear inside the message.
		levelOut := level
		if f.ForceColor {
			if c := levelColorFor(entry.Level); c != "" {
				levelOut = c + level + colorReset
			}
		}
		// A single NewReplacer pass avoids the second substitution that chained
		// ReplaceAll calls cause when one placeholder's value happens to contain
		// another placeholder's literal text.
		r := strings.NewReplacer(
			"%d", timestamp,
			"%level", levelOut,
			"%thread", thread,
			"%logger", shortCaller,
			"%traceId", traceID,
			"%msg", msg,
		)
		return []byte(r.Replace(f.Template) + "\n"), nil
	}

	var levelColor, resetColor string
	if f.ForceColor {
		switch entry.Level {
		case logrus.DebugLevel:
			levelColor = colorCyan
		case logrus.InfoLevel:
			levelColor = colorGreen
		case logrus.WarnLevel:
			levelColor = colorYellow
		case logrus.ErrorLevel:
			levelColor = colorRed
		case logrus.FatalLevel:
			levelColor = colorPurple
		default:
			levelColor = colorReset
		}
		resetColor = colorReset
	}

	fields := ""

	if v, ok := entry.Data["request_id"]; ok {
		if f.ForceColor {
			fields += fmt.Sprintf("%s%v%s ",
				colorBlue, v, colorReset)
		} else {
			fields += fmt.Sprintf("%v ", v)
		}
	}

	for _, k := range keys {
		if f.ForceColor {
			val := fmt.Sprintf("%v", entry.Data[k])
			coloredVal := fmt.Sprintf("%s%s%s", colorWhite, val, colorReset)
			if k == "error" {
				coloredVal = fmt.Sprintf("%s%s%s", colorRed, val, colorReset)
			}
			fields += fmt.Sprintf("%s%s%s=%s ",
				colorCyan, k, colorReset, coloredVal)
		} else {
			fields += fmt.Sprintf("%s=%v ", k, entry.Data[k])
		}
	}

	fields = strings.TrimSpace(fields)

	if f.ForceColor {
		coloredTimestamp := fmt.Sprintf("%s%s%s", colorGray, timestamp, resetColor)
		coloredCaller := caller
		if caller != "" {
			coloredCaller = fmt.Sprintf("%s%s%s", colorPurple, caller, resetColor)
		}
		return []byte(fmt.Sprintf("%s%-5s%s[%s] [%s] %-20s | %s\n",
			levelColor, level, resetColor, coloredTimestamp, fields, coloredCaller, entry.Message)), nil
	}

	return []byte(fmt.Sprintf("%-5s[%s] [%s] %-20s | %s\n",
		level, timestamp, fields, caller, entry.Message)), nil
}

func getGoroutineID() string {
	buf := make([]byte, 64)
	buf = buf[:runtime.Stack(buf, false)]
	// buf format: "goroutine 123 [running]:" followed by the stack
	i := 0
	for i < len(buf) && buf[i] != ' ' {
		i++
	}
	if i >= len(buf) {
		return "0"
	}
	buf = buf[i+1:]
	j := 0
	for j < len(buf) && buf[j] != ' ' {
		j++
	}
	return string(buf[:j])
}

// init applies the global logging configuration.
func init() {
	ConfigureFromEnv()
}

// ConfigureFromEnv re-applies the logging configuration from the environment.
// This lets LOG_LEVEL / LOG_PATH take effect as soon as main() has loaded .env.
func ConfigureFromEnv() {
	loggerMu.Lock()
	defer loggerMu.Unlock()

	if activeLogFile != nil {
		_ = activeLogFile.Close()
		activeLogFile = nil
	}

	logLevel := getLogLevelFromEnv()
	appLogger.SetLevel(logLevel)

	writer := io.Writer(os.Stdout)
	logPath := resolveLogPathFromEnv()
	if logPath != "" {
		file, err := openLogFile(logPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "logger: failed to open log file %s: %v\n", logPath, err)
		} else {
			activeLogFile = file
			writer = io.MultiWriter(os.Stdout, &ansiStripWriter{w: file})
		}
	}

	// Keep writing to stdout by default, and also to a file when one is available.
	appLogger.SetOutput(writer)

	// Disable ANSI colour when not attached to a terminal (Docker log collection
	// and similar), so log aggregation and search are not disrupted.
	forceColor := false
	if fi, err := os.Stdout.Stat(); err == nil {
		forceColor = (fi.Mode() & os.ModeCharDevice) != 0
	}

	// Set the log format without changing the global time zone.
	tmpl := resolveLogFormatFromEnv()
	appLogger.SetFormatter(&CustomFormatter{
		ForceColor:   forceColor,
		Template:     tmpl,
		threadNeeded: strings.Contains(tmpl, "%thread"),
	})
	appLogger.SetReportCaller(false)
}

// GetLogger returns the logger instance.
func GetLogger(c context.Context) *logrus.Entry {
	if logger := c.Value(types.LoggerContextKey); logger != nil {
		return logger.(*logrus.Entry)
	}
	return logrus.NewEntry(appLogger)
}

// SetOutput overrides the internal logger's output destination.
// Intended for use in tests that need to capture and assert on log content
// (e.g. verifying secrets are not written out). Restore the original writer
// (usually os.Stdout) in a defer after the test.
func SetOutput(w io.Writer) {
	loggerMu.Lock()
	defer loggerMu.Unlock()
	appLogger.SetOutput(w)
}

// SetLogLevel sets the log level.
func SetLogLevel(level LogLevel) {
	var logLevel logrus.Level

	switch level {
	case LevelDebug:
		logLevel = logrus.DebugLevel
	case LevelInfo:
		logLevel = logrus.InfoLevel
	case LevelWarn:
		logLevel = logrus.WarnLevel
	case LevelError:
		logLevel = logrus.ErrorLevel
	case LevelFatal:
		logLevel = logrus.FatalLevel
	default:
		logLevel = logrus.InfoLevel
	}

	appLogger.SetLevel(logLevel)
}

// getLogLevelFromEnv reads the log level configuration from the environment.
func getLogLevelFromEnv() logrus.Level {
	logLevelStr := strings.ToLower(os.Getenv("LOG_LEVEL"))

	switch logLevelStr {
	case "debug":
		return logrus.DebugLevel
	case "info":
		return logrus.InfoLevel
	case "warn", "warning":
		return logrus.WarnLevel
	case "error":
		return logrus.ErrorLevel
	case "fatal":
		return logrus.FatalLevel
	default:
		return logrus.DebugLevel
	}
}

func resolveLogPathFromEnv() string {
	if logPath := strings.TrimSpace(os.Getenv("LOG_PATH")); logPath != "" {
		return filepath.Clean(logPath)
	}
	return defaultMacAppLogPath()
}

// resolveLogFormatFromEnv reads a custom log format template from the LOG_FORMAT
// environment variable. An empty value keeps the built-in default format; a
// non-empty value is used as a template and supports the placeholders:
// %d=time %level=level %thread=goroutine %logger=caller %traceId=request ID %msg=message plus structured fields
func resolveLogFormatFromEnv() string {
	return strings.TrimSpace(os.Getenv("LOG_FORMAT"))
}

func defaultMacAppLogPath() string {
	execPath, err := os.Executable()
	if err != nil || !strings.Contains(execPath, ".app/Contents/MacOS") {
		return ""
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	appName := "EnterpriseRag Lite"
	if idx := strings.Index(execPath, ".app/Contents/MacOS"); idx >= 0 {
		bundleName := filepath.Base(execPath[:idx+4])
		if trimmed := strings.TrimSuffix(bundleName, ".app"); trimmed != "" {
			appName = trimmed
		}
	}

	return filepath.Join(homeDir, "Library", "Logs", appName, appName+".log")
}

func openLogFile(logPath string) (io.WriteCloser, error) {
	dir := filepath.Dir(logPath)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}
	return &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    50, // megabytes
		MaxBackups: 3,
		MaxAge:     28, // days
		Compress:   true,
	}, nil
}

// addCaller adds the caller field.
func addCaller(entry *logrus.Entry, skip int) *logrus.Entry {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return entry
	}
	shortFile := path.Base(file)
	funcName := "unknown"
	if fn := runtime.FuncForPC(pc); fn != nil {
		// Keep only the function name, without the package path (e.g. doSomething)
		fullName := path.Base(fn.Name())
		parts := strings.Split(fullName, ".")
		funcName = parts[len(parts)-1]
	}
	return entry.WithField("caller", fmt.Sprintf("%s:%d[%s]", shortFile, line, funcName))
}

// WithRequestID adds the request ID to the log entry.
func WithRequestID(c context.Context, requestID string) context.Context {
	return WithField(c, "request_id", requestID)
}

// WithField adds a single field to the log entry.
func WithField(c context.Context, key string, value interface{}) context.Context {
	logger := GetLogger(c).WithField(key, value)
	return context.WithValue(c, types.LoggerContextKey, logger)
}

// WithFields adds several fields to the log entry.
func WithFields(c context.Context, fields logrus.Fields) context.Context {
	logger := GetLogger(c).WithFields(fields)
	return context.WithValue(c, types.LoggerContextKey, logger)
}

// Debug logs at debug level.
func Debug(c context.Context, args ...interface{}) {
	addCaller(GetLogger(c), 2).Debug(args...)
}

// Debugf logs at debug level using a format string.
func Debugf(c context.Context, format string, args ...interface{}) {
	addCaller(GetLogger(c), 2).Debugf(format, args...)
}

// Info logs at info level.
func Info(c context.Context, args ...interface{}) {
	addCaller(GetLogger(c), 2).Info(args...)
}

// Infof logs at info level using a format string.
func Infof(c context.Context, format string, args ...interface{}) {
	addCaller(GetLogger(c), 2).Infof(format, args...)
}

// Warn logs at warning level.
func Warn(c context.Context, args ...interface{}) {
	addCaller(GetLogger(c), 2).Warn(args...)
}

// Warnf logs at warning level using a format string.
func Warnf(c context.Context, format string, args ...interface{}) {
	addCaller(GetLogger(c), 2).Warnf(format, args...)
}

// Fields aliases logrus.Fields so callers in other packages can use the
// short form `logger.Fields{...}` without importing logrus directly.
type Fields = logrus.Fields

// WarnWithFields emits a warning with structured fields. Use this for
// audit-relevant events (cross-tenant probes, invariant violations) so that
// log aggregators can index the tenant/resource identifiers without
// parsing free-form text. Format-string style (Warnf) is appropriate for
// low-stakes diagnostic messages.
func WarnWithFields(c context.Context, fields Fields, msg string) {
	if fields == nil {
		fields = Fields{}
	}
	addCaller(GetLogger(c), 2).WithFields(fields).Warn(msg)
}

// Error logs at error level.
func Error(c context.Context, args ...interface{}) {
	addCaller(GetLogger(c), 2).Error(args...)
}

// Errorf logs at error level using a format string.
func Errorf(c context.Context, format string, args ...interface{}) {
	addCaller(GetLogger(c), 2).Errorf(format, args...)
}

// ErrorWithFields logs at error level with additional fields.
func ErrorWithFields(c context.Context, err error, fields logrus.Fields) {
	if fields == nil {
		fields = logrus.Fields{}
	}
	if err != nil {
		fields["error"] = err.Error()
	}
	addCaller(GetLogger(c), 2).WithFields(fields).Error("an error occurred")
}

// Fatal logs at fatal level and exits the program.
func Fatal(c context.Context, args ...interface{}) {
	addCaller(GetLogger(c), 2).Fatal(args...)
}

// Fatalf logs at fatal level using a format string and exits the program.
func Fatalf(c context.Context, format string, args ...interface{}) {
	addCaller(GetLogger(c), 2).Fatalf(format, args...)
}

// CloneContext copies the key values from a context into a new context.
//
// Which keys survive is decided by types.contextCloneAcrossDetach, which lives
// next to where context keys are declared so that adding a key and deciding
// its fate are the same edit. Keeping that decision here instead meant every
// new key silently defaulted to being dropped.
func CloneContext(ctx context.Context) context.Context {
	newCtx := context.Background()

	for _, k := range types.ContextKeysClonedAcrossDetach() {
		if v := ctx.Value(k); v != nil {
			newCtx = context.WithValue(newCtx, k, v)
		}
	}

	// Preserve the active OpenTelemetry span across the rebuild. The Langfuse
	// *Trace handle above carries the trace id, but span PARENTING flows through
	// the OTel span context (trace.SpanFromContext), which CloneContext would
	// otherwise drop — orphaning child spans opened after a CloneContext (e.g.
	// the agent engine's agent.execute becoming a separate trace from the HTTP
	// root). Re-inject the recording span so children stitch to the same trace.
	if sp := trace.SpanFromContext(ctx); sp.IsRecording() {
		newCtx = trace.ContextWithSpan(newCtx, sp)
	}

	return newCtx
}

// CloneContextWithoutTrace copies the same identity keys as CloneContext but
// drops the Langfuse *Trace handle and the OpenTelemetry span. Use it for
// background work that must keep tenant/session identity yet must not attach
// child spans (Docker Engine HTTP, idle sweeps, image pulls kicked off after
// a tool has returned) onto the originating chat trace.
func CloneContextWithoutTrace(ctx context.Context) context.Context {
	newCtx := context.Background()
	for _, k := range types.ContextKeysClonedAcrossDetach() {
		if k == types.LangfuseTraceContextKey {
			continue
		}
		if v := ctx.Value(k); v != nil {
			newCtx = context.WithValue(newCtx, k, v)
		}
	}
	return newCtx
}
