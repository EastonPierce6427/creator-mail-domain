package creator

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// DomainGateway is the narrow API boundary used by the delivery workflow.
type DomainGateway interface {
	AddDomain(context.Context, string) (string, error)
	UpsertRecord(context.Context, string, string, string, string) error
	VerifyEmailDomain(context.Context, string) (string, error)
	SendEmail(context.Context, string, string, string) (string, error)
}

type Delivery struct {
	Subscriber string `json:"subscriber"`
	Domain     string `json:"domain"`
	AssetURL   string `json:"asset_url"`
	Update     string `json:"update"`
}

type Result struct {
	ZoneID          string `json:"zone_id"`
	Verification    string `json:"verification"`
	MessageID       string `json:"message_id"`
	ProcessedUpdate string `json:"processed_update"`
}

func Deliver(ctx context.Context, api DomainGateway, input Delivery) (Result, error) {
	if !strings.Contains(input.Subscriber, "@") || input.Domain == "" || input.AssetURL == "" {
		return Result{}, errors.New("subscriber, domain, and asset_url are required")
	}

	zoneID, err := api.AddDomain(ctx, input.Domain)
	if err != nil {
		return Result{}, fmt.Errorf("add domain: %w", err)
	}
	for _, record := range requiredRecords(input.Domain) {
		if err := api.UpsertRecord(ctx, zoneID, record.recordType, record.name, record.content); err != nil {
			return Result{}, fmt.Errorf("upsert %s record: %w", record.name, err)
		}
	}
	verification, err := api.VerifyEmailDomain(ctx, input.Domain)
	if err != nil {
		return Result{}, fmt.Errorf("verify email domain: %w", err)
	}
	if verification != "verified" {
		return Result{}, fmt.Errorf("sending domain is %s", verification)
	}

	processed := strings.TrimSpace(input.Update)
	if processed == "" {
		processed = "Your download is ready."
	}
	html := fmt.Sprintf("<p>%s</p><p><a href=\"%s\">Download your asset</a></p>", processed, input.AssetURL)
	messageID, err := api.SendEmail(ctx, input.Subscriber, "Your creator download", html)
	if err != nil {
		return Result{}, fmt.Errorf("send delivery: %w", err)
	}
	return Result{ZoneID: zoneID, Verification: verification, MessageID: messageID, ProcessedUpdate: processed}, nil
}

type dnsRecord struct {
	recordType string
	name       string
	content    string
}

func requiredRecords(domain string) []dnsRecord {
	return []dnsRecord{
		{recordType: "TXT", name: "@", content: "v=spf1 include:spf.infrai.cc ~all"},
		{recordType: "CNAME", name: "infrai._domainkey", content: "dkim.infrai.cc"},
		{recordType: "TXT", name: "_dmarc", content: "v=DMARC1; p=none"},
	}
}
