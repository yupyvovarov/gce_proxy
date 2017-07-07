package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	randomdata "github.com/Pallinder/go-randomdata"
	"golang.org/x/oauth2/google"
	compute "google.golang.org/api/compute/v1"
)

const apiURL string = "https://www.googleapis.com/compute/v1/projects/"

var projectID, region, zone, machineType, imageID, diskType *string
var diskSize *int64

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

// User to create in GCE instance
type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// PublicIP created instance
type InstanceName struct {
	Name string `json:"name"`
}

// PublicIP created instance
type PublicIP struct {
	IP string `json:"ip"`
}

// Log http requests
func logging(f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.URL.Path)
		f(w, r)
	}
}

// Insert an instance into GCE
func insertInstance(instanceName, username, password *string) {
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
	startupScript := fmt.Sprintf("#! /bin/bash\n\nuseradd -p $(openssl passwd -1 %s) %s\necho '%s ALL=(ALL) NOPASSWD: ALL' > /etc/sudoers.d/%s\nsed -i 's|[#]*PasswordAuthentication no|PasswordAuthentication yes|g' /etc/ssh/sshd_config\nsed -i 's|UsePAM no|UsePAM yes|g' /etc/ssh/sshd_config\nsystemctl restart sshd.service", *password, *username, *username, *username)

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
		Metadata: &compute.Metadata{
			Items: []*compute.MetadataItems{
				&compute.MetadataItems{
					Key:   "startup-script",
					Value: &startupScript,
				},
			},
		},
	}

	resp, err := computeService.Instances.Insert(*projectID, *zone, rb).Context(ctx).Do()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("%#v\n", resp)
}

// getInstanceIP public of instance
func getInstanceIP(instanceName *string) string {
	ctx := context.Background()

	c, err := google.DefaultClient(ctx, compute.CloudPlatformScope)
	if err != nil {
		log.Fatal(err)
	}

	computeService, err := compute.New(c)
	if err != nil {
		log.Fatal(err)
	}

	var status, ip string
	var resp *compute.Instance
	for {
		if !strings.HasPrefix(status, "RUNNING") {
			resp, err = computeService.Instances.Get(*projectID, *zone, *instanceName).Context(ctx).Do()
			if err != nil {
				log.Fatal(err)
			}
			status = resp.Status
		} else {
			break
		}
	}

	ip = resp.NetworkInterfaces[0].AccessConfigs[0].NatIP
	fmt.Println(*instanceName, ":", ip)
	return ip
}

// healthcheck endpoint
// GET /healthcheck
func healthcheck(w http.ResponseWriter, r *http.Request) {
}

// Create GCE instance endpoint
// POST /v1/instances/create
func instancesCreate(w http.ResponseWriter, r *http.Request) {
	var user User
	json.NewDecoder(r.Body).Decode(&user)
	instanceName := strings.ToLower(randomdata.SillyName())
	insertInstance(&instanceName, &user.Username, &user.Password)
	ip := PublicIP{
		IP: getInstanceIP(&instanceName),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ip)
}

// Get instance IP address endpoint
// POST /v1/instances/ip
func instancesIP(w http.ResponseWriter, r *http.Request) {
	var instance InstanceName
	json.NewDecoder(r.Body).Decode(&instance)
	ip := PublicIP{
		IP: getInstanceIP(&instance.Name),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ip)
}

func main() {
	// Load configuration file
	var configFile = flag.String("config", "config.json", "Location of the config file.")
	var port = flag.Int("port", 8080, "Listen port.")
	flag.Parse()
	file, _ := os.Open(*configFile)
	decoder := json.NewDecoder(file)
	configuration := Configuration{}
	err := decoder.Decode(&configuration)
	if err != nil {
		log.Fatal(err)
	}
	// Set $GOOGLE_APPLICATION_CREDENTIALS environment variable
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", configuration.AccountKey)
	// Set Compute Engine properties for new instance
	projectID = &configuration.ProjectID
	region = &configuration.Region
	zone = &configuration.Zone
	machineType = &configuration.MachineType
	imageID = &configuration.ImageID
	diskType = &configuration.DiskType
	diskSize = &configuration.DiskSize
	fmt.Println("Configuration loaded\nStarting service")

	fmt.Println(time.Now(), " Google Cloud Platform Proxy is running port", *port)
	http.HandleFunc("/healthcheck", logging(healthcheck))
	http.HandleFunc("/v1/instances/create", logging(instancesCreate))
	http.HandleFunc("/v1/instances/ip", logging(instancesIP))
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", *port), nil))
}
