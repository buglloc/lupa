package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootArgs struct {
	PrivateKey        string
	RemoteAddr        string
	RemoteFingerprint string
}

var rootCmd = &cobra.Command{
	Use:           "lupac",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	flags := rootCmd.PersistentFlags()
	flags.StringVar(&rootArgs.PrivateKey, "key", "id_rsa", "key for authentication")
	flags.StringVar(&rootArgs.RemoteAddr, "addr", "localhost:2022", "remote addr to connect to")
	flags.StringVar(&rootArgs.RemoteFingerprint, "fingerprint", "", "remote host fingerprint")

	rootCmd.AddCommand(
		getCmd,
		putCmd,
	)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
