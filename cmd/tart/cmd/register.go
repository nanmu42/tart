package cmd

import (
	"fmt"
	"os"

	"github.com/nanmu42/tart/config"
	"github.com/nanmu42/tart/executor"
	"github.com/nanmu42/tart/network"

	"github.com/pelletier/go-toml/v2"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(registerCmd)
	registerCmd.Flags().StringVar(&endpoint, "endpoint", "", "Gitlab URL, only scheme + host, e.g. https://gitlab.example.com")
	registerCmd.Flags().StringVar(&registrationToken, "token", "", "Gitlab Runner registration token")
	registerCmd.Flags().StringVar(&description, "description", "", "Description to this runner, submitted to Gitlab")
	_ = registerCmd.MarkFlagRequired("endpoint")
	_ = registerCmd.MarkFlagRequired("token")
}

var (
	endpoint          string
	registrationToken string
	description       string
)

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Register self to Gitlab and print TOML config into stdout",
	Example: `# redirect the output into config file
tart register --endpoint https://gitlab.example.com --token your_token_here > tart.toml`,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()

		client, err := network.NewClient(network.ClientOpt{
			Endpoint: endpoint,
			Features: executor.SupportFeatures(),
		})
		if err != nil {
			err = fmt.Errorf("initializing network client: %w", err)
			return
		}

		accessToken, err := client.Register(ctx, network.RegisterParam{
			Token:       registrationToken,
			Description: description,
		})
		if err != nil {
			err = fmt.Errorf("registering tart via Gitlab API: %w", err)
			return
		}

		cfg := config.Config{
			GitlabEndpoint: endpoint,
			AccessToken:    accessToken,
			Executor: executor.Config{
				KernelPath: "vmlinux-5.10.bin",
				RootFSPath: "jammy.rootfs.ext4",
				IP:         "172.18.0.2",
				GatewayIP:  "172.18.0.1",
				Netmask:    "255.255.255.0",
				TapDevice:  "tap0",
				TapMac:     "AA:FC:42:42:66:88",
			},
		}

		encoder := toml.NewEncoder(os.Stdout)
		err = encoder.Encode(cfg)
		if err != nil {
			err = fmt.Errorf("encoding config toml: %w", err)
			return
		}

		return
	},
}
