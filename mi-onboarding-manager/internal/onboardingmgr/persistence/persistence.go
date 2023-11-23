/*
   Copyright (C) 2023 Intel Corporation
   SPDX-License-Identifier: Apache-2.0
*/

package persistence

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"reflect"
	"strings"
	"time"

	pb "github.com/intel-sandbox/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/api/grpc/onboardingmgr"
	"github.com/intel-sandbox/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/onboardingmgr/config"
	"github.com/intel-sandbox/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/pkg/logger"
	"k8s.io/client-go/rest"
)

type (
	ArtifactData struct {
		ID          string           `json:"id" db:"id"`
		Category    ArtifactCategory `json:"category" db:"category"`
		Name        string           `json:"name" db:"name"`
		Version     string           `json:"version" db:"version" `
		Description string           `json:"descrip" db:"descrip"`
		Detail      string           `json:"detail" db:"detail"`
		PackageUrl  string           `json:"pkg_url" db:"pkg_url"`
		Author      string           `json:"author" db:"author"`
		State       string           `json:"state" db:"state"`
		License     string           `json:"license" db:"license"`
	}

	NodeData struct {
		ID               string `json:"id" db:"id"`
		HwID             string `json:"hwid" db:"hwid"`
		PlatformType     string `json:"plat_type" db:"plat_type"`
		FwArtID          string `json:"fw_art_id" db:"fw_art_id"`
		OsArtID          string `json:"os_art_id" db:"os_art_id"`
		AppArtID         string `json:"app_art_id" db:"app_art_id"`
		PlatformArtID    string `json:"plat_art_id" db:"plat_art_id"`
		DeviceType       string `json:"dev_type" db:"dev_type"`
		DeviceInfoAgent  string `json:"dev_info_agent" db:"dev_info_agent"`
		DeviceStatus     string `json:"dev_status" db:"dev_status"`
		UpdateStatus     string `json:"update_status" db:"update_status"`
		UpdateAvailable  string `json:"update_avl" db:"update_avl"`
		OnboardingStatus string `json:"onboard_status" db:"onboard_status"`
	}

	ArtifactCategory string

	ProfileData struct {
		ID               string `json:"id" db:"id"`
		Name             string `json:"name" db:"name"`
		OsArtID          string `json:"os_art_id" db:"os_art_id"`
		FwArtID          string `json:"fw_art_id" db:"fw_art_id"`
		ImgArtID         string `json:"img_art_id" db:"img_art_id"`
		AppArtID         string `json:"app_art_id" db:"app_art_id"`
		HwData           string `json:"hw_data" db:"hw_data"`
		OnboardingParams string `json:"onboard_params" db:"onboard_params"`
		CustomerParams   string `json:"customer_params" db:"customer_params"`
		StartOnboarding  bool   `json:"start_onboard" db:"start_onboard"`
	}

	GroupData struct {
		ID      string `json:"id" db:"id"`
		Name    string `json:"name" db:"name"`
		NodeIDs string `json:"node_ids" db:"node_ids"`
	}
)

const (
	Platform    ArtifactCategory = "platform"
	Bios        ArtifactCategory = "bios"
	Os          ArtifactCategory = "os"
	Application ArtifactCategory = "app"
	Container   ArtifactCategory = "container"
)

func (ac ArtifactCategory) String() string {
	return string(ac)
}

type Repository interface {
	CreateNodes(ctx context.Context, data []NodeData) ([]NodeData, error)
	UpdateNodes(ctx context.Context, data []NodeData) error
	GetNodes(ctx context.Context, data NodeData) ([]*NodeData, error)
	DeleteNodes(ctx context.Context, ids []string) error

	CreateArtifacts(ctx context.Context, data []ArtifactData) ([]ArtifactData, error)
	UpdateArtifacts(ctx context.Context, data []ArtifactData) error
	GetArtifacts(ctx context.Context, data ArtifactData) ([]*ArtifactData, error)
	DeleteArtifacts(ctx context.Context, ids []string) error

	CreateProfiles(ctx context.Context, data []ProfileData) ([]ProfileData, error)
	UpdateProfiles(ctx context.Context, data []ProfileData) error
	GetProfiles(ctx context.Context, data ProfileData) ([]*ProfileData, error)
	DeleteProfiles(ctx context.Context, ids []string) error

	CreateGroups(ctx context.Context, data []GroupData) ([]GroupData, error)
	UpdateGroups(ctx context.Context, data []GroupData) error
	GetGroups(ctx context.Context, data GroupData) ([]*GroupData, error)
	DeleteGroups(ctx context.Context, ids []string) error

	Close() error
}

