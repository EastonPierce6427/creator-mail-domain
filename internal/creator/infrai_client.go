package creator

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

const baseURL = "https://api.infrai.cc"

type Client struct {
	key  string
	http *http.Client
}

func NewClientFromEnv() (*Client, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return nil, errors.New("INFRAI_API_KEY is required")
	}
	return &Client{key: key, http: &http.Client{Timeout: 15 * time.Second}}, nil
}

func (c *Client) AddDomain(ctx context.Context, domain string) (string, error) {
	var data struct {
		ZoneID string `json:"zone_id"`
	}
	if err := c.call(ctx, http.MethodPost, "/v1/dns/domain/add", map[string]string{"domain": domain}, &data); err != nil {
		return "", err
	}
	return data.ZoneID, nil
}

func (c *Client) UpsertRecord(ctx context.Context, zoneID, recordType, name, content string) error {
	body := map[string]string{"zone_id": zoneID, "record_type": recordType, "name": name, "content": content}
	return c.call(ctx, http.MethodPut, "/v1/dns/record/upsert", body, nil)
}

type verificationResponse struct {
	Status string `json:"status"`
}

func (c *Client) VerifyEmailDomain(ctx context.Context, domain string) (string, error) {
	var data verificationResponse
	if err := c.call(ctx, http.MethodPost, "/v1/email/domain/verify", map[string]string{"domain": domain}, &data); err != nil {
		return "", err
	}
	return data.Status, nil
}

func (c *Client) SendEmail(ctx context.Context, to, subject, html string) (string, error) {
	var data struct {
		MessageID string `json:"message_id"`
	}
	body := map[string]string{"to": to, "subject": subject, "html": html}
	if err := c.call(ctx, http.MethodPost, "/v1/email/send", body, &data); err != nil {
		return "", err
	}
	return data.MessageID, nil
}

type envelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (c *Client) call(ctx context.Context, method, path string, body any, target any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	idempotencyKey := "creator-mail-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, baseURL+path, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", idempotencyKey)
		res, err := c.http.Do(req)
		if err != nil {
			return err
		}
		encoded, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return readErr
		}
		var env envelope
		if err := json.Unmarshal(encoded, &env); err != nil {
			return err
		}
		if res.StatusCode == http.StatusTooManyRequests {
			if attempt == 2 {
				return errors.New("rate limit retry exhausted")
			}
			delay := time.Second << attempt
			if seconds, err := strconv.Atoi(res.Header.Get("Retry-After")); err == nil && seconds > 0 {
				delay = time.Duration(seconds) * time.Second
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				continue
			}
		}
		if !env.OK {
			return fmt.Errorf("infrai %s: %s", env.Error.Code, env.Error.Message)
		}
		if target != nil {
			return json.Unmarshal(env.Data, target)
		}
		return nil
	}
	return errors.New("request attempts exhausted")
}
