package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/imroc/req"
)

func IsIPv4(address string) bool {
	return strings.Count(address, ":") < 2
}

func IsIPv6(address string) bool {
	return strings.Count(address, ":") >= 2
}

func main() {
	hostname := flag.String("hostname", "", "Hostname of this machine")
	zoneID := flag.String("zone-id", "", "The ZoneID of your Route53 zone")
	domain := flag.String("domain", "", "The domain of your Route53 zone")
	flag.Parse()

	if *hostname == "" {
		h, err := os.Hostname()
		if err != nil {
			fmt.Println("Could not determine hostname. Please provide the -hostname flag")
			os.Exit(1)
		}
		hostname = &strings.Split(h, ".")[0]
	}

	if *zoneID == "" || *domain == "" {
		fmt.Println("-iface, -zone-id and -domain are all required")
		os.Exit(1)
	}

	var changes []*route53.Change

	for _, addr_type := range []string{"A", "AAAA"} {
		addr := getPublicIP(addr_type)
		if addr != "" {
			changes = append(changes, createChange(*hostname, *domain, addr, addr_type))
		}
	}

	sess, err := session.NewSession()
	if err != nil {
		fmt.Println("failed to create session,", err)
		return
	}

	svc := route53.New(sess)
	pushChanges(svc, changes, *zoneID)

}

type ApiResponse struct {
	IP string `json:"ip"`
}

func getPublicIP(addrType string) string {
	var url string
	if addrType == "A" {
		url = "https://api4.my-ip.io/ip.json"
	} else {
		url = "https://api6.my-ip.io/ip.json"
	}
	r, err := req.Get(url)
	if err != nil || r.Response().StatusCode != 200 {
		fmt.Printf("Could not determine your %s address\n", addrType)
		return ""
	}
	data := ApiResponse{}
	r.ToJSON(&data)
	return data.IP
}

func createChange(name string, domain string, addr string, addr_type string) *route53.Change {
	fmt.Printf("%s: %s\n", addr_type, addr)
	return &route53.Change{
		Action: aws.String("UPSERT"),
		ResourceRecordSet: &route53.ResourceRecordSet{
			Name: aws.String(fmt.Sprintf("%s.%s", name, domain)),
			Type: aws.String(addr_type),
			ResourceRecords: []*route53.ResourceRecord{
				{
					Value: aws.String(addr),
				},
			},
			TTL: aws.Int64(300),
		},
	}
}

func pushChanges(svc *route53.Route53, changes []*route53.Change, zoneID string) {

	params := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{
			Changes: changes,
		},
		HostedZoneId: aws.String(zoneID),
	}
	_, err := svc.ChangeResourceRecordSets(params)

	if err != nil {
		panic(err)
	}
	fmt.Println("Pushed to R53")
}
