package server

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"net/http"
	"sort"
)

// ValidateTwilioSignature validates the Twilio signature of a request.
func ValidateTwilioSignature(r *http.Request, authToken string, url string) bool {
	expectedSignature := r.Header.Get("X-Twilio-Signature")
	if expectedSignature == "" {
		return false
	}

	data := url
	if r.Method == http.MethodPost {
		r.ParseForm()
		keys := make([]string, 0, len(r.PostForm))
		for k := range r.PostForm {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			data += k + r.PostForm.Get(k)
		}
	}

	mac := hmac.New(sha1.New, []byte(authToken))
	mac.Write([]byte(data))
	computedSignature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return computedSignature == expectedSignature
}

// HandleIncomingCall handles incoming Twilio voice calls.
func HandleIncomingCall(wssURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		builder := NewTwiMLBuilder()
		// In a real app, you would generate a unique session token
		params := map[string]string{
			"auth_token": "temporary_token",
		}
		builder.ConnectStream(wssURL, params)

		w.Header().Set("Content-Type", "text/xml")
		fmt.Fprint(w, builder.Build())
	}
}
