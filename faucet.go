package main

import (
	"bufio"
	"encoding/json"
	"io"
	"net/http"

	"github.com/blockcypher/libgrin/v4/client"
	"github.com/blockcypher/libgrin/v4/core"
	"github.com/blockcypher/libgrin/v4/libwallet"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// The FaucetEndpoint designed to give some grins
type FaucetEndpoint struct {
	ownerAPI *client.SecureOwnerAPI
}

// StartHandler starts the Pool API handler
func (fe *FaucetEndpoint) StartHandler(router *mux.Router) {
	// Initialize and open wallet
	url := "http://127.0.0.1:3420/v3/owner"
	fe.ownerAPI = client.NewSecureOwnerAPI(url)
	if err := fe.ownerAPI.Init(); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Cannot init wallet")
		return
	}
	if err := fe.ownerAPI.Open(nil, ""); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Cannot open wallet")
		return
	}
	// set tor config
	torConfig := libwallet.TorConfig{
		UseTorListener: true,
		SocksProxyAddr: "127.0.0.1:59050",
		SendConfigDir:  "/opt/grin-wallet/tor",
	}
	if err := fe.ownerAPI.SetTorConfig(torConfig); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("PAYOUTS: Cannot set tor config")
		return
	}

	router.HandleFunc("/", fe.giveGrins).Methods("POST")
}

type postRequestGrinsBody struct {
	Address string `json:"address"`
}

// faucetSuccessResponse is the response when a payout is successfully sent
type FaucetSuccessResponse struct {
	Status bool   `json:"status"`
	TxID   string `json:"txid"`
	Amount uint64 `json:"amount"`
}

// faucetSuccessResponse is the response when a payout failed to be sent
type FaucetErrorResponse struct {
	Status bool   `json:"status"`
	Error  string `json:"error"`
}

var amount uint64 = 1000000000

// giveGrins make it rain
func (fe *FaucetEndpoint) giveGrins(w http.ResponseWriter, r *http.Request) {
	// 1. Read the body, limited to maximum size
	bodyReader := bufio.NewReader(r.Body)
	tBuffer := make([]byte, 2000)
	var tString string
	for {
		n, err := bodyReader.Read(tBuffer)
		tString = tString + string(tBuffer[:n])
		if err != nil {
			if err == io.EOF {
				break
			}
			log.WithFields(log.Fields{
				"error": err,
			}).Error("Could not read request body")
			w.WriteHeader(http.StatusBadRequest)
			response := FaucetErrorResponse{Status: false, Error: err.Error()}
			json.NewEncoder(w).Encode(response)
			return
		}
	}

	// 2. Attempt to unmarshal the body
	var t postRequestGrinsBody
	if err := json.Unmarshal([]byte(tString), &t); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("Could not read request body")
		w.WriteHeader(http.StatusBadRequest)
		response := FaucetErrorResponse{Status: false, Error: err.Error()}
		json.NewEncoder(w).Encode(response)
		return
	}

	// 3. Issue send tx
	initSendTxArgs := libwallet.InitTxSendArgs{
		Dest:    t.Address,
		PostTx:  false,
		Fluff:   true,
		SkipTor: false,
	}

	initTxArgs := libwallet.InitTxArgs{
		SrcAcctName:               nil,
		Amount:                    core.Uint64(amount),
		MinimumConfirmations:      2,
		MaxOutputs:                500,
		NumChangeOutputs:          1,
		SelectionStrategyIsUseAll: false,
		TargetSlateVersion:        nil,
		TTLBlocks:                 nil,
		EstimateOnly:              nil,
		SendArgs:                  &initSendTxArgs,
	}

	// 4. InitSendTx
	slate, err := fe.ownerAPI.InitSendTx(initTxArgs)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Cannot init send tx")
		w.WriteHeader(http.StatusBadRequest)
		response := FaucetErrorResponse{Status: false, Error: err.Error()}
		json.NewEncoder(w).Encode(response)
		return
	}

	// 5. Post Tx
	// We put this function in the end so worst case there is no funds lost (tx not broadcasted)
	if err := fe.ownerAPI.PostTx(*slate, true); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("Cannot post transaction")
		w.WriteHeader(http.StatusBadRequest)
		response := FaucetErrorResponse{Status: false, Error: err.Error()}
		json.NewEncoder(w).Encode(response)
		return
	}

	// 6. Finished exit function
	log.WithFields(log.Fields{
		"addr": t.Address,
	}).Info("Successfully sent grins")
	w.WriteHeader(http.StatusOK)
	response := FaucetSuccessResponse{Status: true, TxID: slate.ID.String(), Amount: amount}
	json.NewEncoder(w).Encode(response)
	return
}
