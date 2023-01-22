package metrics

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Releem/mysqlconfigurer/config"
	"github.com/advantageous/go-logback/logging"
)

var Ready bool

// Set up channel on which to send signal notifications.
// We must use a buffered channel or risk missing the signal
// if we're not ready to receive when the signal is sent.
func makeTerminateChannel() <-chan os.Signal {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	return ch
}

func RunWorker(gatherers []MetricsGatherer, repeaters map[string][]MetricsRepeater, logger logging.Logger,
	configuration *config.Config, configFile string, FirstRun bool) {
	var GenerateTimer *time.Timer
	if logger == nil {
		if configuration.Debug {
			logger = logging.NewSimpleDebugLogger("Worker")
		} else {
			logger = logging.NewSimpleLogger("Worker")
		}
	}

	timer := time.NewTimer(1 * time.Second)
	configTimer := time.NewTimer(configuration.ReadConfigSeconds * time.Second)
	if FirstRun {
		GenerateTimer = time.NewTimer(0 * time.Second)
	} else {
		GenerateTimer = time.NewTimer(configuration.GenerateConfigSeconds * time.Second)
	}

	// var configuration *config.Config
	// if newConfig, err := config.LoadConfig(configFile, logger); err != nil {
	// 	logger.PrintError("Error reading config", err)
	// } else {
	// 	configuration = newConfig
	// }
	terminator := makeTerminateChannel()

	for {
		select {
		case <-terminator:
			logger.Info("Exiting")
			os.Exit(0)
		case <-timer.C:
			Ready = false
			timer.Reset(configuration.TimePeriodSeconds * time.Second)
			metrics := collectMetrics(gatherers, logger)
			if Ready {
				processMetrics(metrics, repeaters, configuration, logger)
			}

		case <-configTimer.C:
			configTimer.Reset(configuration.ReadConfigSeconds * time.Second)
			if newConfig, err := config.LoadConfig(configFile, logger); err != nil {
				logger.PrintError("Error reading config", err)
			} else {
				configuration = newConfig
				logger.Debug("LOADED NEW CONFIG", "APIKEY", configuration.GetApiKey())
			}

		case <-GenerateTimer.C:
			Ready = false
			logger.Println(" * Collecting metrics to recommend a config...")
			metrics := collectMetrics(gatherers, logger)
			if Ready {
				processConfigurations(metrics, repeaters, configuration, logger)
			}
			if FirstRun {
				os.Exit(0)
			}
			// logger.Println("Generating the recommended configuration")
			// cmd := exec.Command("/bin/bash", "/opt/releem/mysqlconfigurer.sh")
			// cmd.Env = os.Environ()
			// cmd.Env = append(cmd.Env, "PATH=/bin:/sbin:/usr/bin:/usr/sbin")
			// stdout, err := cmd.Output()
			// if err != nil {
			// 	logger.PrintError("Config generation with error", err)
			// }
			// logger.Debug(string(stdout))
			GenerateTimer.Reset(configuration.GenerateConfigSeconds * time.Second)
		}
	}
}

func processMetrics(metrics Metrics, repeaters map[string][]MetricsRepeater,
	configuration *config.Config, logger logging.Logger) {
	for _, r := range repeaters["Metrics"] {
		err := r.ProcessMetrics(configuration, metrics)
		if err != nil {
			logger.PrintError("Repeater failed", err)
		}
	}
}

func processConfigurations(metrics Metrics, repeaters map[string][]MetricsRepeater,
	configuration *config.Config, logger logging.Logger) {
	for _, r := range repeaters["Configurations"] {
		err := r.ProcessMetrics(configuration, metrics)
		if err != nil {
			logger.PrintError("Repeater failed", err)
		}
	}
}

func collectMetrics(gatherers []MetricsGatherer, logger logging.Logger) Metrics {
	var metrics Metrics
	for _, g := range gatherers {
		err := g.GetMetrics(&metrics)
		if err != nil {
			logger.Error("Problem getting metrics from gatherer")
			return Metrics{}
		}
	}
	Ready = true
	return metrics
}