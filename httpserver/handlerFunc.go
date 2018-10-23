package httpserver

import (
	"encoding/hex"
	"encoding/json"
	emtTypes "github.com/tendermint/ethermint/types"
	tmTypes "github.com/tendermint/tendermint/types"
	"io"
	"net/http"
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
	tHandler.HandlersMap["/GetPosTable"] = tHandler.GetPosTable
	tHandler.HandlersMap["/GetAccountMap"] = tHandler.GetAccountMap
	tHandler.HandlersMap["/GetCurrentValidators"] = tHandler.GetCurrentValidators
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

	PosReceipt := &Pos{
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

	PosReceipt := &Pos{
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
func (tHandler *THandler) GetPosTable(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("this is pos table structure"))
}

// This function will return the used data structure
func (tHandler *THandler) GetAccountMap(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("this is account map structure"))
}

// This function will return the used data structure
func (tHandler *THandler) GetCurrentValidators(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("this is current validators structure"))
}
