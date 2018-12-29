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
	"sort"
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
	for i := 0; i < len(tHandler.strategy.CurrHeightValData.CurrCandidateValidators); i++ {
		tmPubKey, _ := tmTypes.PB2TM.PubKey(tHandler.strategy.CurrHeightValData.CurrCandidateValidators[i].PubKey)
		nextValidators = append(nextValidators, &Validator{
			//Address:       tHandler.strategy.CurrHeightValData.CurrCandidateValidators[i].Address,
			PubKey:        tHandler.strategy.CurrHeightValData.CurrCandidateValidators[i].PubKey,
			Power:         tHandler.strategy.CurrHeightValData.CurrCandidateValidators[i].Power,
			AddressString: hex.EncodeToString(tmPubKey.Address()),
		})
	}

	PosReceipt := &PTableAll{
		PosItemMap:              tHandler.strategy.CurrHeightValData.PosTable.PosItemMap,
		AccountMapList:          tHandler.strategy.CurrHeightValData.AccountMap,
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
	for i := 0; i < len(tHandler.strategy.CurrHeightValData.CurrCandidateValidators); i++ {
		tmPubKey, _ := tmTypes.PB2TM.PubKey(tHandler.strategy.CurrHeightValData.CurrCandidateValidators[i].PubKey)
		nextValidators = append(nextValidators, &Validator{
			//Address:       tHandler.strategy.CurrHeightValData.CurrCandidateValidators[i].Address,
			PubKey:        tHandler.strategy.CurrHeightValData.CurrCandidateValidators[i].PubKey,
			Power:         tHandler.strategy.CurrHeightValData.CurrCandidateValidators[i].Power,
			AddressString: hex.EncodeToString(tmPubKey.Address()),
		})
	}

	PosReceipt := &PTableAll{
		PosItemMap:              tHandler.strategy.CurrHeightValData.PosTable.PosItemMap,
		AccountMapList:          tHandler.strategy.CurrHeightValData.AccountMap,
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
		PosItemMap:   tHandler.strategy.CurrHeightValData.PosTable.PosItemMap,
		Threshold:    tHandler.strategy.CurrHeightValData.PosTable.Threshold,
		PosArraySize: tHandler.strategy.CurrHeightValData.PosTable.PosArraySize,
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
		PosItemMap:   tHandler.strategy.NextEpochValData.NextPosTable.PosItemMap,
		Threshold:    tHandler.strategy.NextEpochValData.NextPosTable.Threshold,
		PosArraySize: tHandler.strategy.NextEpochValData.NextPosTable.PosArraySize,
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
		MapList: tHandler.strategy.CurrHeightValData.AccountMap.MapList,
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
		MapList: tHandler.strategy.NextEpochValData.NextAccountMap.MapList,
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
	for tmAddressStr, v := range tHandler.strategy.CurrHeightValData.CurrentValidators {
		//pubKey := tHandler.strategy.CurrHeightValData.UpdateValidators[i].PubKey
		//tmPubKey, _ := tmTypes.PB2TM.PubKey(pubKey)
		preValidators = append(preValidators, &Validator{
			//Address:       tmAddress,
			AddressString: tmAddressStr,
			PubKey:        v.PubKey,
			Power:         v.Power,
			Signer:        v.Signer,
			Beneficiary:   tHandler.strategy.CurrHeightValData.AccountMap.MapList[tmAddressStr].Beneficiary,
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
	proposer := tHandler.strategy.CurrHeightValData.ProposerAddress
	PreBlockProposer := &PreBlockProposer{
		PreBlockProposer: proposer,
		Beneficiary:      tHandler.strategy.CurrHeightValData.AccountMap.MapList[proposer].Beneficiary,
		Signer:           tHandler.strategy.CurrHeightValData.AccountMap.MapList[proposer].Signer,
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
	for i := 0; i < len(tHandler.strategy.CurrHeightValData.CurrCandidateValidators); i++ {
		pubKey := tHandler.strategy.CurrHeightValData.CurrCandidateValidators[i].PubKey
		tmPubKey, _ := tmTypes.PB2TM.PubKey(pubKey)
		tmAddressStr := tmPubKey.Address().String()
		signer := tHandler.strategy.CurrHeightValData.AccountMap.MapList[tmAddressStr].Signer
		balance := tHandler.strategy.CurrHeightValData.PosTable.PosItemMap[signer].Balance
		preValidators = append(preValidators, &Validator{
			//Address:       tmAddress,
			Power:         int64(1),
			AddressString: tmAddressStr,
			PubKey:        tHandler.strategy.CurrHeightValData.CurrCandidateValidators[i].PubKey,
			SignerBalance: balance,
			Signer:        signer,
			Beneficiary:   tHandler.strategy.CurrHeightValData.AccountMap.MapList[tmAddressStr].Beneficiary,
			BlsKeyString:  tHandler.strategy.CurrHeightValData.AccountMap.MapList[tmAddressStr].BlsKeyString,
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
	var keys []string
	for k := range tHandler.strategy.NextEpochValData.NextCandidateValidators {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, tmAddressStr := range keys {
		signer := tHandler.strategy.NextEpochValData.NextAccountMap.MapList[tmAddressStr].Signer
		balance := tHandler.strategy.NextEpochValData.NextPosTable.PosItemMap[signer].Balance
		preValidators = append(preValidators, &Validator{
			//Address:       tmAddress,
			Power:         int64(1),
			AddressString: tmAddressStr,
			PubKey:        tHandler.strategy.NextEpochValData.NextCandidateValidators[tmAddressStr].PubKey,
			SignerBalance: balance,
			Signer:        signer,
			Beneficiary:   tHandler.strategy.NextEpochValData.NextAccountMap.MapList[tmAddressStr].Beneficiary,
			BlsKeyString:  tHandler.strategy.CurrHeightValData.AccountMap.MapList[tmAddressStr].BlsKeyString,
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
	for i := 0; i < len(tHandler.strategy.InitialValidators); i++ {
		pubKey := tHandler.strategy.InitialValidators[i].PubKey
		tmPubKey, _ := tmTypes.PB2TM.PubKey(pubKey)
		tmAddressStr := tmPubKey.Address().String()
		preValidators = append(preValidators, &Validator{
			//Address:       tmAddress,
			AddressString: tmAddressStr,
			PubKey:        tHandler.strategy.InitialValidators[i].PubKey,
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
	minerBonus.Div(tHandler.strategy.CurrHeightValData.TotalBalance, divisor.Mul(big.NewInt(100), big.NewInt(365*24*60*60/5)))

	encourage := &Encourage{
		TotalBalance:          tHandler.strategy.CurrHeightValData.TotalBalance,
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
	jsonStr, err := json.Marshal("unread txpool event size: " + strconv.Itoa(tHandler.
		backend.Ethereum().TxPool().GetTxpoolChainHeadSize()))
	if err != nil {
		w.Write([]byte("error occured when marshal into json"))
	} else {
		w.Write(jsonStr)
	}
}
