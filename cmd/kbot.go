/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/hirosassa/zerodriver"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelBridge "go.opentelemetry.io/otel/bridge/opentracing"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.18.0"
	"go.opentelemetry.io/otel/trace"
	telebot "gopkg.in/telebot.v3"
)

var (
	// TeleToken bot
	TeleToken = os.Getenv("TELE_TOKEN")
	// MetricsHost exporter host:port
	MetricsHost = os.Getenv("METRICS_HOST")
	// TracesHost exporter host:port
	TracesHost = os.Getenv("TRACES_HOST")
	//"tempo-distributor.monitoring.svc.cluster.local"
)
var otlp_grpc = "4317"

var (
	otelTracer            trace.Tracer
	bridgeTracer          *otelBridge.BridgeTracer
	wrapperTracerProvider *otelBridge.WrapperTracerProvider
	otrc_ctx              context.Context
	otrc_span             opentracing.Span
)

// Initialize OpenTelemetry
func initMetrics(ctx context.Context) {

	// Create a new OTLP Metric gRPC exporter with the specified endpoint and options
	exporter, _ := otlpmetricgrpc.New(
		ctx,
		otlpmetricgrpc.WithEndpoint(MetricsHost),
		otlpmetricgrpc.WithInsecure(),
	)

	// Define the resource with attributes that are common to all metrics.
	// labels/tags/resources that are common to all metrics.
	resource := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(fmt.Sprintf("kbot_%s", appVersion)),
	)

	// Create a new MeterProvider with the specified resource and reader
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(resource),
		sdkmetric.WithReader(
			// collects and exports metric data every 10 seconds.
			sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(10*time.Second)),
		),
	)

	// Set the global MeterProvider to the newly created MeterProvider
	otel.SetMeterProvider(mp)

}

// Initializes an OTLP exporter, and configures the corresponding trace and
// metric providers.
func initTraces(ctx context.Context) {

	logger := zerodriver.NewProductionLogger()

	res, err := resource.New(ctx,
		resource.WithAttributes(
			// the service name used to display traces in backends
			semconv.ServiceName("kbot-trace-service"),
			semconv.ServiceNameKey.String(appVersion),
		),
	)
	if err != nil {
		logger.Fatal().Str("Error", err.Error()).Msg("<initTraces> failed to create resource: 'kbot-trace-service'")
		return
	}

	endpoint := strings.Split(TracesHost, ":")
	if len(endpoint) == 1 {
		TracesHost = TracesHost + ":" + otlp_grpc
	}

	// Set up a trace exporter
	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithEndpoint(TracesHost), otlptracegrpc.WithInsecure())
	if err != nil {
		logger.Fatal().Str("Error", err.Error()).Msg("<initTraces> failed to create trace exporter")
		return
	}

	// Register the trace exporter with a TracerProvider, using a batch
	// span processor to aggregate spans before export.
	bsp := sdktrace.NewBatchSpanProcessor(traceExporter)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)
	otelTracer = tracerProvider.Tracer("kbot_tracer")
	bridgeTracer, wrapperTracerProvider = otelBridge.NewTracerPair(otelTracer)
	//otel.SetTracerProvider(tracerProvider)
	otel.SetTracerProvider(wrapperTracerProvider)

	// set global propagator to tracecontext (the default is no-op).
	otel.SetTextMapPropagator(propagation.TraceContext{})

	opentracing.SetGlobalTracer(bridgeTracer)
}

func push_metrics(ctx context.Context, payload string) {
	// Get the global MeterProvider and create a new Meter with the name "kbot_light_signal_counter"
	meter := otel.GetMeterProvider().Meter("kbot_command")

	// Get or create an Int64Counter instrument with the name "kbot_light_signal_<payload>"
	counter, _ := meter.Int64Counter(fmt.Sprintf("kbot_command_%s", payload))

	// Add a value of 1 to the Int64Counter
	counter.Add(ctx, 1)
}

