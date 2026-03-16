package config

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"dependency-track-exporter/internal/version"
)

type Config struct {
	WebListenAddress     string
	WebMetricsPath       string
	DTrackAddress        string
	DTrackAPIKey         string
	LogLevel             string
	LogFormat            string
	PollPeriod           time.Duration
	ClientRequestTimeout time.Duration
	ExitCode             int
}

func Parse(args []string, stdout io.Writer) (Config, error) {
	cfg := Config{
		WebListenAddress:     ":9916",
		WebMetricsPath:       "/metrics",
		DTrackAddress:        envOrDefault("DEPENDENCY_TRACK_ADDR", "http://localhost:8080"),
		DTrackAPIKey:         os.Getenv("DEPENDENCY_TRACK_API_KEY"),
		LogLevel:             "info",
		LogFormat:            "logfmt",
		PollPeriod:           1800 * time.Second,
		ClientRequestTimeout: 10 * time.Second,
		ExitCode:             -1,
	}

	fs := flag.NewFlagSet("dependency-track-exporter", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	help := fs.Bool("help", false, "")
	helpLong := fs.Bool("help-long", false, "")
	helpMan := fs.Bool("help-man", false, "")
	showVersion := fs.Bool("version", false, "")

	fs.StringVar(&cfg.WebListenAddress, "web.listen-address", cfg.WebListenAddress, "")
	fs.StringVar(&cfg.WebMetricsPath, "web.metrics-path", cfg.WebMetricsPath, "")
	fs.StringVar(&cfg.DTrackAddress, "dtrack.address", cfg.DTrackAddress, "")
	fs.StringVar(&cfg.DTrackAPIKey, "dtrack.api-key", cfg.DTrackAPIKey, "")
	fs.StringVar(&cfg.LogLevel, "log.level", cfg.LogLevel, "")
	fs.StringVar(&cfg.LogFormat, "log.format", cfg.LogFormat, "")
	fs.DurationVar(&cfg.PollPeriod, "poll.period", cfg.PollPeriod, "")
	fs.DurationVar(&cfg.ClientRequestTimeout, "client-request-timeout", cfg.ClientRequestTimeout, "")

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	if *help || *helpLong || *helpMan {
		printHelp(stdout)
		cfg.ExitCode = 0
		return cfg, nil
	}
	if *showVersion {
		fmt.Fprintf(stdout, "dependency-track-exporter version %s\n", version.String())
		cfg.ExitCode = 0
		return cfg, nil
	}

	if cfg.DTrackAPIKey == "" {
		return Config{}, errors.New("--dtrack.api-key is required or set DEPENDENCY_TRACK_API_KEY")
	}
	if cfg.PollPeriod <= 0 {
		return Config{}, errors.New("--poll.period must be > 0")
	}
	if cfg.ClientRequestTimeout <= 0 {
		return Config{}, errors.New("--client-request-timeout must be > 0")
	}
	if !validLogLevel(cfg.LogLevel) {
		return Config{}, fmt.Errorf("invalid --log.level %q", cfg.LogLevel)
	}
	if !validLogFormat(cfg.LogFormat) {
		return Config{}, fmt.Errorf("invalid --log.format %q", cfg.LogFormat)
	}
	if !strings.HasPrefix(cfg.WebMetricsPath, "/") {
		return Config{}, errors.New("--web.metrics-path must start with /")
	}

	return cfg, nil
}

func printHelp(w io.Writer) {
	fmt.Fprint(w, `Usage: dependency-track-exporter [<flags>]

A stdlib-only Dependency-Track exporter.
=
Flags:
  --help                           Show context-sensitive help (also try --help-long and --help-man).
  --web.listen-address=:9916       Address to listen on for metrics.
  --web.metrics-path=/metrics      Path under which to expose metrics.
  --dtrack.address=ADDR            Dependency-Track server address
                                   (default: http://localhost:8080 or $DEPENDENCY_TRACK_ADDR)
  --dtrack.api-key=KEY             Dependency-Track API key
                                   (default: $DEPENDENCY_TRACK_API_KEY)
  --log.level=info                 Only log messages with the given severity or above.
                                   One of: debug, info, warn, error
  --log.format=logfmt              Output format of log messages.
                                   One of: logfmt, json
  --poll.period=1800s              Frequency of requests to Dependency-Track.
  --client.request-timeout=10s     Timeout value for client requests to Dependency-Track.
  --version                        Show application version.
`)
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func validLogLevel(v string) bool {
	switch v {
	case "debug", "info", "warn", "error":
		return true
	default:
		return false
	}
}

func validLogFormat(v string) bool {
	switch v {
	case "logfmt", "json":
		return true
	default:
		return false
	}
}

type Logger struct {
	level  string
	format string
	std    *log.Logger
}

func NewLogger(format, level string, out io.Writer) *Logger {
	return &Logger{
		level:  level,
		format: format,
		std:    log.New(out, "", 0),
	}
}

func (l *Logger) Debug(msg string, kv ...any) { l.log("debug", msg, kv...) }
func (l *Logger) Info(msg string, kv ...any)  { l.log("info", msg, kv...) }
func (l *Logger) Warn(msg string, kv ...any)  { l.log("warn", msg, kv...) }
func (l *Logger) Error(msg string, kv ...any) { l.log("error", msg, kv...) }

func enabled(current, wanted string) bool {
	rank := map[string]int{
		"debug": 0,
		"info":  1,
		"warn":  2,
		"error": 3,
	}
	return rank[wanted] >= rank[current]
}

func escapeJSON(s string) string {
	replacer := strings.NewReplacer(
		`\`, `\\`,
		`"`, `\"`,
		"\n", `\n`,
		"\r", `\r`,
		"\t", `\t`,
	)
	return replacer.Replace(s)
}

func (l *Logger) log(level, msg string, kv ...any) {
	if !enabled(l.level, level) {
		return
	}

	timestamp := time.Now().UTC().Format(time.RFC3339)

	if l.format == "json" {
		var b strings.Builder
		b.WriteString(`{"ts":"`)
		b.WriteString(timestamp)
		b.WriteString(`","level":"`)
		b.WriteString(level)
		b.WriteString(`","msg":"`)
		b.WriteString(escapeJSON(msg))
		b.WriteString(`"`)
		for i := 0; i+1 < len(kv); i += 2 {
			key := fmt.Sprint(kv[i])
			val := fmt.Sprint(kv[i+1])
			b.WriteString(`,"`)
			b.WriteString(escapeJSON(key))
			b.WriteString(`":"`)
			b.WriteString(escapeJSON(val))
			b.WriteString(`"`)
		}
		b.WriteString("}")
		l.std.Print(b.String())
		return
	}

	var b strings.Builder
	b.WriteString("ts=")
	b.WriteString(timestamp)
	b.WriteString(" level=")
	b.WriteString(level)
	b.WriteString(` msg="`)
	b.WriteString(strings.ReplaceAll(msg, `"`, `'`))
	b.WriteString(`"`)
	for i := 0; i+1 < len(kv); i += 2 {
		b.WriteByte(' ')
		b.WriteString(fmt.Sprint(kv[i]))
		b.WriteByte('=')
		b.WriteString(fmt.Sprintf("%q", fmt.Sprint(kv[i+1])))
	}
	l.std.Print(b.String())
}
