#Tendermint
Tendermint Core is Byzantine Fault Tolerant (BFT) middleware that takes
a state transition machine - written in any programming language and
securely replicates it on many machines.

##BLS In Tendermint
We use BLS in Tendermint to generate random number among some peers. each
peer can broadcast its sign slice to other peers, when the peer receive the
number of sign slice greater than threshold, it can recover group sign, and
use the group sign to generate random number by hash function. All peers
work together to generate a random number at each height of block, each peer
generate the random number is equal and this can proved by mathematics, the
random number of each height is completely different, and nobody knows what
the next random number is. We used the random number to select the sequence
who make next block. This means that we reach a consensus among different
people about random number in every height.

#Implement BLS
BLS algorithm is divided into two processes, the first is initialization, and
the second is generating random number. We should have a successful completion
of the initialization, than use the result of initialization to join generating
random number.

##BLS initialization
We use BLSInit struct to complete initialization, as follow:
```
type BLSInit struct {
	mtx sync.RWMutex

	thresholdNum              int
	groupSize                 int
	nodeID                    string
	epoch                     int64
	nodeIndex                 int
	groupNodeID               []string
	priva0                    tbls.SecretKey
	coeffPoly                 []string
	ourCommit                 *types.CommitmentType
	receiveCommitment         map[string]*types.CommitmentType
	skShareCt                 map[string]*types.SKShareCtAndSign
	receiveSkShare            *SKShareInfo
	approvedSkShare           *types.ApprovedSKShareInfo
	storedApprovalSKShareInfo *StoredApprovalSKShareInfo
	receiveInitFinishMsg      map[string]string
	pkAggVec                  map[string]string
	skAgg                     string
	groupPK                   string
	privkPKE                  string
	pubkPKEVec                map[string]string
	peerBLSInitMsgQueue       chan msgInfo
	internalBLSInitMsgQueue   chan msgInfo
	commitInHeight            int64
}
```
- `mtx`: the lock
- `thresholdNum`: the threshold of the BLS init group
- `groupSize`: the size of the BLS init group
- `nodeID`: ID of current node
- `epoch`: current epoch of initialization
- `nodeIndex`: index of this node in the BLS init group
- `groupNodeID`: node ID in the BLS init group
- `priva0`: the constant term of polynomial
- `coeffPoly`: the coefficient of polynomial
- `ourCommit`: commit of current node
- `receiveCommitment`: receive commitment from other node
- `skShareCt`: secret, broadcast to other node
- `receiveSkShare`: receive the secret from other node
- `approvedSkShare`: pass validation of secret from other node
- `storedApprovalSKShareInfo`: validation result of secret in block
- `receiveInitFinishMsg`: receive init finish msg
- `pkAggVec`: public key of all node, generate when init finish
- `skAgg`: private key of current node, generate when init finish
- `groupPK`: group public key of the BLS init group
- `privkPKE`: private key of current node from secp256k1
- `pubkPKEVec`: public key of all node from secp256k1
- `peerBLSInitMsgQueue`: msg queue to solve external message
- `internalBLSInitMsgQueue`: msg queue to solve internal message
- `commitInHeight`: the height in block of commit

In the beginning of BLS initialization, we should know all node ID in the BLS
init group, and sort the init group, find the location of the current node as
`nodeIndex`, then confirm the threshold of init group, we use half of group
number as `thresholdNum`. when we have `thresholdNum`, it is time for generating
multinomial coefficient, using`SecretKey.SetByCSPRNG()`to generate the constant
term of polynomial as `priva0`, according to `priva0`, generate coefficient.
When multinomial is ready, we should calculate our commits according to coefficient
of multinomial, and use p2p network to broadcast to other peer. Next step is that
broadcasts secrets, first, we should generate secret according to multinomial for
each peer, and then send the secret for corresponding peer. Other peer receive the
secret, they can use the commit to verify the secret and put the result of
verification in block, if they verify success, store the secret. After that, all
node make a statistics for result of verification in block, remove the node that
failed to verify.
When the above process ends, we use the secret that all the remaining nodes send
to me to generate private key for BLS as `skAgg`, generate public key for all the
remaining nodes as `pubkPKEVec`, and generate group public key as `groupPK`. This
is the end of initialization

