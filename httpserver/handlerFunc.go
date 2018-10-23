package httpserver

import (
	"encoding/hex"
	"encoding/json"
	emtTypes "github.com/tendermint/ethermint/types"
	tmTypes "github.com/tendermint/tendermint/types"
	"io"
	"net/http"
	"strings"
)

type THandler struct {
	HandlersMap map[string]HandlersFunc
	strategy    *emtTypes.Strategy
}

type HandlersFunc func(http.ResponseWriter, *http.Request)

func NewTHandler(strategy *emtTypes.Strategy) *THandler {
	return &THandler{
		HandlersMap: make(map[string]HandlersFunc),
		strategy:    strategy,
	}
}

func (tHandler *THandler) RegisterFunc() {
	tHandler.HandlersMap["/hello"] = tHandler.Hello
	tHandler.HandlersMap["/test"] = tHandler.test
	tHandler.HandlersMap["/isUpsert"] = tHandler.IsUpsert
	tHandler.HandlersMap["/isRemove"] = tHandler.IsRemove
	tHandler.HandlersMap["/GetPosTable"] = tHandler.GetPosTableData
	tHandler.HandlersMap["/GetAccountMap"] = tHandler.GetAccountMapData
	tHandler.HandlersMap["/GetCurrentValidators"] = tHandler.GetPreBlockValidators
	tHandler.HandlersMap["/GetPreBlockProposer"] = tHandler.GetPreBlockProposer
	tHandler.HandlersMap["/GetAllCandidateValidators"] = tHandler.GetAllCandidateValidatorPool

}

func (tHandler *THandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h, ok := tHandler.HandlersMap[r.URL.String()]; ok {
		h(w, r)
	}
}

func (tHandler *THandler) test(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "this is test function")
}

func (tHandler *THandler) Hello(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("Hello"))
}

// This function will return the used data structure
func (tHandler *THandler) IsUpsert(w http.ResponseWriter, req *http.Request) {
	var nextValidators []*Validator
	for i := 0; i < len(tHandler.strategy.ValidatorSet.NextHeightCandidateValidators); i++ {
		tmPubKey, _ := tmTypes.PB2TM.PubKey(tHandler.strategy.ValidatorSet.NextHeightCandidateValidators[i].PubKey)
		nextValidators = append(nextValidators, &Validator{
			Address:       tHandler.strategy.ValidatorSet.NextHeightCandidateValidators[i].Address,
			PubKey:        tHandler.strategy.ValidatorSet.NextHeightCandidateValidators[i].PubKey,
			Power:         tHandler.strategy.ValidatorSet.NextHeightCandidateValidators[i].Power,
			AddressString: hex.EncodeToString(tmPubKey.Address()),
		})
	}

	PosReceipt := &PTableAll{
		PosItemMap:              tHandler.strategy.PosTable.PosItemMap,
		AccountMapList:          tHandler.strategy.AccountMapList,
		NextCandidateValidators: nextValidators,
	}
	jsonStr, err := json.Marshal(PosReceipt)
	if err != nil {
		w.Write([]byte("error occured when marshal into json"))
	} else {
		w.Write(jsonStr)
	}
}

// This function will return the used data structure
func (tHandler *THandler) IsRemove(w http.ResponseWriter, req *http.Request) {
	var nextValidators []*Validator
	for i := 0; i < len(tHandler.strategy.ValidatorSet.NextHeightCandidateValidators); i++ {
		tmPubKey, _ := tmTypes.PB2TM.PubKey(tHandler.strategy.ValidatorSet.NextHeightCandidateValidators[i].PubKey)
		nextValidators = append(nextValidators, &Validator{
			Address:       tHandler.strategy.ValidatorSet.NextHeightCandidateValidators[i].Address,
			PubKey:        tHandler.strategy.ValidatorSet.NextHeightCandidateValidators[i].PubKey,
			Power:         tHandler.strategy.ValidatorSet.NextHeightCandidateValidators[i].Power,
			AddressString: hex.EncodeToString(tmPubKey.Address()),
		})
	}

	PosReceipt := &PTableAll{
		PosItemMap:              tHandler.strategy.PosTable.PosItemMap,
		AccountMapList:          tHandler.strategy.AccountMapList,
		NextCandidateValidators: nextValidators,
	}
	jsonStr, err := json.Marshal(PosReceipt)
	if err != nil {
		w.Write([]byte("error occured when marshal into json"))
	} else {
		w.Write(jsonStr)
	}
}

