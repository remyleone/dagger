package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// TODO: remove these ghost commands after next release (v0.9.8)
var downloadCmd = &cobra.Command{
	Use:                "download",
	Short:              "Download an asset from module function",
	Aliases:            []string{"export", "dl"},
	Hidden:             true,
	SilenceUsage:       true,
	DisableFlagParsing: true,
	Args: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf(`%q has been replaced by "dagger call COMMANDS... --output=PATH"`, cmd.CommandPath())
	},
	Run: func(cmd *cobra.Command, args []string) {
		// do nothing
	},
}

var upCmd = &cobra.Command{
	Use:                "up",
	Short:              "Start a service and expose its ports to the host",
	Hidden:             true,
	SilenceUsage:       true,
	DisableFlagParsing: true,
	Args: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf(`%q has been replaced by "dagger call COMMANDS... up"`, cmd.CommandPath())
	},
	Run: func(cmd *cobra.Command, args []string) {
		// do nothing
	},
}

var shellCmd = &cobra.Command{
	Use:                "shell",
	Short:              "Open a shell in a container",
	Hidden:             true,
	SilenceUsage:       true,
	DisableFlagParsing: true,
	Args: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf(`%q has been replaced by "dagger call COMMANDS... shell"`, cmd.CommandPath())
	},
	Run: func(cmd *cobra.Command, args []string) {
		// do nothing
	},
}

var outputPath string
var jsonOutput bool

var callCmd = &FuncCommand{
	Name:  "call",
	Short: "Call a module function",
	Long: `Call a module function and print the result

If the last argument is either Container, Directory, or File, the pipeline
will be evaluated (the result of calling *sync*) without presenting any output.
Providing the --output option (shorthand: -o) is equivalent to calling *export*
instead. To print a property of these core objects, continue chaining by
appending it to the end of the command (for example, *stdout*, *entries*, or
*contents*).
`,
	Init: func(cmd *cobra.Command) {
		cmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Present result as JSON")
		cmd.PersistentFlags().StringVarP(&outputPath, "output", "o", "", "Path in the host to save the result to")
	},
	OnSelectObjectLeaf: func(c *FuncCommand, name string) error {
		switch name {
		case Container, Directory, File:
			if outputPath != "" {
				c.Select("export")
				c.Arg("path", outputPath)
				if name == File {
					c.Arg("allowParentDirPath", true)
				}
				return nil
			}
			c.Select("sync")
		case Terminal:
			c.Select("websocketEndpoint")
		default:
			return fmt.Errorf("return type %q requires a sub-command", name)
		}
		return nil
	},
	BeforeRequest: func(_ *FuncCommand, _ *cobra.Command, modType *modTypeDef) error {
		if modType.Name() != Terminal {
			return nil
		}

		// Even though these flags are global, we only check them just before query
		// execution because you may want to debug an error during loading or for
		// --help.
		if silent || !(progress == "auto" && autoTTY || progress == "tty") {
			return fmt.Errorf("running shell without the TUI is not supported")
		}
		if debug {
			return fmt.Errorf("running shell with --debug is not supported")
		}
		if outputPath != "" {
			return fmt.Errorf("running shell with --output is not supported")
		}
		return nil
	},
	AfterResponse: func(c *FuncCommand, cmd *cobra.Command, modType *modTypeDef, response any) error {
		switch modType.Name() {
		case Terminal:
			termEndpoint, ok := response.(string)
			if !ok {
				return fmt.Errorf("unexpected response %T: %+v", response, response)
			}
			return attachToShell(cmd.Context(), c.c, termEndpoint)
		case Container, Directory, File:
			if outputPath != "" {
				logOutputSuccess(cmd, outputPath)
				return nil
			}

			// Just `sync`, don't print the result (id), but let user know.

			// TODO: This is only "needed" when there's no output because
			// you're left wondering if the command did anything. Otherwise,
			// the output is sent only to progrock (TUI), so we'd need to check
			// there if possible. Decide whether this message is ok in all cases,
			// better to not print it, or to conditionally check.
			cmd.PrintErrf("%s evaluated. Use \"%s --help\" to see available sub-commands.\n", modType.Name(), cmd.CommandPath())
			return nil
		default:
			// TODO: Since IDs aren't stable to be used in the CLI, we should
			// silence all ID results (or present in a compact way like
			// ´<ContainerID:etpdi9gue9l5>`), but need a KindScalar TypeDef
			// to get the name from modType.
			// You can't select `id`, but you can select `sync`, and there
			// may be others.
			buf := new(bytes.Buffer)

			// especially useful for lists and maps
			if jsonOutput {
				// disable HTML escaping to improve readability
				encoder := json.NewEncoder(buf)
				encoder.SetEscapeHTML(false)
				encoder.SetIndent("", "    ")
				if err := encoder.Encode(response); err != nil {
					return err
				}
			} else {
				if err := printFunctionResult(buf, response); err != nil {
					return err
				}
			}

			if outputPath != "" {
				if err := writeOutputFile(outputPath, buf); err != nil {
					return fmt.Errorf("couldn't write output to file: %w", err)
				}
				logOutputSuccess(cmd, outputPath)
			}

			writer := cmd.OutOrStdout()
			buf.WriteTo(writer)

			// TODO(vito) right now when stdoutIsTTY we'll be printing to a Progrock
			// vertex, which currently adds its own linebreak (as well as all the
			// other UI clutter), so there's no point doing this. consider adding
			// back when we switch to printing "clean" output on exit.
			// if stdoutIsTTY && !strings.HasSuffix(buf.String(), "\n") {
			// 	fmt.Fprintln(writer, "⏎")
			// }

			return nil
		}
	},
}

// writeOutputFile writes the buffer to a file, creating the parent directories
// if needed.
func writeOutputFile(path string, buf *bytes.Buffer) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { // nolint: gosec
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644) // nolint: gosec
}

// logOutputSuccess prints to stderr the the output path to the user.
func logOutputSuccess(cmd *cobra.Command, path string) {
	path, err := filepath.Abs(path)
	if err != nil {
		// don't fail because at this point the output has been saved successfully
		cmd.PrintErrf("WARNING: failed to get absolute path: %s\n", err)
		path = outputPath
	}
	cmd.PrintErrf("Saved output to %q.\n", path)
}

func printFunctionResult(w io.Writer, r any) error {
	switch t := r.(type) {
	case []any:
		// TODO: group in progrock
		for _, v := range t {
			if err := printFunctionResult(w, v); err != nil {
				return err
			}
			fmt.Fprintln(w)
		}
		return nil
	case map[string]any:
		// NB: we're only interested in values because this is where we unwrap
		// things like {"container":{"from":{"withExec":{"stdout":"foo"}}}}.
		for _, v := range t {
			if err := printFunctionResult(w, v); err != nil {
				return err
			}
		}
		return nil
	case string:
		fmt.Fprint(w, t)
	default:
		fmt.Fprintf(w, "%+v", t)
	}
	return nil
}
