package outbound

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type Initiator struct {
	AccountSid string
	AuthToken  string
}

func NewInitiator(sid, token string) *Initiator {
	return &Initiator{AccountSid: sid, AuthToken: token}
}

func (i *Initiator) InitiateCall(ctx context.Context, from, to, callbackURL string) (string, error) {
	apiURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Calls.json", i.AccountSid)

	data := url.Values{}
	data.Set("From", from)
	data.Set("To", to)
	data.Set("Url", callbackURL)

	req, _ := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(data.Encode()))
	req.SetBasicAuth(i.AccountSid, i.AuthToken)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("twilio API returned status %d", resp.StatusCode)
	}

	return "call_sid_placeholder", nil
}
