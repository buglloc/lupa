package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:           "get",
	SilenceUsage:  true,
	SilenceErrors: true,
	Short:         "retrieve data from the server",
	RunE: func(_ *cobra.Command, ids []string) error {
		lupac, cleanup, err := dial()
		if err != nil {
			return fmt.Errorf("dial failed: %w", err)
		}
		defer cleanup()

		for _, keyID := range ids {
			data, err := lupac.Get(keyID)
			if err != nil {
				fmt.Printf("unable to get key %q: %v\n", keyID, err)
				continue
			}

			fmt.Printf("%s: %q\n", keyID, string(data))
		}

		return nil
	},
}
