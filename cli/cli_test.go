package cli

import (
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/030/go-utils"
	log "github.com/sirupsen/logrus"
)

func initializer() {
	cmd := exec.Command("bash", "-c", "docker run -d -p 8081:8081 --name nexus sonatype/nexus3:3.16.1")
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal(err, string(stdoutStderr))
	}
}

func available() {
	for !utils.URLExists(pingURL) {
		log.Info("Nexus not available.")
		time.Sleep(30 * time.Second)
	}
}

func pong() bool {
	pongAvailable := false

	// Generated by curl-to-Go: https://mholt.github.io/curl-to-go

	req, err := http.NewRequest("GET", "http://localhost:8081/service/metrics/ping", nil)
	if err != nil {
		// handle err
	}
	req.SetBasicAuth("admin", "admin123")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// handle err
	}
	defer resp.Body.Close()

	//so
	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		bodyString := string(bodyBytes)

		if bodyString == "pong\n" {
			pongAvailable = true
		}
	}
	return pongAvailable
}

func pongAvailable() {
	for !pong() {
		log.Info("Nexus Pong not returned yet.")
		time.Sleep(3 * time.Second)
	}
}

func submitArtifact(f string) {
	cmd := exec.Command("bash", "-c", "curl -u admin:admin123 -X POST \"http://localhost:8081/service/rest/v1/components?repository=maven-releases\" -H  \"accept: application/json\" -H  \"Content-Type: multipart/form-data\" -F \"maven2.asset1=@file"+f+".pom\" -F \"maven2.asset1.extension=pom\" -F \"maven2.asset2=@file"+f+".jar\" -F \"maven2.asset2.extension=jar\"")
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal(err, string(stdoutStderr))
	}
}

func createArtifact(f string, content string) {
	file, err := os.Create(f)
	if err != nil {
		log.Fatal(err)
	}

	file.WriteString(content)
	defer file.Close()
}

func createPOM(f string) {
	createArtifact("file"+f+".pom", "<project>\n<modelVersion>4.0.0</modelVersion>\n<groupId>file"+f+"</groupId>\n<artifactId>file"+f+"</artifactId>\n<version>1.0.0</version>\n</project>")
}

func createJAR(f string) {
	createArtifact("file"+f+".jar", "some-content")
}

func createArtifactsAndSubmit(f string) {
	createPOM(f)
	createJAR(f)
	submitArtifact(f)
}

func postArtifacts() {
	for i := 1; i <= 60; i++ {
		createArtifactsAndSubmit(strconv.Itoa(i))
	}
}

func cleanupFiles(re string) {
	files, err := filepath.Glob(re)
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range files {
		if err := os.Remove(f); err != nil {
			log.Fatal(err)
		}
	}
}

func cleanup() {
	cmd := exec.Command("bash", "-c", "docker stop nexus && docker rm nexus")
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal(err, string(stdoutStderr))
	}
}

func downloadArtifact(repository string, f string, version string, extension string) {
	// Generated by curl-to-Go: https://mholt.github.io/curl-to-go

	req, err := http.NewRequest("GET", "http://localhost:8081/service/rest/v1/search/assets/download?repository="+repository+"&name="+f+"&version="+version+"&maven.extension="+extension+"", nil)
	if err != nil {
		log.Fatal(err)
	}
	req.SetBasicAuth("admin", "admin123")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	// retrieved somewhere else
	body, err := ioutil.ReadAll(resp.Body)
	createArtifact("downloaded-"+f+"-"+version+"."+extension, string(body))
}

func fileExists(f string) bool {
	_, err := os.Stat(f)
	if err != nil {
		return false
	}
	return true
}

func TestSum(t *testing.T) {
	initializer()
	available()
	pongAvailable()
	postArtifacts()
	defer cleanupFiles("file*")

	//get all downloadUrls
	//curl -X GET "http://localhost:8081/service/rest/v1/search/assets?repository=maven-releases" -H  "accept: application/json" | jq .items[].downloadUrl | wc -l
}

func TestDownloadedFiles(t *testing.T) {
	downloadArtifact("maven-releases", "file20", "1.0.0", "pom")
	downloadArtifact("maven-releases", "file20", "1.0.0", "jar")

	files := []string{"downloaded-file20-1.0.0.pom", "downloaded-file20-1.0.0.jar"}
	for _, f := range files {
		if !fileExists(f) {
			t.Errorf("File %s should exist, but does not.", f)
		}
	}
	defer cleanupFiles("downloaded-*")
	defer cleanup()
}