package cli

import (
	"github.com/ravi-hq/cli/internal/api"
	"github.com/ravi-hq/cli/internal/output"
	"github.com/spf13/cobra"
)

var domainsCmd = &cobra.Command{
	Use:   "domains",
	Short: "List available email domains",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewManagementClient()
		if err != nil {
			return err
		}

		domains, err := client.ListDomains()
		if err != nil {
			return err
		}

		return output.Current.Print(domains)
	},
}

func init() {
	rootCmd.AddCommand(domainsCmd)
}