const (
	// CASSANDRA represents cassandra DB
	CASSANDRA = "cassandra"

	nodeRepoName     = "node"
	artifactRepoName = "artifact"
)

type myRepo struct {
	repos map[string]Repository
}

var repo = myRepo{repos: make(map[string]Repository, 2)}
var repoz = make(map[string]config.Database, 2)

func init() {
	repo.repos[nodeRepoName] = nil
	repo.repos[artifactRepoName] = nil
}

func runInKubernetes(conf *config.Config) {

	log.Println("Running inside Kubernetes cluster")

	repoz[nodeRepoName] = conf.Node.Database
	repoz[artifactRepoName] = conf.Artifact.Database
}

func runInDockerContainer(conf *config.Config) {

	log.Println("Running inside Docker container")

	repoz[nodeRepoName] = conf.Node.Database
	repoz[artifactRepoName] = conf.Artifact.Database
}

func runLocally(conf *config.Config) {
	log.Println("Running outside Kubernetes cluster")
	conf.Node.Database.Endpoints = "localhost:9042"
	repoz[nodeRepoName] = conf.Node.Database

	conf.Artifact.Database.Endpoints = "localhost:9042"
	repoz[artifactRepoName] = conf.Artifact.Database
}

func isRunningInDocker() bool {
	_, err := os.Stat("/.dockerenv")
	return !os.IsNotExist(err)
}

// InitDB initialize a database connection
func InitDB(conf *config.Config) {
	logger.GetLogger().Info("Try database connection")

	// Check if running inside Kubernetes cluster
	_, err := rest.InClusterConfig()
	isDocker := isRunningInDocker()
	if err == nil {
		// Running inside Kubernetes cluster
		runInKubernetes(conf)
	} else if isDocker {
		// Running inside Docker container
		runInDockerContainer(conf)
	} else {
		// Running outside Kubernetes cluster and Docker container
		runLocally(conf)
	}

	for n, d := range repoz {
		if d.Dialect == CASSANDRA {
			for {
				ep := strings.Split(d.Endpoints, ",")
				db, err := newCassandra(ep, d.Username, d.Password, conf.CreateTable, conf.Keyspace, conf.Replica)
				if err != nil {
					logger.GetLogger().Infof("Fail to connect %s database: %v", n, err)
					time.Sleep(5 * time.Second)
				}

				if err == nil {
					logger.GetLogger().Infof("Success %s database connection: %s", n, d.Endpoints)
					// set repo
					repo.repos[n] = db
					break
				}
			}
		}
	}
}

// GetNodeRepository returns the node repository
func GetNodeRepository() Repository {
	return repo.repos[nodeRepoName]
}

// GetArtifactRepository returns the artifact repository
func GetArtifactRepository() Repository {
	return repo.repos[artifactRepoName]
}

func isNil(i interface{}) bool {
	if i == nil {
		return true
	}
	switch reflect.TypeOf(i).Kind() {
	case reflect.Ptr, reflect.Map, reflect.Array, reflect.Chan, reflect.Slice:
		return reflect.ValueOf(i).IsNil()
	}
	return false
}

func MarshalToStr(data interface{}) (string, error) {
	if isNil(data) {
		return "", nil
	}
	b := new(strings.Builder)
	err := json.NewEncoder(b).Encode(data)
	if err != nil {
		return "", err
	}
	return b.String(), nil
}

func UnmarshalOnboardingParams(data string) (*pb.OnboardingParams, error) {
	if data == "" {
		return nil, nil
	}
	r := strings.NewReader(data)
	decoder := json.NewDecoder(r)
	var p pb.OnboardingParams
	err := decoder.Decode(&p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func UnmarshalCustomerParams(data string) (*pb.CustomerParams, error) {
	if data == "" {
		return nil, nil
	}
	r := strings.NewReader(data)
	decoder := json.NewDecoder(r)
	var p pb.CustomerParams
	err := decoder.Decode(&p)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func UnmarshalHwData(data string) ([]*pb.HwData, error) {
	if data == "" {
		return nil, nil
	}
	r := strings.NewReader(data)
	decoder := json.NewDecoder(r)
	var p []*pb.HwData
	err := decoder.Decode(&p)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func UnmarshalStrArray(data string) ([]string, error) {
	if data == "" {
		return nil, nil
	}
	r := strings.NewReader(data)
	decoder := json.NewDecoder(r)
	var p []string
	err := decoder.Decode(&p)
	if err != nil {
		return nil, err
	}
	return p, nil
}
