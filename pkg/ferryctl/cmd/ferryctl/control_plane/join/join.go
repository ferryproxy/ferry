package join

import (
	"fmt"
	"strings"

	import_cmd "github.com/ferry-proxy/ferry/pkg/ferryctl/cmd/ferryctl/local/manual/import"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/control_plane"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/kubectl"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/log"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/utils"
	"github.com/ferry-proxy/ferry/pkg/ferryctl/vars"
	"github.com/spf13/cobra"
)

func NewCommand(logger log.Logger) *cobra.Command {
	var (
		controlPlaneTunnelAddress = vars.AutoPlaceholders
		dataPlaneTunnelAddress    = vars.AutoPlaceholders
		dataPlaneApiserverAddress = vars.AutoPlaceholders
		controlPlaneReachable     = true
		dataPlaneReachable        = true
	)

	cmd := &cobra.Command{
		Args: cobra.ExactArgs(1),
		Use:  "join <data-plane-hub-name>",
		Aliases: []string{
			"j",
		},
		Short: "Control plane join commands",
		Long:  `Control plane join commands is used to join other data plane`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			dataPlaneName := args[0]

			if !controlPlaneReachable {
				controlPlaneTunnelAddress = ""
			}

			if !dataPlaneReachable {
				dataPlaneTunnelAddress = ""
			}

			kctl := kubectl.NewKubectl()
			if controlPlaneTunnelAddress == vars.ControlPlaneName {
				controlPlaneTunnelAddress, err = kctl.GetTunnelAddress(cmd.Context())
				if err != nil {
					return err
				}
			}

			fargs := []string{}

			if !controlPlaneReachable {
				if !dataPlaneReachable {
					return fmt.Errorf("TODO: data plane and control plane is not reachable")
				}

				fargs = []string{
					"--reachable=false",
					"--peer-tunnel-address=" + dataPlaneTunnelAddress,
					"--export-host-port=kubernetes.default.svc:443",
					"--import-service-name=" + dataPlaneName + "-apiserver",
				}
			} else {
				if !dataPlaneReachable {
					fargs = []string{
						"--reachable=true",
						"--tunnel-address=" + controlPlaneTunnelAddress,
						"--export-host-port=kubernetes.default.svc:443",
						"--import-service-name=" + dataPlaneName + "-apiserver",
					}
				}
			}

			if len(fargs) != 0 {
				fmt.Printf("# > ferryctl local manual import %s\n", strings.Join(fargs, " "))
				sub := import_cmd.NewCommand(logger)
				sub.SetArgs(fargs)
				err := sub.ExecuteContext(cmd.Context())
				if err != nil {
					return err
				}
			}

			next, err := control_plane.ShowJoin(cmd.Context(), control_plane.ShowJoinConfig{
				ControlPlaneName:          vars.ControlPlaneName,
				DataPlaneName:             dataPlaneName,
				DataPlaneApiserverAddress: dataPlaneApiserverAddress,
				DataPlaneTunnelAddress:    dataPlaneTunnelAddress,
				DataPlaneReachable:        dataPlaneReachable,
				ControlPlaneTunnelAddress: controlPlaneTunnelAddress,
				ControlPlaneReachable:     controlPlaneReachable,
			})
			if err != nil {
				return err
			}

			utils.Prompt(
				fmt.Sprintf("join the %s data cluster", dataPlaneName),
				"ferryctl data-plane init",
				next,
			)

			return nil
		},
	}
	flags := cmd.Flags()
	flags.StringVar(&controlPlaneTunnelAddress, "control-plane-tunnel-address", controlPlaneTunnelAddress, "Tunnel address of the control plane connected to another cluster")
	flags.StringVar(&dataPlaneTunnelAddress, "data-plane-tunnel-address", dataPlaneTunnelAddress, "Tunnel address of the data plane connected to another cluster")
	flags.StringVar(&dataPlaneApiserverAddress, "data-plane-apiserver-address", dataPlaneApiserverAddress, "Apiserver address of the data plane for control plane")
	flags.BoolVar(&controlPlaneReachable, "control-plane-reachable", controlPlaneReachable, "Control plane is reachable")
	flags.BoolVar(&dataPlaneReachable, "data-plane-reachable", dataPlaneReachable, "Data plane is reachable")
	return cmd
}
