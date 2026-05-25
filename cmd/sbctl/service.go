/*
Copyright © 2026 Anthonius

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kilip/sbctl/internal/daemon"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// serviceCmd represents the service command
var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage sbctl background service",
	Long:  `Install, uninstall, start, stop, and view logs of the sbctl background service.`,
}

func getManager() (*daemon.Manager, error) {
	configFile := viper.ConfigFileUsed()
	configDir := filepath.Dir(configFile)
	return daemon.NewManager(configDir, configFile)
}

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the sbctl service",
	Run: func(cmd *cobra.Command, args []string) {
		m, err := getManager()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		if err := m.Install(); err != nil {
			fmt.Printf("Failed to install service: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Service installed successfully")
	},
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall the sbctl service",
	Run: func(cmd *cobra.Command, args []string) {
		m, err := getManager()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		if err := m.Uninstall(); err != nil {
			fmt.Printf("Failed to uninstall service: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Service uninstalled successfully")
	},
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the sbctl service",
	Run: func(cmd *cobra.Command, args []string) {
		m, err := getManager()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		if err := m.Start(); err != nil {
			fmt.Printf("Failed to start service: %v\n", err)
			os.Exit(1)
		}
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the sbctl service",
	Run: func(cmd *cobra.Command, args []string) {
		m, err := getManager()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		if err := m.Stop(); err != nil {
			fmt.Printf("Failed to stop service: %v\n", err)
			os.Exit(1)
		}
	},
}

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the sbctl service",
	Run: func(cmd *cobra.Command, args []string) {
		m, err := getManager()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		if err := m.Restart(); err != nil {
			fmt.Printf("Failed to restart service: %v\n", err)
			os.Exit(1)
		}
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check service status",
	Run: func(cmd *cobra.Command, args []string) {
		m, err := getManager()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		if err := m.Status(); err != nil {
			fmt.Printf("Failed to get status: %v\n", err)
			os.Exit(1)
		}
	},
}

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show service information",
	Run: func(cmd *cobra.Command, args []string) {
		m, err := getManager()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		if err := m.Info(); err != nil {
			fmt.Printf("Failed to get info: %v\n", err)
			os.Exit(1)
		}
	},
}

var restartTopCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the sbctl service",
	Run:   restartCmd.Run,
}

var statusTopCmd = &cobra.Command{
	Use:   "status",
	Short: "Check service status",
	Run:   statusCmd.Run,
}

var infoTopCmd = &cobra.Command{
	Use:   "info",
	Short: "Show service information",
	Run:   infoCmd.Run,
}

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View service logs",
	Run: func(cmd *cobra.Command, args []string) {
		m, err := getManager()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		if err := m.Logs(); err != nil {
			fmt.Printf("Error viewing logs: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(serviceCmd)
	serviceCmd.AddCommand(installCmd)
	serviceCmd.AddCommand(uninstallCmd)
	serviceCmd.AddCommand(startCmd)
	serviceCmd.AddCommand(stopCmd)
	serviceCmd.AddCommand(restartCmd)
	serviceCmd.AddCommand(statusCmd)
	serviceCmd.AddCommand(infoCmd)
	serviceCmd.AddCommand(logsCmd)

	rootCmd.AddCommand(restartTopCmd)
	rootCmd.AddCommand(statusTopCmd)
	rootCmd.AddCommand(infoTopCmd)
}
