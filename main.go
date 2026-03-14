package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/rjvkn/curli/args"
	"github.com/rjvkn/curli/formatter"
	"github.com/rjvkn/curli/internal"
	"golang.org/x/term"
)

func main() {
	os.Exit(run(os.Args, os.Stdin, os.Stdout, os.Stderr))
}

type terminalState struct {
	stdin  bool
	stdout bool
	stderr bool
}

func run(argv []string, stdinFile, stdoutFile, stderrFile *os.File) int {
	if len(argv) == 2 && argv[1] == "version" {
		fmt.Printf("curli %s (%s)\n", internal.VERSION, internal.DATE)
		return 0
	}

	terminals := terminalState{
		stdin:  term.IsTerminal(int(stdinFile.Fd())),
		stdout: term.IsTerminal(int(stdoutFile.Fd())),
		stderr: term.IsTerminal(int(stderrFile.Fd())),
	}

	scheme := formatter.DefaultColorScheme
	if err := setupWindowsConsole(int(stdoutFile.Fd())); err != nil {
		scheme = formatter.ColorScheme{}
	}

	opts := args.Parse(argv)
	pretty := opts.Remove("pretty")
	verbose := opts.Has("v") || opts.Has("verbose")

	opts.Remove("i")

	var (
		stdin  io.Reader = stdinFile
		stdout io.Writer = stdoutFile
		stderr io.Writer = stderrFile
		input            = &bytes.Buffer{}
	)

	opts = normalizeOpts(opts)
	switch argv[0] {
	case "post", "put", "delete":
		if !opts.Has("X") && !opts.Has("request") {
			opts = append(opts, "-X", argv[0])
		}
	case "head":
		if !opts.Has("I") && !opts.Has("head") {
			opts = append(opts, "-I")
		}
	}

	if opts.Has("h") || opts.Has("help") {
		stdout = &formatter.HelpAdapter{Out: stdout, CmdName: argv[0]}
	} else {
		if terminals.stdout {
			stdout = &formatter.BinaryFilter{Out: stdout}
		}

		if pretty || terminals.stderr {
			stderr = &formatter.HeaderColorizer{Out: stderr, Scheme: scheme}
		}

		if data := opts.Val("d"); data != "" {
			input.Write([]byte(data))
		} else if !terminals.stdin {
			opts = append(opts, "-d@-")
			stdin = io.TeeReader(stdin, input)
		}
	}

	if opts.Has("curl") {
		opts.Remove("curl")
		fmt.Print("curl")
		for _, opt := range opts {
			if strings.Contains(opt, " ") {
				fmt.Printf(" %q", opt)
			} else {
				fmt.Printf(" %s", opt)
			}
		}
		fmt.Println()
		return 0
	}

	response := executeCurl(opts, stdin, verbose, requestPreview(input.Bytes(), opts, scheme, pretty || terminals.stderr), terminals.stdout)

	io.Copy(stderr, &response.stderrBuf)
	io.Copy(stderr, &response.headerBuf)
	writeResponseBody(stdout, response.outBuf.Bytes(), detectContentType(response.headerBuf.Bytes()), pretty || terminals.stdout, terminals.stdout, scheme)
	return response.status
}

func normalizeOpts(opts args.Opts) args.Opts {
	if len(opts) == 0 {
		return append(opts, "-h", "all")
	}
	return append(opts, "-s", "-S")
}

type curlResponse struct {
	outBuf    bytes.Buffer
	stderrBuf bytes.Buffer
	headerBuf bytes.Buffer
	status    int
}

func executeCurl(opts args.Opts, stdin io.Reader, verbose bool, postPreview []byte, discardStdout bool) curlResponse {
	headerFile, err := os.CreateTemp("", "curli-headers-*")
	if err != nil {
		return curlResponse{status: 1}
	}
	headerPath := headerFile.Name()
	headerFile.Close()
	defer os.Remove(headerPath)

	curlOpts := append(args.Opts{}, opts...)
	curlOpts = append(curlOpts, "--dump-header", headerPath)

	cmd := exec.Command("curl", curlOpts...)
	cmd.Stdin = stdin

	var response curlResponse
	cmd.Stderr = &formatter.HeaderCleaner{
		Out:                 &response.stderrBuf,
		Verbose:             verbose,
		DropResponseHeaders: true,
		Post:                postPreview,
	}

	if (opts.Has("I") || opts.Has("head")) && discardStdout {
		cmd.Stdout = io.Discard
	} else {
		cmd.Stdout = &response.outBuf
	}

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if ws, ok := exitErr.ProcessState.Sys().(syscall.WaitStatus); ok {
				response.status = ws.ExitStatus()
			}
		} else {
			fmt.Fprint(&response.stderrBuf, err)
			response.status = 1
		}
	}

	if headerBytes, err := os.ReadFile(headerPath); err == nil {
		response.headerBuf.Write(headerBytes)
	} else if response.status == 0 {
		fmt.Fprintf(&response.stderrBuf, "curli: failed to read response headers: %v\n", err)
		response.status = 1
	}

	return response
}

func detectContentType(headers []byte) string {
	for _, line := range strings.Split(string(headers), "\n") {
		if strings.HasPrefix(strings.ToLower(line), "content-type:") {
			value := strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
			mediaType, _, err := mime.ParseMediaType(value)
			if err != nil {
				return strings.ToLower(value)
			}
			return strings.ToLower(mediaType)
		}
	}
	return ""
}

func isJSONContentType(contentType string) bool {
	return contentType == "application/json" || strings.HasSuffix(contentType, "+json")
}

func requestPreview(body []byte, opts args.Opts, scheme formatter.ColorScheme, colorize bool) []byte {
	if len(body) == 0 || !isJSONContentType(requestContentType(opts)) {
		return body
	}

	previewScheme := formatter.ColorScheme{}
	if colorize {
		previewScheme = scheme
	}
	formatted, err := formatter.FormatJSON(body, previewScheme)
	if err != nil {
		return body
	}
	return bytes.TrimRight(formatted, "\n")
}

func requestContentType(opts args.Opts) string {
	for _, h := range append(opts.Vals("H"), opts.Vals("header")...) {
		name, value, ok := strings.Cut(h, ":")
		if !ok || !strings.EqualFold(name, "Content-Type") {
			continue
		}
		mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(value))
		if err != nil {
			return strings.ToLower(strings.TrimSpace(value))
		}
		return strings.ToLower(mediaType)
	}
	return ""
}

func writeResponseBody(stdout io.Writer, body []byte, contentType string, pretty bool, colorize bool, scheme formatter.ColorScheme) {
	if !pretty || !shouldFormatJSONBody(body, contentType) {
		stdout.Write(body)
		return
	}

	bodyScheme := formatter.ColorScheme{}
	if colorize {
		bodyScheme = scheme
	}

	formatted, err := formatter.FormatJSON(body, bodyScheme)
	if err != nil {
		stdout.Write(body)
		return
	}
	stdout.Write(formatted)
}

func shouldFormatJSONBody(body []byte, contentType string) bool {
	if isJSONContentType(contentType) {
		return true
	}
	return json.Valid(bytes.TrimSpace(body))
}
