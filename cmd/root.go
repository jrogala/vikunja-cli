package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var outputJSON bool

var rootCmd = &cobra.Command{
	Use:   "vikunja-cli",
	Short: "CLI client for Vikunja task management",
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&outputJSON, "json", false, "output raw JSON")
	rootCmd.SetHelpFunc(customHelp)
	rootCmd.CompletionOptions.HiddenDefaultCmd = true
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func customHelp(cmd *cobra.Command, args []string) {
	if cmd == rootCmd {
		printTree()
		return
	}

	// Leaf command: show full details
	if !cmd.HasSubCommands() {
		printLeafHelp(cmd)
		return
	}

	// Mid-level command: show subtree
	printSubtree(cmd)
}

func printTree() {
	fmt.Println("vikunja-cli - Vikunja task management CLI")
	fmt.Println("")
	fmt.Println("Global: --json (raw JSON output)")
	fmt.Println("")
	fmt.Println("Commands:")

	for _, cmd := range rootCmd.Commands() {
		if cmd.Hidden || cmd.Name() == "help" || cmd.Name() == "completion" {
			continue
		}
		if cmd.HasSubCommands() {
			fmt.Printf("  %s\n", cmd.Name())
			for _, sub := range cmd.Commands() {
				if sub.Hidden {
					continue
				}
				aliases := ""
				if len(sub.Aliases) > 0 {
					aliases = " (" + strings.Join(sub.Aliases, ", ") + ")"
				}
				fmt.Printf("    %-10s %s%s\n", sub.Name(), sub.Short, aliases)
			}
		} else {
			fmt.Printf("  %-12s %s\n", cmd.Name(), cmd.Short)
		}
	}

	fmt.Println("")
	fmt.Println("Run 'vikunja-cli <command> <subcommand> --help' for full details.")
}

func printSubtree(cmd *cobra.Command) {
	fmt.Printf("%s\n\n", cmd.Short)

	for _, sub := range cmd.Commands() {
		if sub.Hidden {
			continue
		}
		aliases := ""
		if len(sub.Aliases) > 0 {
			aliases = " (" + strings.Join(sub.Aliases, ", ") + ")"
		}
		fmt.Printf("  %-10s %s%s\n", sub.Name(), sub.Short, aliases)
	}

	fmt.Println("")
	fmt.Printf("Run 'vikunja-cli %s <subcommand> --help' for full details.\n", cmd.Name())
}

func printLeafHelp(cmd *cobra.Command) {
	fmt.Printf("%s %s\n", cmd.UseLine(), "")
	fmt.Println(cmd.Short)

	if cmd.HasLocalFlags() {
		fmt.Println("")
		fmt.Println("Flags:")
		cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
			shorthand := ""
			if f.Shorthand != "" {
				shorthand = "-" + f.Shorthand + ", "
			}
			def := ""
			if f.DefValue != "" && f.DefValue != "false" && f.DefValue != "0" {
				def = " (default: " + f.DefValue + ")"
			}
			fmt.Printf("  %s--%s %s%s\n", shorthand, f.Name, f.Usage, def)
		})
	}
}
