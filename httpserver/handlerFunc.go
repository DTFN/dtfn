package httpserver

import (
	"encoding/json"
	"github.com/green-element-chain/gelchain/ethereum"
	emtTypes "github.com/green-element-chain/gelchain/types"
	tmTypes "github.com/tendermint/tendermint/types"
	"io"
	"math/big"
	"net/http"
	"strconv"
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
	//tHandler.HandlersMap["/isUpsert"] = tHandler.IsUpsert
	//tHandler.HandlersMap["/isRemove"] = tHandler.IsRemove
	tHandler.HandlersMap["/GetPosTable"] = tHandler.GetPosTableData
	tHandler.HandlersMap["/GetAccountMap"] = tHandler.GetAccountMapData
	tHandler.HandlersMap["/GetCurrentValidators"] = tHandler.GetPreBlockUpdateValidators
	tHandler.HandlersMap["/GetPreBlockProposer"] = tHandler.GetPreBlockProposer
	tHandler.HandlersMap["/GetAllCandidateValidators"] = tHandler.GetAllCandidateValidatorPool
	tHandler.HandlersMap["/GetEncourage"] = tHandler.GetEncourage

	tHandler.HandlersMap["/GetPosTable"] = tHandler.GetPosTableData
	tHandler.HandlersMap["/GetAccountMap"] = tHandler.GetAccountMapData
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
/*func (tHandler *THandler) IsUpsert(w http.ResponseWriter, req *http.Request) {
	var nextValidators []*Validator
	for _,posItem:=range tHandler.strategy.NextEpochValData.PosTable.PosItemMap {
		nextValidators = append(nextValidators, &Validator{
			//Address:       tHandler.strategy.CurrentHeightValData.CurrCandidateValidators[i].Address,
			PubKey:        posItem.PubKey,
			Power:         posItem.Slots,
			AddressString: posItem.TmAddress,
		})
	}

	PosReceipt := &PTableAll{
		PosItemMap:              tHandler.strategy.CurrEpochValData.PosTable.PosItemMap,
		AccountMapList:          tHandler.strategy.CurrEpochValData.PosTable.TmAddressToSignerMap,
		NextCandidateValidators: nextValidators,
	}
	jsonStr, err := json.Marshal(PosReceipt)
	if err != nil {
		w.Write([]byte("error occured when marshal into json"))
	} else {
		w.Write(jsonStr)
	}
}*/

// This function will return the used data structure
func (tHandler *THandler) GetPosTableData(w http.ResponseWriter, req *http.Request) {
	tHandler.strategy.CurrEpochValData.PosTable.Mtx.RLock()
	defer tHandler.strategy.CurrEpochValData.PosTable.Mtx.RUnlock()
	PosTable := &PosItemMapData{
		PosItemMap: tHandler.strategy.CurrEpochValData.PosTable.PosItemMap,
		Threshold:  tHandler.strategy.CurrEpochValData.PosTable.Threshold,
		TotalSlots: tHandler.strategy.CurrEpochValData.PosTable.TotalSlots,
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
	tHandler.strategy.NextEpochValData.PosTable.Mtx.RLock()
	defer tHandler.strategy.NextEpochValData.PosTable.Mtx.RUnlock()
	PosTable := &PosItemMapData{
		PosItemMap: tHandler.strategy.NextEpochValData.PosTable.PosItemMap,
		Threshold:  tHandler.strategy.NextEpochValData.PosTable.Threshold,
		TotalSlots: tHandler.strategy.NextEpochValData.PosTable.TotalSlots,
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
	tHandler.strategy.CurrEpochValData.PosTable.Mtx.RLock()
	defer tHandler.strategy.CurrEpochValData.PosTable.Mtx.RUnlock()
	AccountMap := &AccountMapData{
		MapList: tHandler.strategy.CurrEpochValData.PosTable.TmAddressToSignerMap,
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
	tHandler.strategy.NextEpochValData.PosTable.Mtx.RLock()
	defer tHandler.strategy.NextEpochValData.PosTable.Mtx.RUnlock()
	AccountMap := &AccountMapData{
		MapList: tHandler.strategy.NextEpochValData.PosTable.TmAddressToSignerMap,
	}
	jsonStr, err := json.Marshal(AccountMap)
	if err != nil {
		w.Write([]byte("error occured when marshal into json"))
	} else {
		w.Write(jsonStr)
	}
}

// This function will return the used data structure
func (tHandler *THandler) GetPreBlockUpdateValidators(w http.ResponseWriter, req *http.Request) {
	var preValidators []*Validator
	tHandler.strategy.CurrEpochValData.PosTable.Mtx.RLock()
	defer tHandler.strategy.CurrEpochValData.PosTable.Mtx.RUnlock()
	for tmAddressStr, v := range tHandler.strategy.CurrentHeightValData.Validators {
		//pubKey := tHandler.strategy.CurrentHeightValData.UpdateValidators[i].PubKey
		//tmPubKey, _ := tmTypes.PB2TM.PubKey(pubKey)
		preValidators = append(preValidators, &Validator{
			//Address:       tmAddress,
			AddressString: tmAddressStr,
			PubKey:        v.PubKey,
			Power:         v.Power,
			Signer:        v.Signer,
			Beneficiary:   tHandler.strategy.CurrEpochValData.PosTable.PosItemMap[v.Signer].Beneficiary,
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
	proposer := tHandler.strategy.CurrentHeightValData.ProposerAddress
	tHandler.strategy.CurrEpochValData.PosTable.Mtx.RLock()
	defer tHandler.strategy.CurrEpochValData.PosTable.Mtx.RUnlock()
	signer := tHandler.strategy.CurrEpochValData.PosTable.TmAddressToSignerMap[proposer]
	PreBlockProposer := &PreBlockProposer{
		PreBlockProposer: proposer,
		Beneficiary:      tHandler.strategy.CurrEpochValData.PosTable.PosItemMap[signer].Beneficiary,
		Signer:           signer,
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
	tHandler.strategy.CurrEpochValData.PosTable.Mtx.RLock()
	defer tHandler.strategy.CurrEpochValData.PosTable.Mtx.RUnlock()
	for signer, posItem := range tHandler.strategy.CurrEpochValData.PosTable.PosItemMap {
		pubKey := posItem.PubKey
		//tmPubKey, _ := tmTypes.PB2TM.PubKey(pubKey)
		tmAddressStr := posItem.TmAddress
		balance := big.NewInt(1)
		balance.Mul(tHandler.strategy.CurrEpochValData.PosTable.Threshold, big.NewInt(posItem.Slots))
		preValidators = append(preValidators, &Validator{
			//Address:       tmAddress,
			Power:         int64(1),
			AddressString: tmAddressStr,
			PubKey:        pubKey,
			SignerBalance: balance,
			Signer:        signer,
			Beneficiary:   posItem.Beneficiary,
			BlsKeyString:  posItem.BlsKeyString,
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
	topKSigners := tHandler.strategy.NextEpochValData.PosTable.TopKSigners(100)
	tHandler.strategy.NextEpochValData.PosTable.Mtx.RLock()
	defer tHandler.strategy.NextEpochValData.PosTable.Mtx.RUnlock()
	for _, signer := range topKSigners {
		posItem := tHandler.strategy.NextEpochValData.PosTable.PosItemMap[signer]
		balance := big.NewInt(1)
		balance.Mul(tHandler.strategy.CurrEpochValData.PosTable.Threshold, big.NewInt(posItem.Slots))
		preValidators = append(preValidators, &Validator{
			//Address:       tmAddress,
			Power:         int64(1),
			AddressString: posItem.TmAddress,
			PubKey:        posItem.PubKey,
			SignerBalance: balance,
			Signer:        signer,
			Beneficiary:   posItem.Beneficiary,
			BlsKeyString:  posItem.BlsKeyString,
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
	minerBonus := big.NewInt(0)
	divisor := big.NewInt(1)
	minerBonus.Div(tHandler.strategy.CurrEpochValData.TotalBalance, divisor.Mul(big.NewInt(100), big.NewInt(365*24*60*60/5)))

	encourage := &Encourage{
		TotalBalance:          tHandler.strategy.CurrEpochValData.TotalBalance,
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
