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
	Domain    string   `json:"domain"`
	Subdomain []string `json:"subdomain"`
}

func processEntry(subdomain string, domain string, apikey string, externalIP string, uuid string) {
	log.Print("processing " + subdomain)
	hostname := subdomain + "." + domain
	ips, err := net.LookupHost(hostname)
	if err != nil {
		return
	}
	for _, ip := range ips {
		if ip == externalIP {
			log.Printf("found %s as %s, no need to update", hostname, ip)
			return
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
		log.Fatal(string(reply.Code) + " " + reply.Message)
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
	var hosts configuration
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
	uuid, err := getUUID(hosts.Key, hosts.Domain)

	var wg sync.WaitGroup
	for _, subdomain := range hosts.Subdomain {
		wg.Add(1)
		go func(subdomain string) {
			defer wg.Done()
			processEntry(subdomain, hosts.Domain, hosts.Key, externalIP.String(), uuid)
		}(subdomain)
	}
	wg.Wait()

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
