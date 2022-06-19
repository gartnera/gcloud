package container

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	container "cloud.google.com/go/container/apiv1"
	"github.com/gartnera/gcloud/auth"
	"github.com/spf13/cobra"
	"google.golang.org/api/option"
	containerpb "google.golang.org/genproto/googleapis/container/v1"
	clientcmd "k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

var clustersCmd = &cobra.Command{
	Use: "clusters",
}

var clustersGetCredentialsCmd = &cobra.Command{
	Use:  "get-credentials",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		cFlags, err := getCommonFlags(cmd.Flags())
		if err != nil {
			return err
		}
		// for whatever reason, option.WithTokenSource(auth.TokenSource()) does not work here
		ts, err := auth.TokenSource()
		if err != nil {
			return fmt.Errorf("unable to get token source: %w", err)
		}
		client, err := container.NewClusterManagerClient(ctx, option.WithTokenSource(ts))
		if err != nil {
			return fmt.Errorf("unable to get container client: %w", err)
		}
		clusterName := args[0]
		var gName string
		var kName string
		if cFlags.region != "" {
			gName = fmt.Sprintf("projects/%s/locations/%s/clusters/%s", cFlags.project, cFlags.region, clusterName)
			kName = fmt.Sprintf("gke_%s_%s_%s", cFlags.project, cFlags.region, clusterName)
		} else {
			gName = fmt.Sprintf("projects/%s/zones/%s/clusters/%s", cFlags.project, cFlags.zone, clusterName)
			kName = fmt.Sprintf("gke_%s_%s_%s", cFlags.project, cFlags.zone, clusterName)
		}
		req := &containerpb.GetClusterRequest{
			Name: gName,
		}
		cluster, err := client.GetCluster(ctx, req)
		if err != nil {
			return fmt.Errorf("unable to get cluster %s: %w", clusterName, err)
		}

		kubeConfigPath := os.Getenv("KUBECONFIG")
		if kubeConfigPath == "" {
			kubeConfigPath = os.ExpandEnv("${HOME}/.kube/config")
		}
		kubeConfigDir := filepath.Dir(kubeConfigPath)
		_ = os.MkdirAll(kubeConfigDir, os.ModeDir)

		var kubeConfig *clientcmdapi.Config
		if _, err = os.Stat(kubeConfigPath); os.IsNotExist(err) {
			kubeConfig, err = clientcmd.Load([]byte{})
			if err != nil {
				return fmt.Errorf("unable to initialize kubeconfig: %w", err)
			}
		} else {
			kubeConfig, err = clientcmd.LoadFromFile(kubeConfigPath)
			if err != nil {
				return fmt.Errorf("unable to load kubeconfig: %w", err)
			}
		}
		caCertificateDecoded, err := base64.StdEncoding.DecodeString(cluster.MasterAuth.ClusterCaCertificate)
		if err != nil {
			return fmt.Errorf("unable to decode ca certificate: %w", err)
		}
		kubeConfig.Clusters[kName] = &clientcmdapi.Cluster{
			Server:                   fmt.Sprintf("https://%s", cluster.Endpoint),
			CertificateAuthorityData: caCertificateDecoded,
		}
		kubeConfig.AuthInfos[kName] = &clientcmdapi.AuthInfo{
			Exec: &clientcmdapi.ExecConfig{
				Command:         "gke-gcloud-auth-plugin",
				APIVersion:      "client.authentication.k8s.io/v1beta1",
				InteractiveMode: clientcmdapi.NeverExecInteractiveMode,
			},
		}
		kubeConfig.Contexts[kName] = &clientcmdapi.Context{
			Cluster:  kName,
			AuthInfo: kName,
		}
		kubeConfig.CurrentContext = kName

		err = clientcmd.WriteToFile(*kubeConfig, kubeConfigPath)
		if err != nil {
			return fmt.Errorf("unable to write kubeconfig: %w", err)
		}

		return nil
	},
}

func registerConfigHelperCmd(parent *cobra.Command) {
	clustersCmd.AddCommand(clustersGetCredentialsCmd)
	parent.AddCommand(clustersCmd)
}
