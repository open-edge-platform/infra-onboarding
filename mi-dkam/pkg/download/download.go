// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package download

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/config"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/logging"
)

var zlog = logging.GetLogger("MIDKAMAuth")

func DownloadMicroOS(scriptPath string) (bool, error) {
	zlog.Info().Msgf("Inside Download and sign artifact... %s", scriptPath)
	url := "http://rs-proxy-files.rs-proxy.svc.cluster.local:8081/publish/fm_en_artifacts/hook-os/0.8.1-dev-e87f024/alpine_image/hook_x86_64.tar.gz"
	client := &http.Client{
		Transport: &http.Transport{
			ForceAttemptHTTP2: false,
			MaxIdleConns:      10,
			IdleConnTimeout:   30,
		},
	}
	// Create an HTTP GET request with the specified URL
	req, httperr := http.NewRequest("GET", url, nil)
	if httperr != nil {
		zlog.MiSec().Fatal().Err(httperr).Msgf("Error creating request: %v\n", httperr)
		return false, httperr

	}

	// Set the HTTP version to 1.1
	req.Proto = "HTTP/1.1"

	// Perform the HTTP GET request
	resp, clienterr := client.Do(req)
	if clienterr != nil {
		zlog.MiSec().Fatal().Err(clienterr).Msgf("Error performing request: %v\n", clienterr)
		return false, clienterr

	}
	defer resp.Body.Close()

	filePath := config.PVC + "hook_x86_64.tar.gz"
	//Read the response body
	//Create or open the local file for writing
	file, fileerr := os.Create(filePath)
	if fileerr != nil {
		zlog.MiSec().Fatal().Err(fileerr).Msgf("Error while creating release manifest file.")
		return false, fileerr
	}
	defer file.Close()

	// Copy the response body to the local file
	_, copyErr := io.Copy(file, resp.Body)
	if copyErr != nil {
		zlog.MiSec().Fatal().Err(copyErr).Msgf("Error while coping content ")
	}
	zlog.MiSec().Info().Msg("File downloaded")
	return true, nil

}

func DownloadArtifacts(fileServer string, harborServer string, scriptPath string, tag string) error {
	errp := os.Chdir(scriptPath)
	if errp != nil {
		zlog.MiSec().Fatal().Err(errp).Msgf("Error changing working directory: %v\n", errp)
		return errp
	}
	outDir := filepath.Join(scriptPath, "tmp")
	// 0. cleanup
	os.RemoveAll(outDir)

	mkErr := os.MkdirAll(outDir, 0755) // 0755 sets read, write, and execute permissions for owner, and read and execute permissions for others
	if mkErr != nil {
		zlog.MiSec().Fatal().Err(mkErr).Msgf("Error creating directory: %v", mkErr)
		return mkErr
	}
	zlog.MiSec().Info().Msg("tmp folder created successfully")

	// 1. Create a file store
	// fs, err := file.New(outDir)
	// if err != nil {
	// 	panic(err)
	// }
	// defer fs.Close()

	// // 2. Connect to a remote repository
	// ctx := context.Background()
	// repo, err := remote.NewRepository(harborServer + "/" + config.Artifact)
	// if err != nil {
	// 	panic(err)
	// }

	// // 3. Authenticate (not required in AMR)
	// /*
	// 	repo.Client = &auth.Client{
	// 		Client: retry.DefaultClient,
	// 		Cache:  auth.DefaultCache,
	// 		Credential: auth.StaticCredential(reg, auth.Credential{
	// 			Username: "username",
	// 			Password: "password",
	// 		}),
	// 	}*/

	// // 4. Copy from the remote repository to the file store
	// manifestDescriptor, err := oras.Copy(ctx, repo, tag, fs, tag, oras.DefaultCopyOptions)
	// if err != nil {
	// 	panic(err)
	// }
	// zlog.MiSec().Info().Msgf("Manifest descriptor: %s", manifestDescriptor)

	// // 5.list files
	// zlog.MiSec().Info().Msg("Download files:")
	// entries, err := os.ReadDir(outDir)
	// if err != nil {
	// 	zlog.MiSec().Fatal().Err(err).Msgf("Error reading the folder %v", err)
	// }

	// for _, e := range entries {
	// 	zlog.MiSec().Info().Msgf("filename %s", e.Name())
	// 	manifestFile := filepath.Join(outDir, e.Name())
	// 	releaseFile := filepath.Join(outDir, config.ReleaseVersion+".yaml")
	// 	if strings.Contains(e.Name(), "24.03") {
	// 		e := os.Rename(manifestFile, releaseFile)
	// 		if e != nil {
	// 			zlog.MiSec().Fatal().Err(err).Msgf("Failed to rename file %v", e)
	// 		}
	// 	}
	// }

	url := "http://rs-proxy-files.rs-proxy.svc.cluster.local:8081/publish/release-manifest/24.03.0-dev.yaml"
	client := &http.Client{
		Transport: &http.Transport{
			ForceAttemptHTTP2: false,
			MaxIdleConns:      10,
			IdleConnTimeout:   30,
		},
	}
	// Create an HTTP GET request with the specified URL
	req, httperr := http.NewRequest("GET", url, nil)
	if httperr != nil {
		zlog.MiSec().Fatal().Err(httperr).Msgf("Error creating request: %v\n", httperr)
		return httperr

	}

	// Set the HTTP version to 1.1
	req.Proto = "HTTP/1.1"

	// Perform the HTTP GET request
	resp, clienterr := client.Do(req)
	if clienterr != nil {
		zlog.MiSec().Fatal().Err(clienterr).Msgf("Error performing request: %v\n", clienterr)
		return clienterr

	}
	defer resp.Body.Close()

	filePath := outDir + "/" + config.ReleaseVersion + ".yaml"
	//Read the response body
	//Create or open the local file for writing
	file, fileerr := os.Create(filePath)
	if fileerr != nil {
		zlog.MiSec().Fatal().Err(fileerr).Msgf("Error while creating release manifest file.")
		return fileerr
	}
	defer file.Close()

	// Copy the response body to the local file
	_, copyErr := io.Copy(file, resp.Body)
	if copyErr != nil {
		zlog.MiSec().Fatal().Err(copyErr).Msgf("Error while coping content ")
	}
	zlog.MiSec().Info().Msg("File downloaded")
	return nil

}