##Generate random number
We use BLSState struct to generate random number, as follow:
```
type BLSState struct {
	GroupNodeID     []string                  `json:"groupNodeID"`
	ThresholdNum    int                       `json:"thresholdNum"`
	SkAgg           string                    `json:"skAgg"`
	PkAggVec        map[string]string         `json:"pkAggVec"`
	GroupPK         string                    `json:"groupPK"`
	PreGroupPK      string                    `json:"preGroupPK"`
	PrivkPKE        string                    `json:"privkPKE"`
	PubKPKE         string                    `json:"pubkPKE"`
	PubKeyPKEVec    map[string]string         `json:"PubKeyPKEVec"`
	NodeIndex       int                       `json:"node_index"`
	InitBlsGroup    []*types.BLSNode          `json:"init_bls_group"`
	SkShareSendToMe []*types.SKShareCtAndSign `json:"receive_sk_share"`
	CoeffPloy       []string                  `json:"coeff_ploy"`
	Height          int64                     `json:"height"`
	filePath        string

	mtx                 sync.RWMutex
	stopHeight          int64
	groupSize           int
	genesisMsg          string
	nodeID              string
	signSlice           map[string]string
	isSignReady         bool
	curSign             string
	groupSign           map[int64]string
	peerBLSMsgQueue     chan msgInfo
	internalBLSMsgQueue chan msgInfo
}
```
- `GroupNodeID`: node ID in the BLS group
- `ThresholdNum`: the threshold of the BLS group
- `SkAgg`: private key of current node, generate when init finish
- `PkAggVec`: public key of all node, generate when init finish
- `GroupPK`: group public key of the BLS group
- `PreGroupPK`: index of this node in the BLS group
- `PrivkPKE`: private key of current node from secp256k1 to store in file
- `PubKPKE`: public key of current node from secp256k1 to store in file
- `PubKeyPKEVec`: public key of all node from secp256k1 to store in file
- `NodeIndex`: index of this node in the BLS init group
- `InitBlsGroup`: next BLS init group store in file
- `SkShareSendToMe`: receive the secret from other node
- `CoeffPloy`: the coefficient of polynomial to store in file
- `Height`: the height of BLS random number
- `filePath`: config file path
- `mtx`: the lock
- `stopHeight`: stop height for BLS
- `groupSize`: the size of the BLS group
- `genesisMsg`: genesis msg of BLS
- `nodeID`: ID of current node
- `signSlice`: receive the sign slice of current height
- `isSignReady`: whether generate the group sign
- `curSign`: the group sign of current height
- `groupSign`: store group sign for recent 3 height
- `peerBLSMsgQueue`: msg queue to solve external message
- `internalBLSMsgQueue`: msg queue to solve internal message

When initialization of BLS ends, we get some important operational parameters,
such as `SkAgg`, `GroupPK` and `PkAggVec`, these parameters can be used to
participate in random number generation. The working process is as follows:
all peer use their `SkAgg` to sign on the same message, the message is the
`groupSign` of last height add Hash(`groupSign`), the`groupSign` of last height
is sane for all peers, so message is same. When we sign for the message, broadcast
the sign to other peer. if other peers receive the sign, use `PkAggVec` to verify
the sign. if verify successful, store the sign. Once the number of sign greater
than `ThresholdNum`, recover group sign by the receiving sign slice, and we can
use `GroupPK` to verify the group sign.

##About BLS initialization
We distinguish initialization as the first initialization and continuous
initialization. First initialization use p2p network to send messages between
different peers, If a peer does not receive all message from another peer due to
network problem, the initialization will cause failure. because we can not
guarantee all peer receive message from same peer, in the first initialization,
we required all peer receive all message from others strictly, there is no fault
tolerance in the first initialization.
In continuous initialization, we use p2p network and data in block to broadcast
message in different peers, the data in the block is consistent as you can see,
so we put some important data in block to make consensus in different peers, if
peer A don't reveive data from peer B, peer A will write the info in block that
A don't pass B's secret, when other peer see the message in block, they will
record the info. When they generate BLS data, they will remove the failed node,
use valid data to generate BLS data. So continuous initialization has some fault
tolerance.