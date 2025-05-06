package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	externalip "github.com/glendc/go-external-ip"
)

func getPublicIP(addrType string, consensus *externalip.Consensus) (string, error) {
	if addrType == "A" {
		consensus.UseIPProtocol(4)
	} else {
		consensus.UseIPProtocol(6)
	}
	ip, err := consensus.ExternalIP()
	return ip.String(), err
}

func createChange(hostname string, domain string, addr string, addrType string, ttl int64) *route53.Change {
	var name string

	if hostname == "@" {
		// Record name for the apex is just the domain
		name = domain
	} else {
		name = fmt.Sprintf("%s.%s", name, domain)
	}

	return &route53.Change{
		Action: aws.String("UPSERT"),
		ResourceRecordSet: &route53.ResourceRecordSet{
			Name: aws.String(name),
			Type: aws.String(addrType),
			ResourceRecords: []*route53.ResourceRecord{
				{
					Value: aws.String(addr),
				},
			},
			TTL: aws.Int64(ttl),
		},
	}
}

func pushChanges(svc *route53.Route53, changes []*route53.Change, zoneID string) error {
	params := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{
			Changes: changes,
		},
		HostedZoneId: aws.String(zoneID),
	}
	_, err := svc.ChangeResourceRecordSets(params)

	return err
}

func main() {
	hostname := flag.String("hostname", "", "Hostname of this machine. Use @ for domain apex")
	zoneID := flag.String("zone-id", "", "The ZoneID of your Route53 zone")
	domain := flag.String("domain", "", "The domain of your Route53 zone")
	ttl := flag.Int64("ttl", 300, "The TTL value for the created records")
	flag.Parse()

	// Log without timestamp
	log.SetFlags(0)

	if *hostname == "" {
		h, err := os.Hostname()
		if err != nil {
			log.Fatalf("Could not determine your hostname. Please provide the -hostname flag")
		}
		hostname = &strings.Split(h, ".")[0]
	}

	if *zoneID == "" || *domain == "" {
		fmt.Println("-zone-id and -domain are both required")
		os.Exit(1)
	}

	var changes []*route53.Change
	consensus := externalip.DefaultConsensus(nil, nil)

	for _, addrType := range []string{"A", "AAAA"} {
		addr, err := getPublicIP(addrType, consensus)
		if err != nil {
			log.Printf("Could not determine a address for the %s: %s\n", addrType, err)
			continue
		}
		log.Printf("Discovered addr for %s record: %s\n", addrType, addr)
		changes = append(changes, createChange(*hostname, *domain, addr, addrType, *ttl))
	}

	if len(changes) == 0 {
		log.Fatalln("Couldn't determine a single public IP for this machine. Abort.")
	}

	sess, err := session.NewSession()
	if err != nil {
		fmt.Println("Failed to create a AWS session,", err)
		os.Exit(1)
	}

	svc := route53.New(sess)
	err = pushChanges(svc, changes, *zoneID)
	if err != nil {
		log.Fatalf("Error pushing changes to AWS: %s\n", err)
	}
	log.Printf("Pushed %d records to AWS\n", len(changes))
}