// This function will return the used data structure
func (tHandler *THandler) GetPosTableData(w http.ResponseWriter, req *http.Request) {
	PosTable := &PosItemMapData{
		PosItemMap: tHandler.strategy.PosTable.PosItemMap,
	}
	jsonStr, err := json.Marshal(PosTable)
	if err != nil {
		w.Write([]byte("error occured when marshal into json"))
	} else {
		w.Write(jsonStr)
	}
}

// This function will return the used data structure
func (tHandler *THandler) GetAccountMapData(w http.ResponseWriter, req *http.Request) {
	AccountMap := &AccountMapData{
		MapList: tHandler.strategy.AccountMapList.MapList,
	}
	jsonStr, err := json.Marshal(AccountMap)
	if err != nil {
		w.Write([]byte("error occured when marshal into json"))
	} else {
		w.Write(jsonStr)
	}
}

// This function will return the used data structure
func (tHandler *THandler) GetPreBlockValidators(w http.ResponseWriter, req *http.Request) {
	var preValidators []*Validator
	for i := 0; i < len(tHandler.strategy.ValidatorSet.CurrentValidators); i++ {
		tmAddress := tHandler.strategy.ValidatorSet.CurrentValidators[i].Address
		tmAddressStr := strings.ToLower(hex.EncodeToString(tmAddress))
		preValidators = append(preValidators, &Validator{
			Address:       tmAddress,
			AddressString: tmAddressStr,
			Power:         tHandler.strategy.ValidatorSet.CurrentValidatorWeight[i],
			Signer:        tHandler.strategy.AccountMapList.MapList[tmAddressStr].Signer,
			Beneficiary:   tHandler.strategy.AccountMapList.MapList[tmAddressStr].Beneficiary,
		})
	}

	jsonStr, err := json.Marshal(preValidators)
	if err != nil {
		w.Write([]byte("error occured when marshal into json"))
	} else {
		w.Write(jsonStr)
	}
}

// This function will return the used data structure
func (tHandler *THandler) GetPreBlockProposer(w http.ResponseWriter, req *http.Request) {
	proposer := tHandler.strategy.ProposerAddress
	PreBlockProposer := &PreBlockProposer{
		PreBlockProposer: proposer,
		Beneficiary:      tHandler.strategy.AccountMapList.MapList[proposer].Beneficiary,
		Signer:           tHandler.strategy.AccountMapList.MapList[proposer].Signer,
	}
	jsonStr, err := json.Marshal(PreBlockProposer)
	if err != nil {
		w.Write([]byte("error occured when marshal into json"))
	} else {
		w.Write(jsonStr)
	}
}

func (tHandler *THandler) GetAllCandidateValidatorPool(w http.ResponseWriter, req *http.Request) {
	var preValidators []*Validator
	for i := 0; i < len(tHandler.strategy.ValidatorSet.NextHeightCandidateValidators); i++ {
		tmAddress := tHandler.strategy.ValidatorSet.NextHeightCandidateValidators[i].Address
		tmAddressStr := strings.ToLower(hex.EncodeToString(tmAddress))
		preValidators = append(preValidators, &Validator{
			Address:       tmAddress,
			Power:         int64(1),
			AddressString: tmAddressStr,
			SignerBalance: tHandler.strategy.AccountMapList.MapList[tmAddressStr].SignerBalance,
			Signer:        tHandler.strategy.AccountMapList.MapList[tmAddressStr].Signer,
			Beneficiary:   tHandler.strategy.AccountMapList.MapList[tmAddressStr].Beneficiary,
		})
	}

	jsonStr, err := json.Marshal(preValidators)
	if err != nil {
		w.Write([]byte("error occured when marshal into json"))
	} else {
		w.Write(jsonStr)
	}
}
