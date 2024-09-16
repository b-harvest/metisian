package log

import (
	"errors"
	"fmt"
	"github.com/b-harvest/metisian/util"
	"os"
	"runtime/debug"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
)

var eventQueue chan func()

func init() {
	// Logger setup
	output := zerolog.ConsoleWriter{
		Out:         os.Stderr,
		TimeFormat:  time.RFC3339,
		FormatLevel: logColorFormatter(),
	}

	log.Logger = log.Output(output)

	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	zerolog.ErrorStackFieldName = "trace"

	eventQueue = make(chan func())

	// For thread safe
	go func() {
		for event := range eventQueue {
			event()
		}
	}()
}

func enqueue(event func()) {
	eventQueue <- event
}

func Info(msg string) {
	event := func() {
		log.Info().Msg(msg)
	}
	enqueue(event)
}

func Warn(msg string) {
	event := func() {
		log.Warn().Msg(msg)
	}
	enqueue(event)
}

func Error(err error) {
	_, _, f := util.Trace(2)
	event := func() {
		log.Error().Err(err).Msg(f)
	}
	enqueue(event)
}

func ErrorDynamicArgs(i ...interface{}) {
	Error(errors.New(fmt.Sprintf("%v", i)))
}

func Fatal(err error) {
	stack := string(debug.Stack())
	event := func() {
		log.Fatal().Err(err).Msg("\n" + stack)
	}
	enqueue(event)
}

func Debug(msg any) {
	message := fmt.Sprint(msg)
	event := func() {
		log.Debug().Msg(message)
	}
	enqueue(event)
}

const (
	colorBlack = iota + 30
	colorRed
	colorGreen
	colorYellow
	colorBlue
	colorMagenta
	colorCyan
	colorWhite

	colorBold     = 1
	colorDarkGray = 90
)

func logColorFormatter() func(interface{}) string {
	return func(i interface{}) string {
		var l string
		if ll, ok := i.(string); ok {
			switch ll {
			case zerolog.LevelTraceValue:
				l = colorize("TRC", colorMagenta)
			case zerolog.LevelDebugValue:
				l = colorize("DBG", colorCyan)
			case zerolog.LevelInfoValue:
				l = colorize("INF", colorGreen)
			case zerolog.LevelWarnValue:
				l = colorize("WRN", colorYellow)
			case zerolog.LevelErrorValue:
				l = colorize(colorize("ERR", colorRed), colorBold)
			case zerolog.LevelFatalValue:
				l = colorize(colorize("FTL", colorRed), colorBold)
			case zerolog.LevelPanicValue:
				l = colorize(colorize("PNC", colorRed), colorBold)
			default:
				l = colorize(ll, colorBold)
			}
		}
		return l
	}
}

// colorize returns the string s wrapped in ANSI code c, unless disabled is true.
func colorize(s interface{}, c int) string {
	return fmt.Sprintf("\x1b[%dm%v\x1b[0m", c, s)
}
