package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"txp/internal/config"
	"txp/internal/player"
	"txp/internal/state"
	"txp/internal/storage"
	"txp/internal/ui"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

func main() {
	configPath := flag.String("config-path", "", "Path to config file")
	flag.Parse()

	var resolved config.ResolvedConfig
	var configErr error
	if *configPath != "" {
		resolvedPath := resolveConfigPath(*configPath)
		loaded, err := config.LoadWithPath(resolvedPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Config warning:", err)
			configErr = err
		}
		resolved = loaded
	} else {
		loaded, err := config.Load()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Config warning:", err)
			configErr = err
		}
		resolved = loaded
	}
	var db *storage.Database
	if *configPath != "" {
		dbPath := filepath.Join(filepath.Dir(resolved.Path), "txp.db")
		database, err := storage.OpenWithPath(dbPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Database error:", err)
			os.Exit(1)
		}
		db = database
	} else {
		database, err := storage.Open()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Database error:", err)
			os.Exit(1)
		}
		db = database
	}
	defer func() {
		_ = db.DB.Close()
	}()

	tasks := state.NewTaskManager()
	playerClient, err := player.Start()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Player error:", err)
	}
	app := ui.NewApp(resolved.Config, resolved.Path, resolved.Scope, db.Path, db, tasks, playerClient)
	if logger := initZapLogger(resolved.Path); logger != nil {
		app.SetLogger(logger)
		defer func() { _ = logger.Sync() }()
	}
	if configErr != nil {
		app.AppendLog("warn", fmt.Sprintf("Config warning: %v", configErr))
	}
	if totals, ok, err := storage.GetTotals(db.DB); err == nil {
		if ok {
			app.SetStatsTotals(totals, true)
		} else if computed, err := storage.ComputeTotals(db.DB); err == nil {
			_ = storage.SaveTotals(db.DB, computed)
			app.SetStatsTotals(computed, true)
		}
	}
	err = app.Run()
	clearScreen()
	if err != nil {
		fmt.Fprintln(os.Stderr, "App error:", err)
		os.Exit(1)
	}
	if playerClient != nil {
		playerClient.Close()
	}
}

func clearScreen() {
	_, _ = os.Stdout.WriteString("\033[H\033[2J")
}

func resolveConfigPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return trimmed
	}
	if strings.HasSuffix(trimmed, "/") || strings.HasSuffix(trimmed, "\\") {
		return filepath.Join(trimmed, "config.json")
	}
	info, err := os.Stat(trimmed)
	if err == nil && info.IsDir() {
		return filepath.Join(trimmed, "config.json")
	}
	return trimmed
}

func initZapLogger(configPath string) *zap.Logger {
	logPath := strings.TrimSpace(os.Getenv("TXP_LOG_PATH"))
	if logPath == "" {
		if configPath != "" {
			logPath = filepath.Join(filepath.Dir(configPath), "txp.log")
		} else {
			if home, err := os.UserHomeDir(); err == nil {
				logPath = filepath.Join(home, ".txp", "txp.log")
			}
		}
	}
	if logPath == "" {
		return nil
	}

	level := parseZapLevel(strings.TrimSpace(os.Getenv("TXP_LOG_LEVEL")))
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoder := zapcore.NewConsoleEncoder(encoderConfig)
	writer := zapcore.AddSync(&lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    10,
		MaxBackups: 1,
		Compress:   false,
	})
	core := zapcore.NewCore(encoder, writer, level)
	return zap.New(core)
}

func parseZapLevel(level string) zapcore.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.ErrorLevel
	}
}
