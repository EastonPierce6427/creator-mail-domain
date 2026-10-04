package creator

import (
	"context"
	"encoding/json"
	"testing"
)

type fakeGateway struct{ calls []string }

func (f *fakeGateway) AddDomain(context.Context, string) (string, error) {
	f.calls = append(f.calls, "add")
	return "zone_12", nil
}
func (f *fakeGateway) UpsertRecord(_ context.Context, _ string, kind, name, _ string) error {
	f.calls = append(f.calls, kind+":"+name)
	return nil
}
func (f *fakeGateway) VerifyEmailDomain(context.Context, string) (string, error) {
	f.calls = append(f.calls, "verify")
	return "verified", nil
}
func (f *fakeGateway) SendEmail(context.Context, string, string, string) (string, error) {
	f.calls = append(f.calls, "send")
	return "msg_7", nil
}

func TestDeliverWritesMailRecordsBeforeSending(t *testing.T) {
	tests := []struct {
		name        string
		input       Delivery
		wantMessage string
	}{
		{name: "asset with update", input: Delivery{Subscriber: "member@example.com", Domain: "maker.example", AssetURL: "https://files.example/guide.pdf", Update: "New guide"}, wantMessage: "msg_7"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &fakeGateway{}
			got, err := Deliver(context.Background(), fake, tt.input)
			if err != nil {
				t.Fatal(err)
			}
			if got.MessageID != tt.wantMessage {
				t.Fatalf("message_id = %q", got.MessageID)
			}
			want := []string{"add", "TXT:@", "CNAME:infrai._domainkey", "TXT:_dmarc", "verify", "send"}
			for i, call := range want {
				if fake.calls[i] != call {
					t.Fatalf("call %d = %q, want %q", i, fake.calls[i], call)
				}
			}
		})
	}
}

func TestEmailDomainVerificationResponseShape(t *testing.T) {
	var data verificationResponse
	if err := json.Unmarshal([]byte(`{"domain":"maker.example","status":"verified"}`), &data); err != nil {
		t.Fatal(err)
	}
	if data.Status != "verified" {
		t.Fatalf("status = %q, want verified", data.Status)
	}
}
