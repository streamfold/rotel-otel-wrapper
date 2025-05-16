package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"syscall"

	"github.com/streamfold/rotel-otel-wrapper/internal/config"
)

type dbgFileType interface {
	io.WriteCloser
	io.StringWriter
}

func main() {
	// Get all command line arguments
	args := os.Args

	if len(args) <= 2 {
		log.Fatalf("Usage: rotel-otel-wrapper --config <path to config>")
	}

	rotelPath := os.Getenv("ROTEL_PATH")

	// Get the ROTEL_PATH environment variable
	if rotelPath == "" {
		log.Fatalf("ROTEL_PATH environment variable not set")
	}

	// Check for --config argument and process the config file
	configPath := ""
	for i := 1; i < len(args); i++ {
		if args[i] == "--config" && i+1 < len(args) {
			configPath = args[i+1]
			break
		}
	}

	dbgFile := os.Getenv("ROTEL_WRAPPER_DEBUG_FILE")
	var df dbgFileType

	// Open the file for appending (create it if it doesn't exist)
	if dbgFile != "" {
		f, err := os.OpenFile(dbgFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalf("Error opening file %s: %v", dbgFile, err)
		}
		df = f

		defer func() {
			if df != nil {
				_ = df.Close()
			}
		}()
	} else {
		df = nopWriteCloser{io.Discard}
	}

	// Log all arguments to the file
	argsStr := strings.Join(args, " ")
	if _, err := df.WriteString(argsStr + "\n"); err != nil {
		log.Fatalf("Error writing to file: %v", err)
	}

	if configPath == "" {
		log.Fatalf("Can not find --config path")
	}

	// Read the config file
	configData, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalf("Error reading config file: %v", err)
	}

	// Append the config file contents to args.out
	if _, err := df.WriteString("--- Config file contents ---\n"); err != nil {
		log.Fatalf("Error writing to file: %v", err)
	}
	if _, err := df.Write(configData); err != nil {
		log.Fatalf("Error writing to file: %v", err)
	}
	if _, err := df.WriteString("\n--- End of config file ---\n"); err != nil {
		log.Fatalf("Error writing to file: %v", err)
	}

	conf := config.ReadConfig(configPath)
	cmdArgs := []string{"start", "--exporter", "otlp"}

	if grpc := conf.Receivers.OTLP.Protocols.GRPC; grpc != nil {
		cmdArgs = append(cmdArgs, "--otlp-grpc-endpoint", grpc.Endpoint)
		cmdArgs = append(cmdArgs, "--otlp-http-endpoint", "localhost:0")
	} else {
		if http := conf.Receivers.OTLP.Protocols.HTTP; http != nil {
			cmdArgs = append(cmdArgs, "--otlp-http-endpoint", http.Endpoint)
			cmdArgs = append(cmdArgs, "--otlp-grpc-endpoint", "localhost:0")
		} else {
			log.Fatalf("can not find receiver configuration")
		}
	}

	if otlp := conf.Exporters.OTLP; otlp != nil {
		cmdArgs = append(cmdArgs, "--otlp-exporter-endpoint", endpointWithScheme(otlp))
		cmdArgs = append(cmdArgs, "--otlp-exporter-protocol", "grpc")
		cmdArgs = append(cmdArgs, "--otlp-exporter-compression", otlp.Compression)
	} else {
		if otlphttp := conf.Exporters.OTLPHTTP; otlphttp != nil {
			cmdArgs = append(cmdArgs, "--otlp-exporter-endpoint", endpointWithScheme(otlphttp))
			cmdArgs = append(cmdArgs, "--otlp-exporter-protocol", "http")
			cmdArgs = append(cmdArgs, "--otlp-exporter-compression", otlphttp.Compression)
		} else {
			log.Fatalf("can not find exporter configuration")
		}
	}

	hasBatch := false
	for _, pipeline := range conf.Service.Pipelines {
		// XXX: There's usually only one pipeline, but check if any have a batch processor
		if slices.Contains(pipeline.Processors, "batch") {
			hasBatch = true
		}
	}

	// If there's no batch, disable it
	if !hasBatch {
		cmdArgs = append(cmdArgs, "--disable-batching")
	}

	// Get the program name
	programName := filepath.Base(rotelPath)

	// Create command for execution
	binary, lookErr := exec.LookPath(rotelPath)
	if lookErr != nil {
		log.Fatalf("Error finding executable: %v\n", lookErr)
	}

	execArgs := []string{programName}
	execArgs = append(execArgs, cmdArgs...)

	_, _ = df.WriteString(fmt.Sprintf("Executing rotel at: %v (%v)\n", rotelPath, execArgs))

	_ = df.Close()
	df = nil

	execErr := syscall.Exec(binary, execArgs, os.Environ())
	if execErr != nil {
		log.Fatalf("Error executing %s: %v\n", rotelPath, execErr)
	}
}

func endpointWithScheme(otlp *config.OTLPExporterConfig) string {
	endpoint := otlp.Endpoint
	if strings.HasPrefix(endpoint, "http://") {
		return endpoint
	}

	if strings.HasPrefix(endpoint, "https://") || !otlp.TLS.Insecure {
		panic("https endpoints not supported")
	}

	return fmt.Sprintf("http://%s", endpoint)
}

type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error {
	return nil
}

func (nopWriteCloser) WriteString(s string) (n int, err error) {
	return len(s), nil
}
