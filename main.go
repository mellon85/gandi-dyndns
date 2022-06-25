package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"sync"

	externalip "github.com/GlenDC/go-external-ip"
)

const gandiHost = "https://dns.api.gandi.net"
const endpoint = gandiHost + "/api/v5"

type configuration struct {
	Key       string   `json:"key"`
	Subdomain []string `json:"subdomain"`
}

func processEntry(subdomain string, domain string, apikey string, externalIP string, uuid string) {
	hostname := subdomain + "." + domain
	log.Printf("Processing %s", hostname)
	ips, err := net.LookupHost(hostname)
	if err != nil {
		log.Printf("Hostname not found %s", hostname)
		return
	}
	for _, ip := range ips {
		if ip == externalIP {
			log.Printf("Found %s as %s, no need to update", hostname, ip)
			return
		} else {
			log.Printf("Found %s as %s, needs to be updated", hostname, ip)
		}
	}

	type gandiMessage struct {
		IPs []string `json:"rrset_values"`
	}
	var message = gandiMessage{IPs: []string{externalIP}}
	data, err := json.Marshal(&message)
	if err != nil {
		log.Fatal("Failed to encode", message)
	}
	url := endpoint + "/zones/" + uuid + "/records/" + subdomain + "/A"

	_, err = query(url, apikey, data)
	if err != nil {
		log.Printf("Failed to update %s to %s", hostname, externalIP)
	} else {
		log.Printf("Updated %s to %s", hostname, externalIP)
	}
}

func getUUID(apiKey string, domain string) (string, error) {
	url := endpoint + "/domains/" + domain

	respData, err := query(url, apiKey, nil)
	if err != nil {
		return "", err
	}

	type gandiReply struct {
		Code     int    `json:"code"`
		ZoneUUID string `json:"zone_uuid"`
		Message  string `json:"message"`
	}

	reply := gandiReply{}
	err = json.Unmarshal(respData, &reply)
	if err != nil {
		return "", err
	}

	if reply.Code != 0 {
		log.Fatalf("%+v %s", reply.Code, reply.Message)
		return "blop", nil // XXX need proper error
	}

	return reply.ZoneUUID, nil
}

func main() {
	confFile, err := ioutil.ReadFile("conf.json")
	if err != nil {
		log.Fatal("No configuration file")
		return
	}
	hosts := make(map[string]configuration)
	err = json.Unmarshal(confFile, &hosts)
	if err != nil {
		log.Fatal("Failed to read configuration file")
		return
	}

	// identify our IP address to being with
	consensus := externalip.DefaultConsensus(nil, nil)
	externalIP, err := consensus.ExternalIP()
	if err == nil {
		log.Print("external ip: ", externalIP.String())
	}
	eIP := externalIP.String()
	code := 0

	var wg sync.WaitGroup
	for domain, configuration := range hosts {
		uuid, err := getUUID(configuration.Key, domain)
		if err != nil {
			log.Fatal("Failed to update " + domain)
			code += 1
		}
		for _, subdomain := range configuration.Subdomain {
			wg.Add(1)
			go func(subdomain string) {
				defer wg.Done()
				processEntry(subdomain, domain, configuration.Key, eIP, uuid)
			}(subdomain)
		}
	}
	wg.Wait()

	if code > 0 {
		panic("Failed to update a domain")
	}
}

func query(url string, apiKey string, body []byte) ([]byte, error) {
	client := &http.Client{}

	resp, err := client.Get(gandiHost)
	if err != nil {
		return []byte{}, err
	}

	req, err := http.NewRequest("GET", url, bytes.NewReader(body))
	if err != nil {
		return []byte{}, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Api-Key", apiKey)
	resp, err = client.Do(req)
	if err != nil {
		return []byte{}, err
	}
	respData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, err
	}
	return respData, nil
}
