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

	// 1. Start with the full URL
	data := url

	// 2. Append all POST parameters, sorted alphabetically by key
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

	// 3. Compute HMAC-SHA1
	mac := hmac.New(sha1.New, []byte(authToken))
	mac.Write([]byte(data))
	computedSignature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return computedSignature == expectedSignature
}

// TwilioSignatureMiddleware is a middleware that validates Twilio signatures.
func TwilioSignatureMiddleware(authToken string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// In a real scenario, you'd want the full URL as Twilio sees it.
		// For simplicity, we assume the URL is reconstructed or passed.
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
