package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/libopenstorage/cloudops"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func main() {

	err := flag.CommandLine.Parse([]string{})
	if err != nil {
		logrus.Warnf("Error parsing flag: %v", err)
	}
	app := cli.NewApp()
	app.Name = "cloudops"
	app.Usage = "Cloud Provider for Kubernetes"
	app.Action = run

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "provider,p",
			Usage: "Cloud provider name",
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatalf("Error starting stork: %v", err)
	}
}

func helloWorld(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello, Kubernetes DaemonSet!")
}

func run(c *cli.Context) {
	logrus.Infof("Starting cloudops")
	providerName := c.String("provider")
	logrus.Infof("Starting %v provider", providerName)

	// sdksocket := fmt.Sprintf("/var/lib/osd/provider/cloudops.sock")
	// DriverAPIBase := "/var/lib/osd/provider/"
	// if _, _, err := server.StartCloudopsAPI(
	// 	providerName, sdksocket,
	// 	DriverAPIBase,
	// 	0,
	// ); err != nil {
	// 	log.Fatalf("Error Starting Cloudops REST server: %v", err)
	// }

	//storageProvider := &StorageProvider{} // Initialize your storage provider here

	router := mux.NewRouter()
	router.HandleFunc("/", helloWorld).Methods("GET")
	router.HandleFunc("/create", CreateVolumeHandler).Methods("POST")

	http.Handle("/", router)
	go http.ListenAndServe(":8090", nil)

	serverURL := "http://localhost:8090" // Replace with your DaemonSet service IP

	// Make an HTTP GET request to the server
	resp, err := http.Get(serverURL)
	if err != nil {
		fmt.Println("Error making GET request:", err)
		return
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return
	}

	// Print the response
	fmt.Printf("Response from REST server: %s\n", string(body))

	/*r := mux.NewRouter()
	r.HandleFunc("/create", CreateVolumeHandler).Methods("POST")

	http.Handle("/", r)
	fmt.Println("Rest server is running on :8090")
	go http.ListenAndServe(":8090", nil)*/

	//	provider = providerNameName

	// Define the payload (template, labels, options) you want to send to the server
	payload := map[string]interface{}{
		"template": "your_template_data",
		"labels": map[string]string{
			"label1": "value1",
			"label2": "value2",
		},
		"options": map[string]string{
			"option1": "value1",
			"option2": "value2",
		},
	}

	// Serialize the payload as JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Error serializing JSON:", err)
		return
	}

	// Send a POST request to the server
	resp, err = http.Post("http://localhost:8090/create", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Error making POST request:", err)
		return
	}

	defer resp.Body.Close()

	fmt.Println("Server response:", resp.Status)

	// 	Send a POST request to the server

	/*payload := map[string]interface{}{
		"template": "your_template_data",
		"labels": map[string]string{
			"label1": "value1",
			"label2": "value2",
		},
		"options": map[string]string{
			"option1": "value1",
			"option2": "value2",
		},
	}

	// Serialize the payload as JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Error serializing JSON:", err)
		return
	}

	resp, err := http.Post("http://localhost:8090/create", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Error making POST request:", err)
		return
	}

	defer resp.Body.Close()

	fmt.Println("Server response:", resp.Status)*/

	stopCh := make(chan struct{})
	<-stopCh
	logrus.Infof("Finished")

}

func CreateVolumeHandler(w http.ResponseWriter, r *http.Request) {
	logrus.Infof("CreateVolu meHandler ")

	storageManager, err := cloudops.NewStorageManager(
		*decisionMatrix,
		cloudops.ProviderType("azure"),
	)

	// Parse the request, call the StorageProvider's Create method, and return a response
	// Example:
	// template := ParseTemplateFromRequest(r)
	// labels := ParseLabelsFromRequest(r)
	// options := ParseOptionsFromRequest(r)

	// volume, err := storageProvider.Create(template, labels, options)
	// if err != nil {
	//     // Handle the error
	//     http.Error(w, "Error creating volume", http.StatusInternalServerError)
	//     return
	// }

	// // Serialize the volume as JSON and send it as the response
	// jsonResponse, err := SerializeVolume(volume)
	// if err != nil {
	//     http.Error(w, "Error serializing volume", http.StatusInternalServerError)
	//     return
	// }

	// w.Header().Set("Content-Type", "application/json")
	// w.WriteHeader(http.StatusCreated)
	// w.Write(jsonResponse)
}
