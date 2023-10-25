package cmd

import (
	"fmt"
	"os"

	"github.com/drewstinnett/taskpoet/internal/ui"
	"github.com/spf13/cobra"
)

// uiCmd represents the ui command
func newUICmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "ui",
		Short:  "Run the UI",
		Hidden: true,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("ui called")
			p := ui.NewUI(poetC)
			if _, err := p.Run(); err != nil {
				fmt.Printf("Alas, there's been an error: %v", err)
				os.Exit(1)
			}
		},
	}
	return cmd
}
