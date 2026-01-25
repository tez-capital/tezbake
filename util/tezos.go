package util

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/tez-capital/tezbake/constants"
	"github.com/tez-capital/tezbake/logging"
)

func resolveSecondaryKey(pkh string) (string, error) {
	url := fmt.Sprintf("%sv1/operations/update_secondary_key?publicKeyHash=%s&select=sender&sort.desc=level&limit=1", constants.TzktConsensusKeyCheckingEndpoint, pkh)
	var bakers []struct {
		Address string `json:"address"`
	}
	logging.Infof("Checking if key %s is a secondary key...", pkh)
	logging.Debugf("Checking through %s...", url)

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: 30 * time.Second, // enforce dial timeout
			}).DialContext,
			TLSHandshakeTimeout: 30 * time.Second,
		},
		Timeout: 2 * time.Minute,
	}
	response, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to check whether key %s is a consensus key: %s", pkh, response.Status)
	}
	err = json.NewDecoder(response.Body).Decode(&bakers)
	if err != nil {
		return "", err
	}
	if len(bakers) > 0 {
		return bakers[0].Address, nil
	}
	return "", fmt.Errorf("key %s is not a consensus key", pkh)
}

func ResolveAttestationProfile(pkh string) (string, error) {
	url := fmt.Sprintf("%sv1/delegates/%s", constants.TzktConsensusKeyCheckingEndpoint, pkh)
	var delegate struct {
		Active bool `json:"active"`
	}
	logging.Infof("Checking if key %s is a delegate...", pkh)
	logging.Debugf("Checking through %s...", url)
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: 30 * time.Second, // enforce dial timeout
			}).DialContext,
			TLSHandshakeTimeout: 30 * time.Second,
		},
		Timeout: 2 * time.Minute,
	}
	response, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	switch {
	case response.StatusCode == http.StatusNotFound:
	case response.StatusCode == http.StatusNoContent:

	default:
		switch {
		case response.StatusCode == http.StatusOK:
			err = json.NewDecoder(response.Body).Decode(&delegate)
			switch {
			case err != nil:
				logging.Warnf("Failed to decode response for key %s: %v", pkh, err)
			case !delegate.Active:
				logging.Warnf("Key %s is not active", pkh)
			}
		default:
			logging.Warnf("Failed to check whether key %s is a delegate: %s", pkh, response.Status)
		}
		return pkh, nil
	}

	secondaryKeyOwner, err := resolveSecondaryKey(pkh)
	if err == nil {
		logging.Infof("Key %s is a secondary key for delegate %s", pkh, secondaryKeyOwner)
		return secondaryKeyOwner, nil
	}

	return "", fmt.Errorf("failed to resolve attestation profile for key %s", pkh)
}
