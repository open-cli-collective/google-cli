package drive

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	driverw "github.com/open-cli-collective/google-cli/internal/rw/drive"
)

func newUploadCommand() *cobra.Command {
	var parent, name string
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "upload <local-path>",
		Short: "Upload a file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			info, err := os.Stat(args[0])
			if err != nil {
				return fmt.Errorf("reading local file: %w", err)
			}
			typ, err := driverw.FileMIMEType(args[0])
			if err != nil {
				return err
			}
			if dryRun {
				target := parent
				if target == "" {
					target = "root"
				}
				fmt.Printf("[dry-run] Would upload file:\nPath: %s\nSize: %d bytes\nMIME type: %s\nParent: %s\n", args[0], info.Size(), typ, target)
				return nil
			}
			client, err := newWriteClient(cmd.Context())
			if err != nil {
				return fmt.Errorf("creating Drive client: %w", err)
			}
			file, err := client.UploadFile(cmd.Context(), args[0], parent, name)
			if err != nil {
				return fmt.Errorf("uploading file: %w", err)
			}
			printFile(file)
			return nil
		},
	}
	cmd.Flags().StringVar(&parent, "parent", "", "Parent folder ID")
	cmd.Flags().StringVar(&name, "name", "", "Drive filename (defaults to the local filename)")
	cmd.Flags().BoolVarP(&dryRun, "dry-run", "n", false, "Preview without making changes")
	return cmd
}
