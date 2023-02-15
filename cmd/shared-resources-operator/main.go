package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"k8s.io/client-go/tools/clientcmd"
	k8sflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/logs"

	"github.com/openshift/library-go/pkg/controller/controllercmd"

	"github.com/openshift/csi-driver-shared-resource-operator/pkg/operator"
	"github.com/openshift/csi-driver-shared-resource-operator/pkg/version"
)

var (
	kubeconfig *string
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	pflag.CommandLine.SetNormalizeFunc(k8sflag.WordSepNormalizeFunc)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	logs.InitLogs()
	defer logs.FlushLogs()

	command := NewOperatorCommand()
	if err := command.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}
}

func NewOperatorCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "shared-resources-operator",
		Short: "OpenShift Projected Shared Resources Operator",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
			os.Exit(1)
		},
	}

	ctrlCmd := controllercmd.NewControllerCommandConfig(
		"shared-resources-operator",
		version.Get(),
		runOperatorWithKubeconfig,
	).NewCommandWithContext(context.TODO()) //TODO cmd.Context()) came back with panic: cannot create context from nil parent
	ctrlCmd.Use = "start"
	ctrlCmd.Short = "Start the Projected Shared Resources Operator"
	kubeconfig = ctrlCmd.Flags().String("kubeconfig", "", "Path to kubeconfig file.  If not provided, configured service account will be used.")

	cmd.AddCommand(ctrlCmd)

	return cmd
}

func runOperatorWithKubeconfig(ctx context.Context, controllerConfig *controllercmd.ControllerContext) error {
	if kubeconfig != nil && *kubeconfig != "" {
		rules := clientcmd.NewDefaultClientConfigLoadingRules()
		rules.ExplicitPath = *kubeconfig
		c, err := rules.Load()
		if err != nil {
			return err
		}
		clientConfig := clientcmd.NewDefaultClientConfig(*c, nil)
		controllerConfig.KubeConfig, err = clientConfig.ClientConfig()
		if err != nil {
			return err
		}
	}
	return operator.RunOperator(ctx, controllerConfig)
}
