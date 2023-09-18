// Package sitehost is the SiteHost libdns provider.
package sitehost

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/libdns/libdns"
	"github.com/sitehostnz/gosh/pkg/api/dns"
)

// Provider implements the libdns interfaces for SiteHost.
// TODO: Support pagination and retries, handle rate limits.
type Provider struct {
	ClientID  string `json:"client_id,omitempty"`
	APIKey    string `json:"apikey,omitempty"`
	Host      string `json:"host,omitempty"`
	zonesMu   sync.Mutex
	dnsClient *dns.Client
}

// GetRecords lists all the records in the zone.
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	zone = strings.Trim(zone, ".")
	return p.getRecords(ctx, zone)
}

// AppendRecords adds records to the zone. It returns the records that were added.
func (p *Provider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	zone = strings.Trim(zone, ".")
	return p.createRecords(ctx, zone, records)
}

// DeleteRecords deletes the records from the zone. If a record does not have an ID,
// it will be looked up. It returns the records that were deleted.
func (p *Provider) DeleteRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	// Create the variable we will use for the response
	var deleted []libdns.Record
	// Creat the variable used to contain the list of records to delete
	var deleteQueue []libdns.Record

	zone = strings.Trim(zone, ".")

	// Iterate over all the records passed in
	for _, r := range records {
		// If the record has an ID, add it to the delete queue
		if r.ID != "" {
			deleteQueue = append(deleteQueue, r)
			continue
		}

		// We don't have an ID, try to find the record
		matches, err := p.getRecordsMatch(ctx, zone, r, true)
		if err != nil {
			return nil, err
		}

		// Iterate over the matches adding them to the queue
		for _, m := range matches {
			deleteQueue = append(deleteQueue, m)
		}
	}

	// Iterate over the delete queue deleting the records
	for _, r := range deleteQueue {
		if err := p.deleteRecord(ctx, zone, r); err != nil {
			return deleted, err

		}
		deleted = append(deleted, r)
	}

	// Return the list of deleted records
	return deleted, nil
}

// SetRecords sets the records in the zone, either by updating existing records
// or creating new ones. It returns the updated records.
func (p *Provider) SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	// Initialize our response variable
	var results []libdns.Record

	zone = strings.Trim(zone, ".")

	// Iterate over the supplied records
	for _, rec := range records {
		// If the record ID is missing, try look it up
		if rec.ID == "" {
			// Find records which match
			matches, err := p.getRecordsMatch(ctx, zone, rec, false)
			if err != nil {
				return nil, err
			}

			// If we have no matches, create the record
			if len(matches) == 0 {
				// record doesn't exist; create it
				result, err := p.createRecordReturn(ctx, zone, rec)
				if err != nil {
					return nil, err
				}
				results = append(results, result)
				continue
			}

			// If we have more than 1 match, error
			if len(matches) > 1 {
				return nil, fmt.Errorf("unexpectedly found more than 1 record for %v", rec)
			}

			// The record does exist, fill in the ID so that we can update it
			rec.ID = matches[0].ID
		}

		// Update the existing record
		if err := p.updateRecord(ctx, zone, rec); err != nil {
			return nil, err
		}

		// Append the record to the list of results
		results = append(results, rec)
	}

	// Return the results
	return results, nil
}

// Interface guards
var (
	_ libdns.RecordGetter   = (*Provider)(nil)
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordSetter   = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
)