// Ensure that pigz and qemu-img are installed
func ensureDependencies() error {
	if !commandExists("pigz") {
		zlog.MiSec().Info().Msg("Installing pigz...")
		err := installPackage("pigz")
		if err != nil {
			return err
		}
	}

	if !commandExists("qemu-img") {
		zlog.MiSec().Info().Msg("Installing qemu-utils...")
		err := installPackage("qemu-utils")
		if err != nil {
			return err
		}
	}

	return nil
}

// Check if a command is available in the system
func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// Install a package using the system's package manager
func installPackage(packageName string) error {
	cmdStr := fmt.Sprintf("sudo apt-get install -y %s", packageName) // Assuming Ubuntu Linux
	cmd := exec.Command("sh", "-c", cmdStr)
	return cmd.Run()
}

// Download an image from a URL and save it to a file
func downloadImage(url string, fileName string) error {
	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, response.Body)
	return err
}

// Convert an image to raw format using qemu-img
func convertImage(inputFile, outputFile string) error {
	cmdStr := fmt.Sprintf("qemu-img convert -O raw %s %s", inputFile, outputFile)
	cmd := exec.Command("sh", "-c", cmdStr)
	return cmd.Run()
}

// Compress an image using pigz
func compressImage(inputFile, outputFile string) error {
	cmdStr := fmt.Sprintf("pigz < %s > %s", inputFile, outputFile)
	cmd := exec.Command("sh", "-c", cmdStr)
	return cmd.Run()
}

func DownloadUbuntuImage(imageUrl string, format string, file string, fileName string) error {
	zlog.Info().Msgf("Inside Download and Raw form conversion...")
	// Check and install dependencies if necessary
	if err := ensureDependencies(); err != nil {
		zlog.MiSec().Fatal().Err(err).Msgf("Error installing dependencies: %v", err)
		return err
	}

	// Download the image
	if err := downloadImage(imageUrl, format); err != nil {
		zlog.MiSec().Fatal().Err(err).Msgf("Error downloading image:%v", err)
		return err
	}

	// Convert the image to raw format
	if err := convertImage("image.img", "image.raw"); err != nil {
		zlog.MiSec().Fatal().Err(err).Msgf("Error converting image:%v", err)
		return err
	}

	// Compress the raw image using pigz
	if err := compressImage("image.raw", file); err != nil {
		zlog.MiSec().Fatal().Err(err).Msgf("Error compressing image:%v", err)
		return err
	}

	// Clean up temporary files
	if err := os.Remove("image.img"); err != nil {
		zlog.MiSec().Fatal().Err(err).Msgf("Error removing temporary file: image.img: %v", err)
	}
	if err := os.Remove("image.raw"); err != nil {
		zlog.MiSec().Fatal().Err(err).Msgf("Error removing temporary file: image.raw %v", err)
	}
	exists, patherr := PathExists(file)
	if patherr != nil {
		zlog.MiSec().Info().Msgf("Error checking path %v", patherr)
	}
	if exists {
		zlog.MiSec().Info().Msg("image raw file Path exists")
		exists, err := PathExists("/data")
		if err != nil {
			zlog.MiSec().Info().Msgf("Error checking path %v", err)
		}
		if exists {
			zlog.MiSec().Info().Msg("PVC Path exists")
			pvcFilePath := "/data" + "/" + fileName
			cmd := exec.Command("mv", file, pvcFilePath)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			err := cmd.Run()
			if err != nil {
				zlog.MiSec().Info().Msgf("error running 'mv' command: %v", err)
			}
		} else {
			zlog.MiSec().Info().Msg("PVC Path not exists")
		}

	} else {
		zlog.MiSec().Info().Msg("image raw file Path not exists:")

	}
	zlog.MiSec().Info().Msg("File downloaded and converted into raw format")
	return nil

}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil // path exists
	}
	if os.IsNotExist(err) {
		return false, nil // path does not exist
	}
	return false, err // an error occurred (other than not existing)
}
