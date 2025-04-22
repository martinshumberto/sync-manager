package commands

import (
	"fmt"
	"os"
	"time"

	"github.com/martinshumberto/sync-manager/common/config"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

// CreateDeviceCommands returns the device management commands
func CreateDeviceCommands(cfg *config.Config) []*cobra.Command {
	// Devices root command
	devicesCmd := &cobra.Command{
		Use:   "devices",
		Short: "Manage connected devices",
		Long:  `View and manage devices connected to your account.`,
	}

	// Devices list command
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List connected devices",
		Long:  `Display a list of all devices connected to your account.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Connected Devices:")
			fmt.Println("-----------------")

			// In a real implementation, we would fetch this from the server
			// For now, we'll just display simulated data
			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"Device ID", "Name", "Last Seen", "Status"})

			// Current device
			table.Append([]string{
				cfg.DeviceID,
				cfg.DeviceName + " (this device)",
				"Now",
				"Online",
			})

			// Simulated other devices
			table.Append([]string{
				"d8f3a1c2-5b6e-7d8f-9a0b-1c2d3e4f5a6b",
				"John's Laptop",
				"2 hours ago",
				"Offline",
			})

			table.Append([]string{
				"a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d",
				"Office Desktop",
				"12 minutes ago",
				"Online",
			})

			table.Render()
			return nil
		},
	}

	// Devices unlink command
	unlinkCmd := &cobra.Command{
		Use:   "unlink <device-id>",
		Short: "Unlink a device from your account",
		Long:  `Remove the connection between a device and your account.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			deviceID := args[0]

			// Check if trying to unlink current device
			if deviceID == cfg.DeviceID {
				return fmt.Errorf("cannot unlink the current device. Use 'reset' command instead if you want to reconfigure this device")
			}

			// Ask for confirmation
			fmt.Printf("Are you sure you want to unlink device %s? (y/n): ", deviceID)
			var response string
			fmt.Scanln(&response)

			if response != "y" && response != "Y" {
				fmt.Println("Operation cancelled.")
				return nil
			}

			fmt.Printf("Unlinking device %s...\n", deviceID)

			// In a real implementation, we would:
			// 1. Connect to the server
			// 2. Remove the device authorization
			// 3. Handle any cleanup

			// Simulate processing
			time.Sleep(1 * time.Second)

			fmt.Println("Device successfully unlinked.")
			fmt.Println("This device will no longer be able to access your account or synchronize files.")

			return nil
		},
	}

	// Devices rename command (for current device)
	renameCmd := &cobra.Command{
		Use:   "rename <new-name>",
		Short: "Rename this device",
		Long:  `Change the name of the current device.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			newName := args[0]
			oldName := cfg.DeviceName

			// Validate name
			if newName == "" {
				return fmt.Errorf("device name cannot be empty")
			}

			// Update the device name
			cfg.DeviceName = newName

			// In a real implementation, we would also:
			// 1. Connect to the server
			// 2. Update the device name in the remote database
			// 3. Sync the changes to other devices

			fmt.Printf("Device renamed from '%s' to '%s'.\n", oldName, newName)

			// Return the updated config to be saved
			return nil
		},
	}

	// Devices info command
	infoCmd := &cobra.Command{
		Use:   "info [device-id]",
		Short: "Show detailed information about a device",
		Long:  `Display detailed information about a specific device or the current device if no ID is provided.`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var deviceID string
			var isCurrentDevice bool

			// If no device ID is provided, use the current device
			if len(args) == 0 {
				deviceID = cfg.DeviceID
				isCurrentDevice = true
			} else {
				deviceID = args[0]
				isCurrentDevice = (deviceID == cfg.DeviceID)
			}

			// In a real implementation, we would fetch device details from the server
			// For now, we'll display information for the current device and simulated data for others

			fmt.Println("Device Information:")
			fmt.Println("------------------")

			if isCurrentDevice {
				fmt.Printf("Device ID:      %s\n", cfg.DeviceID)
				fmt.Printf("Name:           %s (this device)\n", cfg.DeviceName)
				fmt.Printf("Status:         Online\n")
				fmt.Printf("Last Seen:      Now\n")
				fmt.Printf("Storage:        %s\n", cfg.StorageProvider)
				fmt.Printf("Sync Interval:  %s\n", cfg.SyncInterval)
				fmt.Printf("Sync Folders:   %d\n", len(cfg.SyncFolders))

				// Display synced folders
				if len(cfg.SyncFolders) > 0 {
					fmt.Println("\nSynced Folders:")
					table := tablewriter.NewWriter(os.Stdout)
					table.SetHeader([]string{"ID", "Path", "Status"})

					for _, folder := range cfg.SyncFolders {
						status := "Enabled"
						if !folder.Enabled {
							status = "Disabled"
						}

						table.Append([]string{
							folder.ID,
							folder.Path,
							status,
						})
					}

					table.Render()
				}
			} else if deviceID == "d8f3a1c2-5b6e-7d8f-9a0b-1c2d3e4f5a6b" {
				// Simulated device 1
				fmt.Printf("Device ID:      %s\n", deviceID)
				fmt.Printf("Name:           %s\n", "John's Laptop")
				fmt.Printf("Status:         Offline\n")
				fmt.Printf("Last Seen:      2 hours ago\n")
				fmt.Printf("Storage:        minio\n")
				fmt.Printf("Sync Folders:   2\n")
			} else if deviceID == "a1b2c3d4-e5f6-7a8b-9c0d-1e2f3a4b5c6d" {
				// Simulated device 2
				fmt.Printf("Device ID:      %s\n", deviceID)
				fmt.Printf("Name:           %s\n", "Office Desktop")
				fmt.Printf("Status:         Online\n")
				fmt.Printf("Last Seen:      12 minutes ago\n")
				fmt.Printf("Storage:        minio\n")
				fmt.Printf("Sync Folders:   3\n")
			} else {
				return fmt.Errorf("device with ID %s not found", deviceID)
			}

			return nil
		},
	}

	// Add subcommands to devices command
	devicesCmd.AddCommand(listCmd)
	devicesCmd.AddCommand(unlinkCmd)
	devicesCmd.AddCommand(renameCmd)
	devicesCmd.AddCommand(infoCmd)

	return []*cobra.Command{devicesCmd}
}
