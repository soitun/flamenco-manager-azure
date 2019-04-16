package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"gitlab.com/blender-institute/azure-go-test/azbatch"
	"gitlab.com/blender-institute/azure-go-test/azconfig"
	"gitlab.com/blender-institute/azure-go-test/azresource"
	"gitlab.com/blender-institute/azure-go-test/azstorage"
	"gitlab.com/blender-institute/azure-go-test/azvm"
)

const applicationName = "Azure Go Test"

var applicationVersion = "1.0"

// Components that make up the application

var cliArgs struct {
	version        bool
	quiet          bool
	debug          bool
	showStartupCLI bool

	vmName string
}

func parseCliArgs() {
	flag.BoolVar(&cliArgs.version, "version", false, "Shows the application version, then exits.")
	flag.BoolVar(&cliArgs.quiet, "quiet", false, "Disable info-level logging (so warning/error only).")
	flag.BoolVar(&cliArgs.debug, "debug", false, "Enable debug-level logging.")
	flag.BoolVar(&cliArgs.showStartupCLI, "startupCLI", false, "Just show the startup task CLI, do not start the pool.")
	flag.StringVar(&cliArgs.vmName, "vm", "", "Name of the virtual machine to use. If not given, it will be prompted for.")
	flag.Parse()
}

func configLogging() {
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// Only log the warning severity or above by default.
	level := logrus.InfoLevel
	if cliArgs.debug {
		level = logrus.DebugLevel
	} else if cliArgs.quiet {
		level = logrus.WarnLevel
	}
	logrus.SetLevel(level)
	log.SetOutput(logrus.StandardLogger().Writer())
}

func logStartup() {
	level := logrus.GetLevel()
	defer logrus.SetLevel(level)

	logrus.SetLevel(logrus.InfoLevel)
	logrus.WithFields(logrus.Fields{
		"version": applicationVersion,
	}).Infof("Starting %s", applicationName)
}

func main() {
	parseCliArgs()
	if cliArgs.version {
		fmt.Println(applicationVersion)
		return
	}

	configLogging()
	logStartup()

	ctx, cancelCtx := context.WithCancel(context.Background())

	// Handle Ctrl+C
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	go func() {
		for signum := range c {
			logrus.WithField("signal", signum).Info("Signal received, shutting down.")
			cancelCtx()
			time.Sleep(1 * time.Second)
			os.Exit(2)
		}
	}()

	config := azconfig.Load()
	if cliArgs.showStartupCLI {
		ctx, cancel := context.WithDeadline(ctx, time.Now().Add(1*time.Minute))
		defer cancel()

		poolParams := azbatch.PoolParameters()
		withCreds := azstorage.ReplaceAccountDetails(ctx, config, poolParams)
		fmt.Println(*withCreds.StartTask.CommandLine)
		logrus.Info("shutting down after logging account storage key stuff")
		return
	}

	azresource.EnsureResourceGroup(ctx, &config)
	azstorage.EnsureAccount(ctx, &config)
	azbatch.EnsureAccount(ctx, &config)

	vmName, vmExists := azvm.ChooseVM(ctx, config, cliArgs.vmName)
	vm, publicIP := azvm.EnsureVM(ctx, config, vmName, vmExists)
	address := *publicIP.IPAddress
	logrus.WithFields(logrus.Fields{
		"vmName":  *vm.Name,
		"address": address,
	}).Info("found VM public address")

	// azbatch.CreatePool(config)

	cancelCtx()
}
