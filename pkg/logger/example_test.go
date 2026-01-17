package logger_test

import (
	"os"

	"github.com/massonsky/buffalo/pkg/logger"
)

func ExampleNew() {
	// Create a logger with INFO level and colored output
	log := logger.New(
		logger.WithLevel(logger.INFO),
		logger.WithFormatter(logger.NewColoredFormatter()),
		logger.WithOutput(logger.NewStdoutOutput()),
	)

	log.Info("Application started")
	log.Info("Processing request", logger.String("method", "GET"), logger.String("path", "/api/users"))
}

func ExampleLogger_WithFields() {
	log := logger.New(
		logger.WithLevel(logger.INFO),
		logger.WithFormatter(logger.NewTextFormatter()),
		logger.WithOutput(logger.NewStdoutOutput()),
	)

	// Create a child logger with default fields
	requestLogger := log.WithFields(logger.Fields{
		"request_id": "12345",
		"user_id":    "user-1",
	})

	requestLogger.Info("Request received")
	requestLogger.Info("Request processed")
}

func ExampleJSONFormatter() {
	log := logger.New(
		logger.WithLevel(logger.DEBUG),
		logger.WithFormatter(logger.NewJSONFormatter()),
		logger.WithOutput(logger.NewStdoutOutput()),
	)

	log.Info("JSON formatted log", logger.String("key", "value"))
}

func ExampleFileOutput() {
	fileOutput, err := logger.NewFileOutput("logs/app.log")
	if err != nil {
		panic(err)
	}
	defer fileOutput.Close()

	log := logger.New(
		logger.WithLevel(logger.INFO),
		logger.WithFormatter(logger.NewTextFormatter()),
		logger.WithOutput(fileOutput),
	)

	log.Info("Logging to file")
}

func ExampleFileOutput_withRotation() {
	rotation := &logger.RotationConfig{
		MaxSize:    100, // 100 MB
		MaxAge:     7,   // 7 days
		MaxBackups: 5,   // keep 5 old files
		Daily:      true,
	}

	fileOutput, err := logger.NewFileOutputWithRotation("logs/app.log", rotation)
	if err != nil {
		panic(err)
	}
	defer fileOutput.Close()

	log := logger.New(
		logger.WithLevel(logger.INFO),
		logger.WithFormatter(logger.NewJSONFormatter()),
		logger.WithOutput(fileOutput),
	)

	log.Info("Logging with rotation")
}

func ExampleLogger_multipleOutputs() {
	// Log to both console and file
	fileOutput, _ := logger.NewFileOutput("logs/app.log")
	defer fileOutput.Close()

	log := logger.New(
		logger.WithLevel(logger.INFO),
		logger.WithFormatter(logger.NewTextFormatter()),
		logger.WithOutputs(
			logger.NewStdoutOutput(),
			fileOutput,
		),
	)

	log.Info("This goes to both console and file")
}

func Example_differentLevels() {
	log := logger.New(
		logger.WithLevel(logger.DEBUG),
		logger.WithFormatter(logger.NewColoredFormatter()),
		logger.WithOutput(logger.NewConsoleOutput(os.Stdout)),
	)

	log.Debug("Debug message")
	log.Info("Info message")
	log.Warn("Warning message")
	log.Error("Error message")
}

func Example_structuredLogging() {
	log := logger.New(
		logger.WithLevel(logger.INFO),
		logger.WithFormatter(logger.NewJSONFormatter()),
		logger.WithOutput(logger.NewStdoutOutput()),
	)

	log.Info("User action",
		logger.String("action", "login"),
		logger.String("username", "john"),
		logger.String("ip", "192.168.1.1"),
		logger.Int("attempt", 1),
		logger.Bool("success", true),
	)
}
