package auth

import (
	"errors"
	"time"
)

func EnsureFresh(store *Store) (OAuth1Token, OAuth2Token, error) {
	oauth1, oauth2, ok, err := store.LoadTokens()
	if err != nil {
		return oauth1, oauth2, err
	}
	if !ok {
		return oauth1, oauth2, errors.New("not authenticated; run `garmin auth login`")
	}
	now := time.Now()
	if !oauth2.Expired(now) {
		return oauth1, oauth2, nil
	}
	client, err := NewSSOClient()
	if err != nil {
		return oauth1, oauth2, err
	}
	refreshed, err := client.RefreshOAuth2(oauth1)
	if err != nil {
		return oauth1, oauth2, err
	}
	if err := store.SaveTokens(oauth1, refreshed); err != nil {
		return oauth1, refreshed, err
	}
	return oauth1, refreshed, nil
}
