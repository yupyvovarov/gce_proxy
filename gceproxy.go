package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2/google"
	compute "google.golang.org/api/compute/v1"

	randomdata "github.com/Pallinder/go-randomdata"
)

const apiURL string = "https://www.googleapis.com/compute/v1/projects/"

var projectID, region, zone, machineType, imageID, diskType *string
var diskSize *int64

// User to create in GCE instance
type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Configuration Google Compute
type Configuration struct {
	ProjectID   string `json:"projectid"`
	Region      string `json:"region"`
	Zone        string `json:"zone"`
	MachineType string `json:"machinetype"`
	ImageID     string `json:"imageid"`
	AccountKey  string `json:"accountkey"`
	DiskType    string `json:"disktype"`
	DiskSize    int64  `json:"disksize"`
}

// Load config file and set environment variables and cloud image properties
func init() {
	file, _ := os.Open("config.json")
	decoder := json.NewDecoder(file)
	configuration := Configuration{}
	err := decoder.Decode(&configuration)
	if err != nil {
		fmt.Println("error:", err)
	}
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", configuration.AccountKey)
	projectID = &configuration.ProjectID
	region = &configuration.Region
	zone = &configuration.Zone
	machineType = &configuration.MachineType
	imageID = &configuration.ImageID
	diskType = &configuration.DiskType
	diskSize = &configuration.DiskSize
	fmt.Println("Configuration loaded\nStarting service")
}

// Log http requests
func logging(f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.URL.Path)
		f(w, r)
	}
}

// healthcheck
func healthcheck(w http.ResponseWriter, r *http.Request) {
}

// Insert an instance into GCE
func insertInstance(instanceName *string) {
	ctx := context.Background()

	c, err := google.DefaultClient(ctx, compute.CloudPlatformScope)
	if err != nil {
		log.Fatal(err)
	}

	computeService, err := compute.New(c)
	if err != nil {
		log.Fatal(err)
	}

	prefix := apiURL + *projectID
	imageURL := apiURL + *imageID

	rb := &compute.Instance{
		Name:        *instanceName,
		Description: "Instance create with GCE Proxy tool",
		Zone:        *zone,
		MachineType: prefix + "/zones/" + *zone + "/machineTypes/" + *machineType,
		NetworkInterfaces: []*compute.NetworkInterface{
			&compute.NetworkInterface{
				Network:    prefix + "/global/networks/default",
				Subnetwork: prefix + "/regions/" + *region + "/subnetworks/default",
				AccessConfigs: []*compute.AccessConfig{
					&compute.AccessConfig{
						Name: "External NAT",
						Type: "ONE_TO_ONE_NAT"},
				},
			},
		},
		Disks: []*compute.AttachedDisk{
			{
				Type:       "PERSISTENT",
				Boot:       true,
				Mode:       "READ_WRITE",
				AutoDelete: true,
				DeviceName: *instanceName,
				InitializeParams: &compute.AttachedDiskInitializeParams{
					SourceImage: imageURL,
					DiskType:    prefix + "/zones/" + *zone + "/diskTypes/" + *diskType,
					DiskSizeGb:  *diskSize,
				},
			},
		},
	}

	resp, err := computeService.Instances.Insert(*projectID, *zone, rb).Context(ctx).Do()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%#v\n", resp)
}

// Create GCE instance
func createInstance(w http.ResponseWriter, r *http.Request) {
	// var user User
	// json.NewDecoder(r.Body).Decode(&user)
	// fmt.Fprintf(w, "User \"%s\" with password \"%s\" created.", user.Username, user.Password)
	instanceName := strings.ToLower(randomdata.SillyName())
	insertInstance(&instanceName)
}

func main() {
	fmt.Println(time.Now(), " Google Cloud Platform Proxy is running.")
	http.HandleFunc("/healthcheck", logging(healthcheck))
	http.HandleFunc("/v1/instances/create", logging(createInstance))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
