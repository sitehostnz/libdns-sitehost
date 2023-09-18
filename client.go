// Package sitehost is the SiteHost libdns provider.
package sitehost

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/libdns/libdns"
	"github.com/sitehostnz/gosh/pkg/api"
	"github.com/sitehostnz/gosh/pkg/api/dns"
	"github.com/sitehostnz/gosh/pkg/models"
)

// getClient returns an instance of the SiteHost DNS client.
func (p *Provider) getClient() *dns.Client {
	if p.dnsClient != nil {
		return p.dnsClient
	}

	client := api.NewClient(p.APIKey, p.ClientID)
	if p.Host != "" {
		client.BaseURL = &url.URL{
			Scheme: "https",
			Host:   p.Host,
			Path:   "/1.1/",
		}
	}
	p.dnsClient = dns.New(client)

	return p.dnsClient
}

// recordToLibDNSRecord converts a SiteHost record to a libdns record.
func (p *Provider) recordToLibDNSRecord(domain string, record models.DNSRecord) libdns.Record {
	prio, err := strconv.Atoi(record.Priority)
	if err != nil {
		prio = 0
	}
	ttl, err := strconv.Atoi(record.TTL)
	if err != nil {
		ttl = 0
	}

	return libdns.Record{
		ID:       record.ID,
		Type:     record.Type,
		Name:     libdns.RelativeName(record.Name, domain),
		Value:    record.Content,
		TTL:      time.Duration(ttl) * time.Second,
		Priority: prio,
	}
}

// createRecords will create multiple records, but figure out what's changed.
// this is not very pretty but will suffice until changes go in to return the
// record_id of the created record.
func (p *Provider) createRecords(ctx context.Context, domain string, records []libdns.Record) ([]libdns.Record, error) {
	var created []libdns.Record

	// Get the current records for the domain.
	currentRecords, err := p.getRecords(ctx, domain)
	if err != nil {
		return created, err
	}

	// Create a map of the current records.
	currentRecordsMap := make(map[string]libdns.Record)
	for _, record := range currentRecords {
		currentRecordsMap[record.ID] = record
	}

	for _, record := range records {
		err := p.createRecord(ctx, domain, record)
		if err != nil {
			return created, err
		}

		recordsAfter, err := p.getRecords(ctx, domain)
		if err != nil {
			return created, err
		}

		found := false
		for _, recordAfter := range recordsAfter {
			if _, ok := currentRecordsMap[recordAfter.ID]; !ok {
				if recordAfter.Name != record.Name || recordAfter.Type != record.Type || recordAfter.Value != record.Value {
					continue
				}

				found = true
				created = append(created, recordAfter)
				currentRecordsMap[recordAfter.ID] = recordAfter
			}
		}

		if found == false {
			return created, fmt.Errorf("could not find created record")
		}
	}

	return created, nil
}

// createRecord on a DNS zone.
func (p *Provider) createRecord(ctx context.Context, domain string, record libdns.Record) error {
	client := p.getClient()

	response, err := client.AddRecord(ctx, dns.AddRecordRequest{
		ClientID: p.ClientID,
		Domain:   strings.Trim(domain, "."), // Remove trailing dot,
		Type:     record.Type,
		Name:     libdns.AbsoluteName(record.Name, domain),
		Content:  record.Value,
		Priority: strconv.Itoa(record.Priority),
	})
	if err != nil {
		return err
	}

	if response.Status == false {
		return errors.New(response.Msg)
	}

	return nil
}

// createRecordReturn will create a new record and return the created record.
func (p *Provider) createRecordReturn(ctx context.Context, domain string, record libdns.Record) (libdns.Record, error) {
	// Get the current records for the domain.
	currentRecords, err := p.getRecords(ctx, domain)
	if err != nil {
		return libdns.Record{}, err
	}

	// Create a map of the current records.
	currentRecordsMap := make(map[string]libdns.Record)
	for _, record := range currentRecords {
		currentRecordsMap[record.ID] = record
	}

	if err := p.createRecord(ctx, domain, record); err != nil {
		return libdns.Record{}, err
	}

	recordsAfter, err := p.getRecords(ctx, domain)
	if err != nil {
		return libdns.Record{}, err
	}

	for _, recordAfter := range recordsAfter {
		if _, ok := currentRecordsMap[recordAfter.ID]; !ok {
			if recordAfter.Name != record.Name || recordAfter.Type != record.Type || recordAfter.Value != record.Value {
				continue
			}

			return recordAfter, nil
		}
	}

	return libdns.Record{}, errors.New("could not find created record")
}

// deleteRecord on a DNS zone.
func (p *Provider) deleteRecord(ctx context.Context, domain string, record libdns.Record) error {
	client := p.getClient()
	response, err := client.DeleteRecord(ctx, dns.DeleteRecordRequest{
		ClientID: p.ClientID,
		Domain:   strings.Trim(domain, "."), // Remove trailing dot,
		RecordID: record.ID,
	})
	if err != nil {
		return err
	}

	if response.Status == false {
		return errors.New(response.Msg)
	}

	return nil
}

// getRecords for a domain.
func (p *Provider) getRecords(ctx context.Context, domain string) ([]libdns.Record, error) {
	client := p.getClient()
	response, err := client.ListRecords(ctx, dns.ListRecordsRequest{
		Domain: strings.Trim(domain, "."), // Remove trailing dot
	})
	if err != nil {
		return nil, err
	}

	records := make([]libdns.Record, len(response.Return))
	for k, record := range response.Return {
		r := p.recordToLibDNSRecord(domain, record)
		records[k] = r
	}

	return records, nil
}

// getRecordsMatch will return matching records for a libdns record.
func (p *Provider) getRecordsMatch(ctx context.Context, domain string, record libdns.Record, matchContent bool) ([]libdns.Record, error) {
	records, err := p.getRecords(ctx, domain)
	if err != nil {
		return nil, err
	}

	var matches []libdns.Record
	for _, r := range records {
		if r.Name != record.Name {
			continue
		}

		if r.Type != record.Type {
			continue
		}

		if r.Value != record.Value && matchContent {
			continue
		}

		matches = append(matches, r)
	}

	return matches, nil
}

// updateRecord updates a DNS record.
func (p *Provider) updateRecord(ctx context.Context, domain string, record libdns.Record) error {
	client := p.getClient()
	response, err := client.UpdateRecord(ctx, dns.UpdateRecordRequest{
		ClientID: p.ClientID,
		Domain:   strings.Trim(domain, "."), // Remove trailing dot,
		RecordID: record.ID,
		Name:     libdns.AbsoluteName(record.Name, domain),
		Type:     record.Type,
		Content:  record.Value,
		Priority: strconv.Itoa(record.Priority),
	})
	if err != nil {
		return err
	}

	if response.Status == false {
		return errors.New(response.Msg)
	}

	return nil
}
