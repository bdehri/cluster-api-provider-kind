package kind

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"text/template"

	"github.com/bdehri/cluster-api-provider-kind/constants"
)

const (
	alreadyDeletingOutputFormat = "Error response from daemon: removal of container %s-control-plane is already in progress"
	alreadyCreatedOutputFormat  = "ERROR: failed to create cluster: node(s) already exist for a cluster with the name \"%s\""

	kindClusterTemplate = `{{$version := .Version}}
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: {{.Name}}
networking:
  podSubnet: {{.Cidr}}
nodes:
{{range .ControlPlaneCount}}
- role: control-plane
  image: {{$version}}
{{end}}
{{range .WorkerCount}}
- role: worker
  image: {{$version}}
{{end}}`
)

type Kind struct {
	Version           string
	Name              string
	ControlPlaneCount []int
	WorkerCount       []int
	Cidr              string
}

type ClientInt interface {
	Create(name, version, cidr string, controlPlaneCount, workerCount int) error
	Delete(name string) error
	GetKubeconfig(name string) ([]byte, error)
}

func generateKindClusterYaml(name, version, cidr string, controlPlaneCount, workerCount int) (string, error) {
	t := template.New("kind cluster template")
	t, err := t.Parse(kindClusterTemplate)
	if err != nil {
		return "", err
	}

	var buff bytes.Buffer

	err = t.Execute(&buff, Kind{
		Version:           constants.KindK8SImages[version],
		Name:              name,
		ControlPlaneCount: make([]int, controlPlaneCount),
		WorkerCount:       make([]int, workerCount),
		Cidr:              cidr,
	})
	if err != nil {
		return "", err
	}

	return buff.String(), nil
}

func (kindClient *Client) Create(name, version, cidr string, controlPlaneCount, workerCount int) error {
	kindClusterYaml, err := generateKindClusterYaml(name, version, cidr, controlPlaneCount, workerCount)
	if err != nil {
		return err
	}

	cmd := exec.Command("kind", "create", "cluster", "--kubeconfig", "/dev/null", "--config", "-")
	cmd.Stdin = strings.NewReader(kindClusterYaml)
	output, err := cmd.CombinedOutput()
	if strings.TrimSuffix(string(output), "\n") != fmt.Sprintf(alreadyCreatedOutputFormat, name) && err != nil {
		return fmt.Errorf("%s: %s: %w", output, kindClusterYaml, err)
	}
	return nil
}

func (kindClient *Client) Delete(name string) error {
	alreadyDeleting := fmt.Sprintf(alreadyDeletingOutputFormat, name)

	cmd := exec.Command("kind", "delete", "cluster", "--name", name)
	output, err := cmd.Output()
	if err != nil && string(output) == alreadyDeleting {
		return err
	}
	return nil
}

func (kindClient *Client) GetKubeconfig(name string) ([]byte, error) {
	cmd := exec.Command("kind", "get", "kubeconfig", "--name", name)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return output, nil
}

type Client struct{}
