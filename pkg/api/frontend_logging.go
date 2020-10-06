package api

import (
	"fmt"
	"strings"

	"github.com/getsentry/sentry-go"
	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/models"
	"github.com/inconshreveable/log15"
)

var frontendLogger = log.New("frontend")

type FrontendSentryExceptionValue struct {
	Value      string            `json:"value,omitempty"`
	Type       string            `json:"type,omitempty"`
	Stacktrace sentry.Stacktrace `json:"stacktrace,omitempty"`
}

type FrontendSentryException struct {
	Values []FrontendSentryExceptionValue `json:"values,omitempty"`
}

type FrontendSentryEvent struct {
	*sentry.Event
	Exception *FrontendSentryException `json:"exception,omitempty"`
}

func (value *FrontendSentryExceptionValue) FmtMessage() string {
	return fmt.Sprintf("%s: %s", value.Type, value.Value)
}

func (value *FrontendSentryExceptionValue) FmtStacktrace() string {
	var stacktrace = value.FmtMessage()
	for _, frame := range value.Stacktrace.Frames {
		stacktrace += fmt.Sprintf("\n  at %s (%s:%v:%v)", frame.Function, frame.Filename, frame.Lineno, frame.Colno)
	}
	return stacktrace
}

func (exception *FrontendSentryException) FmtStacktraces() string {
	var stacktraces []string
	for _, value := range exception.Values {
		stacktraces = append(stacktraces, value.FmtStacktrace())
	}
	return strings.Join(stacktraces, "\n \n")
}

func (event *FrontendSentryEvent) ToLogContext() log15.Ctx {
	var ctx = make(log15.Ctx)
	ctx["url"] = event.Request.URL
	ctx["user_agent"] = event.Request.Headers["User-Agent"]
	ctx["event_id"] = event.EventID
	if event.Exception != nil {
		ctx["stacktrace"] = event.Exception.FmtStacktraces()
	}

	return ctx
}

func (hs *HTTPServer) LogFrontendMessage(c *models.ReqContext, event FrontendSentryEvent) Response {

	var msg = "unknown"

	if len(event.Message) > 0 {
		msg = event.Message
	} else if event.Exception != nil && len(event.Exception.Values) > 0 {
		msg = event.Exception.Values[0].FmtMessage()
	}

	var ctx = event.ToLogContext()

	switch event.Level {
	case sentry.LevelError:
		frontendLogger.Error(msg, ctx)
	case sentry.LevelWarning:
		frontendLogger.Warn(msg, ctx)
	case sentry.LevelDebug:
		frontendLogger.Debug(msg, ctx)
	default:
		frontendLogger.Info(msg, ctx)
	}

	return Success("ok")
}
