package httpserver

import (
	"encoding/hex"
	"encoding/json"
	"github.com/green-element-chain/gelchain/ethereum"
	emtTypes "github.com/green-element-chain/gelchain/types"
	tmTypes "github.com/tendermint/tendermint/types"
	"io"
	"math/big"
	"net/http"
	"strconv"
	"strings"
)

type THandler struct {
	HandlersMap map[string]HandlersFunc
	strategy    *emtTypes.Strategy
	backend     *ethereum.Backend
}

type HandlersFunc func(http.ResponseWriter, *http.Request)

func NewTHandler(strategy *emtTypes.Strategy, backend *ethereum.Backend) *THandler {
	return &THandler{
		HandlersMap: make(map[string]HandlersFunc),
		strategy:    strategy,
		backend:     backend,
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
	tHandler.HandlersMap["/GetEncourage"] = tHandler.GetEncourage

	tHandler.HandlersMap["/GetNextPosTable"] = tHandler.GetNextPosTableData
	tHandler.HandlersMap["/GetNextAccountMap"] = tHandler.GetNextAccountMapData
	tHandler.HandlersMap["/GetNextAllCandidateValidators"] = tHandler.GetNextAllCandidateValidatorPool
	tHandler.HandlersMap["/GetInitialValidator"] = tHandler.GetInitialValidator
	tHandler.HandlersMap["/GetHeadEventSize"] = tHandler.GetTxPoolEventSize
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
	for i := 0; i < len(tHandler.strategy.CurrRoundValData.CurrCandidateValidators); i++ {
		tmPubKey, _ := tmTypes.PB2TM.PubKey(tHandler.strategy.CurrRoundValData.CurrCandidateValidators[i].PubKey)
		nextValidators = append(nextValidators, &Validator{
			//Address:       tHandler.strategy.CurrRoundValData.CurrCandidateValidators[i].Address,
			PubKey:        tHandler.strategy.CurrRoundValData.CurrCandidateValidators[i].PubKey,
			Power:         tHandler.strategy.CurrRoundValData.CurrCandidateValidators[i].Power,
			AddressString: hex.EncodeToString(tmPubKey.Address()),
		})
	}

	PosReceipt := &PTableAll{
		PosItemMap:              tHandler.strategy.CurrRoundValData.PosTable.PosItemMap,
		AccountMapList:          tHandler.strategy.CurrRoundValData.AccountMapList,
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
	for i := 0; i < len(tHandler.strategy.CurrRoundValData.CurrCandidateValidators); i++ {
		tmPubKey, _ := tmTypes.PB2TM.PubKey(tHandler.strategy.CurrRoundValData.CurrCandidateValidators[i].PubKey)
		nextValidators = append(nextValidators, &Validator{
			//Address:       tHandler.strategy.CurrRoundValData.CurrCandidateValidators[i].Address,
			PubKey:        tHandler.strategy.CurrRoundValData.CurrCandidateValidators[i].PubKey,
			Power:         tHandler.strategy.CurrRoundValData.CurrCandidateValidators[i].Power,
			AddressString: hex.EncodeToString(tmPubKey.Address()),
		})
	}

	PosReceipt := &PTableAll{
		PosItemMap:              tHandler.strategy.CurrRoundValData.PosTable.PosItemMap,
		AccountMapList:          tHandler.strategy.CurrRoundValData.AccountMapList,
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
		PosItemMap:   tHandler.strategy.CurrRoundValData.PosTable.PosItemMap,
		Threshold:    tHandler.strategy.CurrRoundValData.PosTable.Threshold,
		PosArraySize: tHandler.strategy.CurrRoundValData.PosTable.PosArraySize,
	}
	jsonStr, err := json.Marshal(PosTable)
	if err != nil {
		w.Write([]byte("error occured when marshal into json"))
	} else {
		w.Write(jsonStr)
	}
}

// This function will return the used data structure
func (tHandler *THandler) GetNextPosTableData(w http.ResponseWriter, req *http.Request) {
	PosTable := &PosItemMapData{
		PosItemMap:   tHandler.strategy.NextRoundValData.NextRoundPosTable.PosItemMap,
		Threshold:    tHandler.strategy.NextRoundValData.NextRoundPosTable.Threshold,
		PosArraySize: tHandler.strategy.NextRoundValData.NextRoundPosTable.PosArraySize,
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
		MapList: tHandler.strategy.CurrRoundValData.AccountMapList.MapList,
	}
	jsonStr, err := json.Marshal(AccountMap)
	if err != nil {
		w.Write([]byte("error occured when marshal into json"))
	} else {
		w.Write(jsonStr)
	}
}

// This function will return the used data structure
func (tHandler *THandler) GetNextAccountMapData(w http.ResponseWriter, req *http.Request) {
	AccountMap := &AccountMapData{
		MapList: tHandler.strategy.NextRoundValData.NextAccountMapList.MapList,
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
	for i := 0; i < len(tHandler.strategy.CurrRoundValData.CurrentValidators); i++ {
		pubKey := tHandler.strategy.CurrRoundValData.CurrentValidators[i].PubKey
		tmPubKey, _ := tmTypes.PB2TM.PubKey(pubKey)
		tmAddressStr := strings.ToLower(tmPubKey.Address().String())
		preValidators = append(preValidators, &Validator{
			//Address:       tmAddress,
			AddressString: tmAddressStr,
			PubKey:        tHandler.strategy.CurrRoundValData.CurrentValidators[i].PubKey,
			Power:         tHandler.strategy.CurrRoundValData.CurrentValidatorWeight[i],
			Signer:        tHandler.strategy.CurrRoundValData.AccountMapList.MapList[tmAddressStr].Signer,
			Beneficiary:   tHandler.strategy.CurrRoundValData.AccountMapList.MapList[tmAddressStr].Beneficiary,
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
	proposer := tHandler.strategy.CurrRoundValData.ProposerAddress
	PreBlockProposer := &PreBlockProposer{
		PreBlockProposer: proposer,
		Beneficiary:      tHandler.strategy.CurrRoundValData.AccountMapList.MapList[proposer].Beneficiary,
		Signer:           tHandler.strategy.CurrRoundValData.AccountMapList.MapList[proposer].Signer,
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
	for i := 0; i < len(tHandler.strategy.CurrRoundValData.CurrCandidateValidators); i++ {
		pubKey := tHandler.strategy.CurrRoundValData.CurrCandidateValidators[i].PubKey
		tmPubKey, _ := tmTypes.PB2TM.PubKey(pubKey)
		tmAddressStr := strings.ToLower(tmPubKey.Address().String())
		preValidators = append(preValidators, &Validator{
			//Address:       tmAddress,
			Power:         int64(1),
			AddressString: tmAddressStr,
			PubKey:        tHandler.strategy.CurrRoundValData.CurrCandidateValidators[i].PubKey,
			SignerBalance: tHandler.strategy.CurrRoundValData.AccountMapList.MapList[tmAddressStr].SignerBalance,
			Signer:        tHandler.strategy.CurrRoundValData.AccountMapList.MapList[tmAddressStr].Signer,
			Beneficiary:   tHandler.strategy.CurrRoundValData.AccountMapList.MapList[tmAddressStr].Beneficiary,
			BlsKeyString:  tHandler.strategy.CurrRoundValData.AccountMapList.MapList[tmAddressStr].BlsKeyString,
		})
	}

	jsonStr, err := json.Marshal(preValidators)
	if err != nil {
		w.Write([]byte("error occured when marshal into json"))
	} else {
		w.Write(jsonStr)
	}
}

func (tHandler *THandler) GetNextAllCandidateValidatorPool(w http.ResponseWriter, req *http.Request) {
	var preValidators []*Validator
	for i := 0; i < len(tHandler.strategy.NextRoundValData.NextRoundCandidateValidators); i++ {
		pubKey := tHandler.strategy.NextRoundValData.NextRoundCandidateValidators[i].PubKey
		tmPubKey, _ := tmTypes.PB2TM.PubKey(pubKey)
		tmAddressStr := strings.ToLower(tmPubKey.Address().String())
		preValidators = append(preValidators, &Validator{
			//Address:       tmAddress,
			Power:         int64(1),
			AddressString: tmAddressStr,
			PubKey:        tHandler.strategy.NextRoundValData.NextRoundCandidateValidators[i].PubKey,
			SignerBalance: tHandler.strategy.NextRoundValData.NextAccountMapList.MapList[tmAddressStr].SignerBalance,
			Signer:        tHandler.strategy.NextRoundValData.NextAccountMapList.MapList[tmAddressStr].Signer,
			Beneficiary:   tHandler.strategy.NextRoundValData.NextAccountMapList.MapList[tmAddressStr].Beneficiary,
			BlsKeyString:  tHandler.strategy.CurrRoundValData.AccountMapList.MapList[tmAddressStr].BlsKeyString,
		})
	}

	jsonStr, err := json.Marshal(preValidators)
	if err != nil {
		w.Write([]byte("error occured when marshal into json"))
	} else {
		w.Write(jsonStr)
	}
}

func (tHandler *THandler) GetInitialValidator(w http.ResponseWriter, req *http.Request) {
	var preValidators []*Validator
	for i := 0; i < len(tHandler.strategy.NextRoundValData.NextRoundCandidateValidators); i++ {
		pubKey := tHandler.strategy.CurrRoundValData.InitialValidators[i].PubKey
		tmPubKey, _ := tmTypes.PB2TM.PubKey(pubKey)
		tmAddressStr := strings.ToLower(tmPubKey.Address().String())
		preValidators = append(preValidators, &Validator{
			//Address:       tmAddress,
			AddressString: tmAddressStr,
			PubKey:        tHandler.strategy.CurrRoundValData.InitialValidators[i].PubKey,
		})
	}

	jsonStr, err := json.Marshal(preValidators)
	if err != nil {
		w.Write([]byte("error occured when marshal into json"))
	} else {
		w.Write(jsonStr)
	}
}

func (tHandler *THandler) GetEncourage(w http.ResponseWriter, req *http.Request) {
	minerBonus := big.NewInt(1)
	divisor := big.NewInt(1)
	minerBonus.Div(tHandler.strategy.CurrRoundValData.TotalBalance, divisor.Mul(big.NewInt(100), big.NewInt(365*24*60*60/5)))

	encourage := &Encourage{
		TotalBalance:          tHandler.strategy.CurrRoundValData.TotalBalance,
		EncourageAverageBlock: minerBonus,
	}

	jsonStr, err := json.Marshal(encourage)
	if err != nil {
		w.Write([]byte("error occured when marshal into json"))
	} else {
		w.Write(jsonStr)
	}
}

func (tHandler *THandler) GetTxPoolEventSize(w http.ResponseWriter, req *http.Request) {
	jsonStr, err := json.Marshal("unread txpool event size: "+strconv.Itoa(tHandler.
		backend.Ethereum().TxPool().GetTxpoolChainHeadSize()))
	if err != nil {
		w.Write([]byte("error occured when marshal into json"))
	} else {
		w.Write(jsonStr)
	}
}