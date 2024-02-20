package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// Version can be set at link time to override debug.BuildInfo.Main.Version,
// which is "(devel)" when building from within the module. See
// golang.org/issue/29814 and golang.org/issue/29228.
var Version string

const (
	AppName                = "bingorun"
	BuildInfoRevision      = "vcs.revision"
	DefaultBingoFolder     = ".bingo"
	BingoEnvFile           = "variables.env"
	BingoMkFile            = "Variables.mk"
	InstallCmdRemovePrefix = "\t@"
	InstallCmdPrefix       = InstallCmdRemovePrefix + "cd $(BINGO_DIR) &&"
	Usage                  = `Tool for running 'bingo' managed tools.

Usage:

    bingorun <tool-name> [args...]

It runs the specified tool, and (re)installs the tool if missing.

Example:

    bingorun go-enum --marshal --nocase -f=<file.go>


It could be used in go generate directives, for example:

    //go:generate bingorun go-enum --marshal --nocase -f=$GOFILE

Instead of the tool name, you can use the following commands:

    -b, --bin       print the path of the tool binary
    -v, --version   print the version
    -h, --help      print this help message
`
)

func main() {
	err := run()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

//nolint:gomnd
func run() error {
	if len(os.Args) < 2 {
		showUsage()
		return nil
	}
	var bin bool
	var args []string
	toolName := os.Args[1]
	switch toolName {
	case "-b", "--bin":
		bin = true
		if len(os.Args) < 3 {
			showUsage()
			return nil
		}
		toolName = os.Args[2]
		if len(os.Args) > 3 {
			args = os.Args[3:]
		}
	case "-v", "-V", "--version":
		showVersion()
		return nil
	case "-h", "-H", "--help":
		showUsage()
		return nil
	default:
		if strings.HasPrefix(toolName, "-") {
			showUsage()
			return nil
		}
		if len(os.Args) > 2 {
			args = os.Args[2:]
		}
	}
	toolName = kebabToUpperSnake(toolName)
	var err error
	var path string
	file := os.Getenv("GOFILE") // defined when called from go generate
	switch file {
	case "":
		path, err = os.Getwd()
		if err != nil {
			return err
		}
	default:
		if _, err := os.Stat(file); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("file %s not found", file)
			}
			return err
		}
		path, err = filepath.Abs(file)
		if err != nil {
			return err
		}
		path = filepath.Dir(path)
	}
	envFile, err := findBingoEnvFile(path)
	if err != nil {
		return err
	}
	gobin := gobinFolder()
	tool, err := getEnvValueFromFile(envFile, toolName, gobin)
	if err != nil {
		return err
	}
	if err := installToolIfMissing(envFile, tool, gobin); err != nil {
		return err
	}
	if bin {
		fmt.Println(tool)
		return nil
	}
	return sh.RunV(tool, args...)
}

func showUsage() {
	fmt.Print(Usage)
}

func showVersion() {
	if Version != "" {
		fmt.Printf("Version: %s\n", Version)
		return
	}
	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		fmt.Printf("Version: %s", buildInfo.Main.Version)
		for _, s := range buildInfo.Settings {
			if s.Key != BuildInfoRevision {
				continue
			}
			if s.Value != "" {
				fmt.Printf(" %s", s.Value)
			}
			break
		}
		fmt.Println()
	}
}

func gobinFolder() string {
	gobin := os.Getenv("GOBIN")
	if gobin == "" {
		gobin = filepath.Join(os.Getenv("GOPATH"), "bin")
	}
	return gobin
}

func bingoFolder() string {
	f := os.Getenv("BINGO_DIR")
	f = strings.TrimSuffix(f, "/")
	f = strings.TrimSuffix(f, "\\")
	if f != "" {
		return f
	}
	return DefaultBingoFolder
}

func findBingoEnvFile(path string) (string, error) {
	if path == "" {
		return "", errors.New("undefined path")
	}
	for {
		bf := filepath.Join(path, bingoFolder())
		if f, err := os.Stat(bf); err != nil {
			if os.IsNotExist(err) {
				path += "/.."
				continue
			}
			return "", fmt.Errorf("find bingo folder failed: %w", err)
		} else if !f.IsDir() {
			return "", errors.New("invalid bingo folder, it's not a folder")
		}
		ef := filepath.Join(bf, BingoEnvFile)
		if f, err := os.Stat(ef); err != nil {
			if os.IsNotExist(err) {
				return "", errors.New("bingo environment file not found")
			}
			if f.IsDir() {
				return "", errors.New("invalid bingo environment file, it's a folder")
			}
			return "", fmt.Errorf("find bingo environment file failed: %w", err)
		}
		return ef, nil
	}
}

func findBingoMkFile(folder string) (string, error) {
	if _, err := os.Stat(folder); err != nil {
		return "", err
	}
	mf := filepath.Join(folder, BingoMkFile)
	f, err := os.Stat(mf)
	if err == nil {
		if f.IsDir() {
			return "", errors.New("invalid bingo make file, it's a folder")
		}
		return mf, nil
	}
	if os.IsNotExist(err) {
		return "", errors.New("bingo make file not found")
	}
	return "", err
}

func getEnvValueFromFile(envFile, key, gobin string) (string, error) {
	f, err := os.Open(envFile)
	if err != nil {
		return "", err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		l := scanner.Text()
		if strings.HasPrefix(l, key+"=") {
			p := strings.Split(l, "=")
			if len(p) != 2 { //nolint:gomnd // 2 elements for key and value
				return "", fmt.Errorf("invalid bingo environment variable definition: %s", l)
			}
			v := strings.ReplaceAll(p[1], "\"", "")
			if strings.HasPrefix(v, "${GOBIN}") {
				v = strings.Replace(v, "${GOBIN}", gobin, 1)
			}
			return v, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("bingo environment variable not found: %s", key)
}

func getInstallCmdFromFile(mkFile, toolName string) (string, error) {
	f, err := os.Open(mkFile)
	if err != nil {
		return "", err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		l := scanner.Text()
		if strings.HasPrefix(l, InstallCmdPrefix) && strings.Contains(l, toolName) {
			return strings.Replace(l, InstallCmdRemovePrefix, "", 1), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("install %q command not found", toolName)
}

func installToolIfMissing(envFile, tool, gobin string) error {
	f, err := os.Stat(tool)
	if err == nil {
		if f.IsDir() {
			return fmt.Errorf("invalid tool, it's a folder: %s", tool)
		}
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}
	bingo := filepath.Dir(envFile)
	toolName := filepath.Base(tool)
	mkFile, err := findBingoMkFile(bingo)
	if err != nil {
		return err
	}
	cmd, err := getInstallCmdFromFile(mkFile, toolName)
	if err != nil {
		return err
	}
	cmd = strings.Replace(cmd, "$(BINGO_DIR)", bingo, 1)
	cmd = strings.Replace(cmd, "$(GOBIN)", gobin, 1)
	cmd = strings.Replace(cmd, "$(GO)", mg.GoCmd(), 1)
	return sh.RunV("sh", "-c", cmd)
}

func kebabToUpperSnake(s string) string {
	var sb strings.Builder
	for _, c := range s {
		if c == '-' {
			sb.WriteString("_")
		} else {
			if c >= 'a' && c <= 'z' {
				sb.WriteRune(c - 32) //nolint:gomnd // 'a' - 'A' = 32
			} else {
				sb.WriteRune(c)
			}
		}
	}
	return sb.String()
}
