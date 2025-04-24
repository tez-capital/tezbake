package util

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/tez-capital/tezbake/constants"
)

func ResolveAttestationProfile(pkh string) (string, error) {
	url := fmt.Sprintf("%sv1/delegates/%s", constants.TzktConsensusKeyCheckingEndpoint, pkh)
	var delegate struct {
		Active bool `json:"active"`
	}
	log.Infof("Checking if key %s is a delegate...", pkh)
	log.Debugf("Checking through %s...", url)
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
				log.Warnf("Failed to decode response for key %s: %v", pkh, err)
			case !delegate.Active:
				log.Warnf("Key %s is not active", pkh)
			}
		default:
			log.Warnf("Failed to check whether key %s is a delegate: %s", pkh, response.Status)
		}
		return pkh, nil
	}

	url = fmt.Sprintf("%sv1/operations/update_consensus_key?publicKeyHash=%s&select=sender&sort.desc=level&limit=1", constants.TzktConsensusKeyCheckingEndpoint, pkh)
	var bakers []struct {
		Address string `json:"address"`
	}
	log.Infof("Checking if key %s is a consensus key...", pkh)
	log.Debugf("Checking through %s...", url)

	response, err = client.Get(url)
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
	return "", fmt.Errorf("failed to resolve attestation profile for key %s", pkh)
}
