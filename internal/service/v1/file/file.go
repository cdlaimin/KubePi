package file

import (
	"encoding/json"
	"fmt"
	"github.com/KubeOperator/kubepi/internal/model/v1/file"
	"github.com/KubeOperator/kubepi/internal/service/v1/cluster"
	"github.com/KubeOperator/kubepi/internal/service/v1/common"
	"github.com/KubeOperator/kubepi/pkg/util"
	"github.com/KubeOperator/kubepi/pkg/util/kubernetes"
	"github.com/KubeOperator/kubepi/pkg/util/podbase"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type Service interface {
	ExecCommand(request file.Request) ([]byte, error)
	ListFiles(request file.Request) ([]util.File, error)
	DownloadFile(request file.Request) (string, error)
	UploadFile(request file.Request) error
}

type service struct {
	clusterService cluster.Service
}

func NewService() Service {
	return &service{
		clusterService: cluster.NewService(),
	}
}

func (f service) ExecCommand(request file.Request) ([]byte, error) {
	userPath, err := f.GetUserPath(request)
	if err != nil {
		return nil, err
	}
	kotoolCommand := []string{userPath + "/kotools"}
	request.Commands = append(kotoolCommand, request.Commands...)
	bs, err := f.fileBrowser(request)
	if err != nil {
		return nil, err
	}
	return bs, nil
}

func (f service) ListFiles(request file.Request) ([]util.File, error) {

	userPath, err := f.GetUserPath(request)
	if err != nil {
		return nil, err
	}
	kotoolCommand := userPath + "/kotools"
	commands := []string{kotoolCommand, "ls", request.Path}
	request.Commands = commands
	var res []util.File
	bs, err := f.fileBrowser(request)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(bs, &res); err != nil {
		return nil, err
	}
	return res, nil
}

func (f service) DownloadFile(request file.Request) (string, error) {

	var fileP string
	pb, err := f.GetPodBase(request)
	if err != nil {
		return fileP, err
	}
	//clu, err := f.clusterService.Get(request.Cluster, common.DBOptions{})
	//if err != nil {
	//	return fileP, err
	//}
	//config := &kubernetes.Config{
	//	Host:  clu.Spec.Connect.Forward.ApiServer,
	//	Token: clu.Spec.Authentication.BearerToken,
	//}
	//k8sConfig := kubernetes.NewClusterConfig(config)
	//k8sClient, err := kubernetes.NewKubernetesClient(config)
	//if err != nil {
	//	return fileP, err
	//}
	//pb := podbase.PodBase{
	//	Namespace:  request.Namespace,
	//	PodName:    request.PodName,
	//	Container:  request.ContainerName,
	//	K8sClient:  k8sClient,
	//	RestClient: k8sConfig,
	//}
	exec := pb.NewPodExec()
	fileNameWithSuffix := path.Base(request.Path)
	fileType := path.Ext(fileNameWithSuffix)
	fileName := strings.TrimSuffix(fileNameWithSuffix, fileType)
	fileP = filepath.Join(os.TempDir(), fmt.Sprintf("%d", time.Now().UnixNano()))
	err = os.MkdirAll(fileP, os.ModePerm)
	if err != nil {
		return "", err
	}
	fileP = filepath.Join(fileP, fileName+".tar")
	err = exec.CopyFromPod(request.Path, fileP)
	if err != nil {
		return "", err
	}

	return fileP, nil
}

func (f service) UploadFile(request file.Request) error {
	//clu, err := f.clusterService.Get(request.Cluster, common.DBOptions{})
	//if err != nil {
	//	return err
	//}
	//config := &kubernetes.Config{
	//	Host:  clu.Spec.Connect.Forward.ApiServer,
	//	Token: clu.Spec.Authentication.BearerToken,
	//}
	//k8sConfig := kubernetes.NewClusterConfig(config)
	//k8sClient, err := kubernetes.NewKubernetesClient(config)
	//if err != nil {
	//	return err
	//}
	//pb := podbase.PodBase{
	//	Namespace:  request.Namespace,
	//	PodName:    request.PodName,
	//	Container:  request.ContainerName,
	//	K8sClient:  k8sClient,
	//	RestClient: k8sConfig,
	//}
	//exec := pb.NewPodExec()

	pb, err := f.GetPodBase(request)
	if err != nil {
		return err
	}
	exec := pb.NewPodExec()
	err = exec.CopyToPod(request.FilePath, request.Path)
	if err != nil {
		return nil
	}
	return nil
}

func (f service) GetPodBase(request file.Request) (podbase.PodBase, error) {
	var pb podbase.PodBase
	clu, err := f.clusterService.Get(request.Cluster, common.DBOptions{})
	if err != nil {
		return pb, err
	}
	config := &kubernetes.Config{
		Host:  clu.Spec.Connect.Forward.ApiServer,
		Token: clu.Spec.Authentication.BearerToken,
	}
	k8sConfig := kubernetes.NewClusterConfig(config)
	k8sClient, err := kubernetes.NewKubernetesClient(config)
	if err != nil {
		return pb, err
	}
	pb = podbase.PodBase{
		Namespace:  request.Namespace,
		PodName:    request.PodName,
		Container:  request.ContainerName,
		K8sClient:  k8sClient,
		RestClient: k8sConfig,
	}
	return pb, nil
}

func (f service) GetUserPath(request file.Request) (string, error) {
	pb, err := f.GetPodBase(request)
	if err != nil {
		return "", err
	}
	return pb.GetUserPath()
}

func (f service) fileBrowser(request file.Request) (res []byte, err error) {

	pb, err := f.GetPodBase(request)
	if err != nil {
		return nil, err
	}
	res, err = pb.Exec(request.Stdin, request.Commands...)
	if err != nil {
		if strings.Contains(err.Error(), "no such file or directory") ||
			err.Error() == "command terminated with exit code 126" {
			err = pb.InstallKOTools()
			if err != nil {
				return nil, err
			}
			res, err = pb.Exec(request.Stdin, request.Commands...)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return res, err
}
