package cadence

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/uber-go/tally"
	"go.uber.org/cadence/.gen/go/cadence/workflowserviceclient"
	"go.uber.org/cadence/worker"
	"go.uber.org/yarpc"
	"go.uber.org/yarpc/transport/tchannel"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// BuildCadenceServiceClient constructs a cadence client
func BuildCadenceServiceClient(hostPort, clientName, cadenceService string) workflowserviceclient.Interface {
	ch, err := tchannel.NewChannelTransport(tchannel.ServiceName(clientName))
	if err != nil {
		panic("Failed to setup tchannel")
	}
	dispatcher := yarpc.NewDispatcher(yarpc.Config{
		Name: clientName,
		Outbounds: yarpc.Outbounds{
			cadenceService: {Unary: ch.NewSingleOutbound(hostPort)},
		},
	})
	if err := dispatcher.Start(); err != nil {
		panic("Failed to start dispatcher")
	}

	return workflowserviceclient.New(dispatcher.ClientConfig(cadenceService))
}

// BuildCadenceLogger creates a log instance for cadence client
func BuildCadenceLogger(logLevel int) *zap.Logger {
	config := zap.NewDevelopmentConfig()
	config.Level.SetLevel(zapcore.Level(logLevel))

	var err error
	logger, err := config.Build()
	if err != nil {
		panic("Failed to setup logger")
	}

	return logger
}

// StartWorker starts a cadence worker which watches tasks in the given domain and task list
func StartWorker(logger *zap.Logger, service workflowserviceclient.Interface, domain, taskListName string) {
	// TaskListName identifies set of client workflows, activities, and workers.
	// It could be your group or client or application name.
	workerOptions := worker.Options{
		Logger:       logger,
		MetricsScope: tally.NewTestScope(taskListName, map[string]string{}),
	}

	worker := worker.New(
		service,
		domain,
		taskListName,
		workerOptions)

	if err := worker.Start(); err != nil {
		logger.Panic("Failed to start worker", zap.Error(err))
	}

	logger.Info("Started Worker.", zap.String("worker", taskListName))
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	logger.Info("Server is preparing to shutdown")
	worker.Stop()
}
