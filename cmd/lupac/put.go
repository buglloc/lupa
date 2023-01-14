package main

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

var putCmd = &cobra.Command{
	Use:           "put",
	SilenceUsage:  true,
	SilenceErrors: true,
	Short:         "retrieve data from the server",
	RunE: func(_ *cobra.Command, ids []string) error {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("unable to read data: %w", err)
		}

		lupac, cleanup, err := dial()
		if err != nil {
			return fmt.Errorf("dial failed: %w", err)
		}
		defer cleanup()

		keyID, err := lupac.Put(data)
		if err != nil {
			return fmt.Errorf("put failed: %w", err)
		}

		fmt.Printf("saved with keyID: %s\n", keyID)
		return nil
	},
}
