/*
 * SPDX-FileCopyrightText: (C) 2023 Intel Corporation
 * SPDX-License-Identifier: LicenseRef-Intel
 */
package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Helper function to print usage and exit without error. This is helpful in
// conjunction with `Args: onlyNestedSubCommands` when a command is only used
// for collectioning subcommands and persistent flags.
func printUsage(c *cobra.Command, args []string) error {
	return c.Usage()
}

// Helper function to provide the same unknown subcommand error message with
// suggestions to nested commands as Cobra does for the root command.
func onlyNestedSubCommands(c *cobra.Command, args []string) error {
	if len(args) == 0 {
		return nil
	}

	var suggestionsString string
	if !c.DisableSuggestions {
		if c.SuggestionsMinimumDistance <= 0 {
			c.SuggestionsMinimumDistance = 2
		}

		if suggestions := c.SuggestionsFor(args[0]); len(suggestions) > 0 {
			suggestionsString += "\n\nDid you mean this?\n"
			for _, s := range suggestions {
				suggestionsString += fmt.Sprintf("\t%v\n", s)
			}
		}
	}

	c.SilenceUsage = true
	return fmt.Errorf("unknown command %q for %q%s\nRun '%s --help' for usage",
		args[0], c.CommandPath(), suggestionsString, c.CommandPath())
}

// Panic on error, because the error is known statically never to occur. If it
// does, then a programming error occurred, not a user or runtime error, such
// as a race condition.
//
// This helper exists for when the Go type system is not sufficiently strong or
// not sufficiently used.
//
// In the case of Cobra, it is useful when checking the return of
// MarkFlagRequired-type function. The function should only error if there is a
// typo in the code or the flag is created out of order.
func must(err error) {
	if err != nil {
		panic("PROGRAMMING ERROR: " + err.Error())
	}
}
