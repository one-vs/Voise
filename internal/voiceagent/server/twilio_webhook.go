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
func HandleIncomingCall(path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		builder := NewTwiMLBuilder()
		// Construct absolute WSS URL
		scheme := "wss"
		if r.TLS == nil {
			// Note: Twilio requires WSS in production.
			// This is a simplification for local development/testing.
		}
		wssURL := fmt.Sprintf("%s://%s%s", scheme, r.Host, path)

		// In a real app, you would generate a unique session token
		// and store it in Redis or a similar store.
		token := "temp-" + r.FormValue("CallSid")
		params := map[string]string{
			"auth_token": token,
		}
		builder.ConnectStream(wssURL, params)

		w.Header().Set("Content-Type", "text/xml")
		fmt.Fprint(w, builder.Build())
	}
}

// TwilioSignatureMiddleware is a middleware that validates Twilio signatures.
func TwilioSignatureMiddleware(authToken string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		scheme := "https"
		if r.TLS == nil {
			scheme = "http"
		}
		url := fmt.Sprintf("%s://%s%s", scheme, r.Host, r.URL.Path)

		if !ValidateTwilioSignature(r, authToken, url) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}