// kbotCmd represents the kbot command
var kbotCmd = &cobra.Command{
	Use:     "kbot",
	Aliases: []string{"start"},
	Short:   "Start a bot",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {

		logger := zerodriver.NewProductionLogger()
		logger.Level(zerolog.DebugLevel)

		kbot, err := telebot.NewBot(telebot.Settings{
			URL:    "",
			Token:  TeleToken,
			Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
		})

		if err != nil {
			logger.Fatal().Str("Error", err.Error()).Msg("Plaese check TELE_TOKEN env variable.")
			return
		} else {
			logger.Info().Str("Version", appVersion).Msg("kbot started")
		}

		kbot.Handle(telebot.OnText, func(m telebot.Context) error {
			ctx := context.Background()

			otel_ctx, otel_span := otelTracer.Start(ctx,
				"kbot_message_handler",
				trace.WithAttributes(attribute.String("component", "kbot")),
				trace.WithAttributes(attribute.String("TraceID", trace.TraceID{18, 52, 86, 120, 144}.String())),
			)
			defer otel_span.End()
			otel_trace_id := otel_span.SpanContext().TraceID().String()
			br_ctx := bridgeTracer.ContextWithBridgeSpan(otel_ctx, otel_span)
			otrc_span, otrc_ctx = opentracing.StartSpanFromContext(br_ctx, "kbot_start_span")

			setOtrcSpanAttr := func(operationName string) {
				otrc_span.SetOperationName(fmt.Sprintf("command: %s", operationName))
				otrc_span.LogFields(log.String("event", "start kbot answer"), log.String("type", operationName), log.String("start time", time.Now().String()))
				ext.PeerService.Set(otrc_span, fmt.Sprintf("%s-kbot-message", operationName))
				ext.Component.Set(otrc_span, fmt.Sprintf("%s-kbot-message-handler", operationName))
				ext.SpanKind.Set(otrc_span, ext.SpanKindRPCClientEnum)
				ext.SpanKindRPCClient.Set(otrc_span)
			}

			endOtrcSpan := func() {
				otrc_span.LogFields(log.String("event", "end kbot answer"), log.String("end time", time.Now().String()))
				otrc_span.Finish()
			}

			pushRequest := func(payload string, trace_id string) (string, string) {
				strTime := time.Now()
				push_request(payload)
				endTime := time.Now()
				duration := endTime.Sub(strTime)
				msg_out := fmt.Sprintf("<b>Trace request()</b> start at %s, end at %s\nDuration: %s, TraceID: %s", strTime.Format("15:04:05.123"), endTime.Format("15:04:05.123"), duration, trace_id)
				metric_label := "get"
				return msg_out, metric_label
			}

			payload := m.Message().Payload
			msg_text := m.Text()
			msg_out := ""
			metric_label := "undefined"
			logger.Info().Msgf("Income message: %s, payload: %s", msg_text, payload)
			logger.Info().Msgf("OpenTelemetry traceID=%s", otel_trace_id)

			switch payload {
			case "hello":
				metric_label = "hello"
				setOtrcSpanAttr(metric_label)
				err = m.Send(fmt.Sprintf("<b>Hello, %s</b>\nI'm %s!", m.Sender().FirstName, appVersion), telebot.ModeHTML)
			case "":
				switch msg_text {
				case "/start":
					metric_label = "start"
					setOtrcSpanAttr(metric_label)
					err = m.Send("<b>Usage:</b>\n /help - for help message\n hello - to view 'hello message'\n /get &lt;text&gt; - send a request to an external server\n ding - get 'dong' response", telebot.ModeHTML)
				case "/help":
					metric_label = "help"
					setOtrcSpanAttr(metric_label)
					err = m.Send("NP Kbot help page... be soon")
				case "/hello", "hello":
					metric_label = "hello"
					setOtrcSpanAttr(metric_label)
					err = m.Send(fmt.Sprintf("<b>Hello, %s</b>\nI'm %s!", m.Sender().FirstName, appVersion), telebot.ModeHTML)
				case "ding":
					metric_label = "ding"
					setOtrcSpanAttr(metric_label)
					err = m.Send("dong")
				case "/get":
					setOtrcSpanAttr("get")
					msg_out, metric_label = pushRequest(payload, otel_trace_id)
					err = m.Send(msg_out, telebot.ModeHTML)
				default:
					setOtrcSpanAttr(metric_label)
				}
			default:
				if strings.HasPrefix(msg_text, "/get") {
					setOtrcSpanAttr("get")
					msg_out, metric_label = pushRequest(payload, otel_trace_id)
					err = m.Send(msg_out, telebot.ModeHTML)
				} else {
					setOtrcSpanAttr(metric_label)
					err = m.Send("<b>Usage:</b>\n /help - for help message\n hello - to view 'hello message'\n ding - get 'dong' response", telebot.ModeHTML)
				}
			}
			push_metrics(context.Background(), metric_label)
			endOtrcSpan()
			return err
		})
		kbot.Start()
	},
}

func init() {
	ctx := context.Background()
	initMetrics(ctx)
	initTraces(ctx)
	rootCmd.AddCommand(kbotCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// kbotCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// kbotCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
