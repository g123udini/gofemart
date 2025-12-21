package main

import (
	"flag"
	"log"
	"os"
	_ "reflect"
	"testing"

	"github.com/caarlos0/env"
)

var fatal = log.Fatal
var envParse = env.Parse

func TestParseFlags_Defaults(t *testing.T) {
	oldArgs := os.Args
	oldCmd := flag.CommandLine
	oldFatal := fatal
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = oldCmd
		fatal = oldFatal
	}()

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	os.Args = []string{"cmd"}

	fatal = func(v ...any) { t.Fatalf("fatal called: %v", v) }

	f := parseFlags()

	if f.RunAddr != ":8080" {
		t.Fatalf("RunAddr=%q want=%q", f.RunAddr, ":8080")
	}
	if f.Dsn != "postgres://dev:dev@localhost:5432/dev" {
		t.Fatalf("Dsn=%q", f.Dsn)
	}
	if f.AccrualAddress != "http://localhost:8080/accrual" {
		t.Fatalf("AccrualAddress=%q", f.AccrualAddress)
	}
}

func TestParseFlags_OverridesByCLI(t *testing.T) {
	oldArgs := os.Args
	oldCmd := flag.CommandLine
	oldFatal := fatal
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = oldCmd
		fatal = oldFatal
	}()

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	os.Args = []string{
		"cmd",
		"-a", "127.0.0.1:9999",
		"-d", "postgres://u:p@localhost:5432/x",
		"-r", "http://accrual:8081",
	}

	fatal = func(v ...any) { t.Fatalf("fatal called: %v", v) }

	f := parseFlags()

	if f.RunAddr != "127.0.0.1:9999" {
		t.Fatalf("RunAddr=%q", f.RunAddr)
	}
	if f.Dsn != "postgres://u:p@localhost:5432/x" {
		t.Fatalf("Dsn=%q", f.Dsn)
	}
	if f.AccrualAddress != "http://accrual:8081" {
		t.Fatalf("AccrualAddress=%q", f.AccrualAddress)
	}
}

func TestParseFlags_OverridesByEnv(t *testing.T) {
	oldArgs := os.Args
	oldCmd := flag.CommandLine
	oldFatal := fatal
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = oldCmd
		fatal = oldFatal
	}()

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	os.Args = []string{"cmd"}

	t.Setenv("RUN_ADDRESS", "0.0.0.0:7777")
	t.Setenv("DATABASE_URI", "postgres://env:env@localhost:5432/envdb")
	t.Setenv("ACCRUAL_SYSTEM_ADDRESS", "http://env-accrual:9999")

	fatal = func(v ...any) { t.Fatalf("fatal called: %v", v) }

	f := parseFlags()

	if f.RunAddr != "0.0.0.0:7777" {
		t.Fatalf("RunAddr=%q", f.RunAddr)
	}
	if f.Dsn != "postgres://env:env@localhost:5432/envdb" {
		t.Fatalf("Dsn=%q", f.Dsn)
	}
	if f.AccrualAddress != "http://env-accrual:9999" {
		t.Fatalf("AccrualAddress=%q", f.AccrualAddress)
	}
}

func TestParseFlags_EnvWinsOverCLI(t *testing.T) {
	oldArgs := os.Args
	oldCmd := flag.CommandLine
	oldFatal := fatal
	defer func() {
		os.Args = oldArgs
		flag.CommandLine = oldCmd
		fatal = oldFatal
	}()

	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	os.Args = []string{
		"cmd",
		"-a", "127.0.0.1:9999",
		"-d", "postgres://cli:cli@localhost:5432/cli",
		"-r", "http://cli-accrual:1",
	}

	t.Setenv("RUN_ADDRESS", "env:2")
	t.Setenv("DATABASE_URI", "postgres://env:env@localhost:5432/env")
	t.Setenv("ACCRUAL_SYSTEM_ADDRESS", "http://env-accrual:3")

	fatal = func(v ...any) { t.Fatalf("fatal called: %v", v) }

	f := parseFlags()

	if f.RunAddr != "env:2" || f.Dsn != "postgres://env:env@localhost:5432/env" || f.AccrualAddress != "http://env-accrual:3" {
		t.Fatalf("unexpected flags: %+v", f)
	}
}
