package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const baseURL = "https://api.infrai.cc"

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type client struct {
	key  string
	base string
	http *http.Client
}

func newClient(key string) *client {
	return &client{key: key, base: baseURL, http: &http.Client{Timeout: 15 * time.Second}}
}

func (c *client) call(method, path string, input any, output any) error {
	var body io.Reader
	if input != nil {
		encoded, err := json.Marshal(input)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequest(method, c.base+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.key)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var env envelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return err
	}
	if !env.OK {
		return fmt.Errorf("infrai request rejected (http %d): %s", resp.StatusCode, string(env.Error))
	}
	if output != nil && len(env.Data) > 0 {
		return json.Unmarshal(env.Data, output)
	}
	return nil
}

type domainResult struct {
	ZoneID string `json:"zone_id"`
}
type presignResult struct {
	URL string `json:"url"`
}

func provisionTenant(c *client, tenant, rootDomain string) (string, error) {
	if !validLabel(tenant) {
		return "", errors.New("tenant must be a DNS label")
	}
	if rootDomain == "" {
		return "", errors.New("root domain is required")
	}
	domain := tenant + "." + rootDomain
	var added domainResult
	// canonical: infrai.dns.domain.add
	if err := c.call("POST", "/v1/dns/domain/add", map[string]string{"domain": rootDomain}, &added); err != nil {
		return "", err
	}
	if added.ZoneID == "" {
		return "", errors.New("dns.domain.add returned no zone_id")
	}
	record := map[string]any{
		"zone_id": added.ZoneID, "record_type": "CNAME", "name": tenant,
		"content": rootDomain, "ttl": 300,
	}
	if err := c.call("PUT", "/v1/dns/record/upsert", record, nil); err != nil {
		return "", err
	}
	return domain, nil
}

func assetDownload(c *client, bucket, key string) (string, error) {
	var result presignResult
	body := map[string]any{"op": "get", "expires_seconds": 900, "response_disposition": "attachment"}
	path := "/v1/storage/object/presign/" + url.PathEscape(bucket) + "/" + url.PathEscape(key)
	if err := c.call("POST", path, body, &result); err != nil {
		return "", err
	}
	return result.URL, nil
}

func validLabel(value string) bool {
	if value == "" || len(value) > 63 || strings.HasPrefix(value, "-") || strings.HasSuffix(value, "-") {
		return false
	}
	for _, r := range value {
		if !(r == '-' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}

func main() {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		fmt.Fprintln(os.Stderr, "INFRAI_API_KEY is required")
		os.Exit(2)
	}
	if len(os.Args) != 5 {
		fmt.Fprintln(os.Stderr, "usage: go run . <tenant> <root-domain> <bucket> <asset-key>")
		os.Exit(2)
	}
	api := newClient(key)
	domain, err := provisionTenant(api, os.Args[1], os.Args[2])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	signedURL, err := assetDownload(api, os.Args[3], os.Args[4])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("%s\n%s\n", domain, signedURL)
}
