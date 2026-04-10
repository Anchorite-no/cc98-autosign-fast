package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	initConsole()

	pauseOnExit := shouldPauseOnExit()
	exitCode := runApp(pauseOnExit)
	if pauseOnExit {
		waitForExit()
	}
	os.Exit(exitCode)
}

func runApp(pauseOnExit bool) int {
	defaultEnvPath := defaultEnvPath()
	envPath := flag.String("env", defaultEnvPath, "Path to .env file")
	showTiming := flag.Bool("timing", false, "Show per-request timings.")
	flag.Parse()

	templateCreated, err := EnsureEnvFile(*envPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "配置文件准备失败\n%s\n", err)
		return 2
	}

	cfg, err := LoadConfig(*envPath)
	if err != nil {
		printConfigError(err, templateCreated, pauseOnExit)
		return 2
	}

	runner := NewRunner(cfg, os.Stdout, *showTiming)
	if err := runner.Run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "运行失败\n%s\n", err)
		return 1
	}
	return 0
}

func printConfigError(err error, templateCreated, pauseOnExit bool) {
	var validationErr *ConfigValidationError
	if errors.As(err, &validationErr) {
		fmt.Fprintln(os.Stdout, "配置未完成")
		if len(validationErr.MissingFields) > 0 {
			fmt.Fprintf(os.Stdout, "缺少: %s\n", strings.Join(validationErr.MissingFields, ", "))
		}
		for _, message := range validationErr.InvalidMessages {
			fmt.Fprintln(os.Stdout, message)
		}
		if templateCreated {
			if pauseOnExit {
				fmt.Fprintln(os.Stdout, "已生成 .env，请填写后重新双击运行")
			} else {
				fmt.Fprintln(os.Stdout, "已生成 .env，请填写后重新运行")
			}
		} else if pauseOnExit {
			fmt.Fprintln(os.Stdout, "请填写完成后重新双击运行")
		} else {
			fmt.Fprintln(os.Stdout, "请填写完成后重新运行")
		}
		return
	}

	fmt.Fprintln(os.Stdout, "配置文件有误")
	fmt.Fprintln(os.Stdout, err)
	if templateCreated {
		if pauseOnExit {
			fmt.Fprintln(os.Stdout, "已生成 .env，请填写后重新双击运行")
		} else {
			fmt.Fprintln(os.Stdout, "已生成 .env，请填写后重新运行")
		}
	}
}

func defaultEnvPath() string {
	exePath, err := os.Executable()
	if err != nil {
		return ".env"
	}
	return filepath.Join(filepath.Dir(exePath), ".env")
}
