1.pos_table
data structure

function:
2.upsertPosItem

3.removePosItem



app data_structure
1.pos_table
2.accountmap
3.validatorSet


addValidatorTx
removeValidatorTx
GetUpdateValidators

app data_structure
============

1.pos_table:
----------
 the structure of the Postable is :
    type PosTable struct {
        mtx          sync.RWMutex
        PosItemMap   map[common.Address]*posItem
        PosArray     []*posItem                  // All posItem
        PosArraySize int                         // real size of posArray
        threshold    int64                       // threshold value of PosTable
        PrimeArray   *PrimeArray
    }

 its posItem's structure :
    type posItem struct {
    	Signer      common.Address
    	Balance     int64
    	PubKey      abciTypes.PubKey
    	Indexes     map[int]bool
    	Beneficiary common.Address
    }
 * Signer & Beneficiary are all go-ethereum-address. Signer is the TX sender, and Beneficiary is the ETH encourage & gas receiver.
 * PubKey is the tendermint abci PubKey of the tendermint node.
 * Balance is used to decide how many VotingPower the Signer has. VotingPower = int(Balance/threshold)

2.AccountMap
----------
AccountMap is an initial accountMap between tendermint address and go-ethereum-address.
    type AccountMap struct {
        Signer      common.Address
        Beneficiary common.Address
    }

AccountMapList defines the initial list of AccountMap.
    type AccountMapList struct {
        MapList map[string]*AccountMap

        FilePath string
        mtx      sync.Mutex
    }
 * MapList is a map structure, the key 'string' is the tendermint address

3.ValidatorSet
----------
the ValidatorSet structure is :
    type Validators struct {
        CommitteeValidators []*abciTypes.Validator

        CandidateValidators []*abciTypes.Validator

        NextCandidateValidators []*abciTypes.Validator

        CurrentValidators []*abciTypes.Validator


    }
 * CommitteeValidators are the validators of committee , used to support +2/3 ,our node
 * CandidateValidators are the current validators of candidate
 * Next candidate Validators , will changed every 200 height, and be changed by addValidatorTx and removeValidatorTx
 * CurrentValidators are validators of currentBlock, will use to set votePower to 0 ,then remove from tendermint validatorSetwill be select by postable.
   CurrentValidators is the true validators except commmittee validator when height != 1 if height =1 ,CurrentValidator = nil
 * note1 : if we get a addValidatorsTx at height 101, we will put it into the NextCandidateValidators and move into postable NextCandidateValidator will used in the next height200,
   postable will used in the next height 102
 * note2 : if we get a removeValidatorsTx at height 101, we will remove it from the NextCandidateValidators and remove from postable, NextCandidateValidator will used in the next height200
   postable will used in the next height 102

the *abciTypes.Validator is :
type Validator struct {
	Address              []byte   `protobuf:"bytes,1,opt,name=address,proto3" json:"address,omitempty"`
	PubKey               PubKey   `protobuf:"bytes,2,opt,name=pub_key,json=pubKey" json:"pub_key"`
	Power                int64    `protobuf:"varint,3,opt,name=power,proto3" json:"power,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

Function:
============

1.UpsertPosItem
----------
    UpsertPosItem(signer common.Address, balance int64, beneficiary common.Address,pubkey abciTypes.PubKey) (bool, error)
Goals
^^^^^
This function excutes the Upsert of postable, when we call UpsertValidatorTx.


2.removePosItem
----------
    RemovePosItem(account common.Address) (bool, error)
Goals
^^^^^
This function excutes the removation of the 'account' candidate in postable, when we call removeValidatorTx.


3.UpsertValidatorTx
----------
    UpsertValidatorTx(signer common.Address, balance int64,beneficiary common.Address, pubkey crypto.PubKey) (bool, error)
Goals
^^^^^
This function excutes the change of maplist, postable and NextCandidateValidator.
It initials the maplist and NextCandidateValidator the first time we call it, meanwhile, the parameters signer and pubkey(include: abciPubKey, tmAddress) have relation with each oter.
Generally, one signer could map into one or more pubkey, but pubkey only maps into one signer.
NextCandidateValidator is a set of tendermint Validators(marked by the tmAddress).
in sum, the we can take for granted that NextCandidateValidator tendermint node, and the maplist is the set of ethereum signe & Beneficiary related with the tendermint node address.


4.removeValidatorTx
----------
    RemoveValidatorTx(signer common.Address, balance int64,beneficiary common.Address, pubkey crypto.PubKey) (bool, error)
Goals
^^^^^
This function excutes the removation of maplist, postable and NextCandidateValidator.


5.GetUpdateValidators
----------
    GetUpdatedValidators(height int64) abciTypes.ResponseEndBlock
Goals
^^^^^
GetUpdatedValidators returns an updated validator set from the strategy.
When the block heigth is not integral multiple of 200, this function will call the utils.enterSelectValidators() to fetch the validator set.
When the block heigth is integral multiple of 200, this function will call the other function to change the tendermint validator set. //the function is TODO