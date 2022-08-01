package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	externalip "github.com/glendc/go-external-ip"
)

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
		fmt.Println("-zone-id and -domain are both required")
		os.Exit(1)
	}

	var changes []*route53.Change
	consensus := externalip.DefaultConsensus(nil, nil)

	for _, addr_type := range []string{"A", "AAAA"} {
		addr := getPublicIP(addr_type, consensus)
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

func getPublicIP(addrType string, consensus *externalip.Consensus) string {
	if addrType == "A" {
		consensus.UseIPProtocol(4)
	} else {
		consensus.UseIPProtocol(6)
	}
	ip, err := consensus.ExternalIP()
	if err != nil {
		fmt.Printf("Could not determine your %s address\n", addrType)
		return ""
	}
	return ip.String()
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
