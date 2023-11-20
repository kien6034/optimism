package types

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	ErrGameDepthReached = errors.New("game depth reached")

	// NoLocalContext is the LocalContext value used when the cannon trace provider is used alone instead of as part
	// of a split game.
	NoLocalContext = common.Hash{}
)

// LocalContextPreimage type contains the pre and post claims for the
// local context hash.
type LocalContextPreimage struct {
	Pre  Claim
	Post Claim
}

// NewLocalContextPreimage creates a new [LocalContextPreimage] instance.
func NewLocalContextPreimage(pre Claim, post Claim) *LocalContextPreimage {
	return &LocalContextPreimage{
		Pre:  pre,
		Post: post,
	}
}

// UsePrestateBlock returns true if the prestate block should be used based
// on the pre claim.
func (l *LocalContextPreimage) UsePrestateBlock() bool {
	return l.Pre == (Claim{})
}

// Hash returns the Local Context using the preimage.
func (l *LocalContextPreimage) Hash() common.Hash {
	return crypto.Keccak256Hash(l.Preimage())
}

// Preimage returns the preimage for the local context.
func (l *LocalContextPreimage) Preimage() []byte {
	encodeClaim := func(c Claim) []byte {
		data := make([]byte, 64)
		copy(data[0:32], c.Value.Bytes())
		c.Position.ToGIndex().FillBytes(data[32:])
		return data
	}
	var data []byte
	if !l.UsePrestateBlock() {
		data = encodeClaim(l.Pre)
	}
	data = append(data, encodeClaim(l.Post)...)
	return data
}

// PreimageOracleData encapsulates the preimage oracle data
// to load into the onchain oracle.
type PreimageOracleData struct {
	IsLocal      bool
	LocalContext common.Hash
	OracleKey    []byte
	OracleData   []byte
	OracleOffset uint32
}

// GetLocalContextBigInt returns the local context as a big int.
func (p *PreimageOracleData) GetLocalContextBigInt() *big.Int {
	return new(big.Int).SetBytes(p.LocalContext.Bytes())
}

// GetOracleOffsetBigInt returns the oracle offset as a big int.
func (p *PreimageOracleData) GetOracleOffsetBigInt() *big.Int {
	return new(big.Int).SetUint64(uint64(p.OracleOffset))
}

// GetIdent returns the ident for the preimage oracle data.
func (p *PreimageOracleData) GetIdent() *big.Int {
	return new(big.Int).SetBytes(p.OracleKey[1:])
}

// GetPreimageWithoutSize returns the preimage for the preimage oracle data.
func (p *PreimageOracleData) GetPreimageWithoutSize() []byte {
	return p.OracleData[8:]
}

// NewPreimageOracleData creates a new [PreimageOracleData] instance.
func NewPreimageOracleData(lctx common.Hash, key []byte, data []byte, offset uint32) *PreimageOracleData {
	return &PreimageOracleData{
		IsLocal:      len(key) > 0 && key[0] == byte(1),
		LocalContext: lctx,
		OracleKey:    key,
		OracleData:   data,
		OracleOffset: offset,
	}
}

// StepCallData encapsulates the data needed to perform a step.
type StepCallData struct {
	ClaimIndex uint64
	IsAttack   bool
	StateData  []byte
	Proof      []byte
}

// TraceAccessor defines an interface to request data from a TraceProvider with additional context for the game position.
// This can be used to implement split games where lower layers of the game may have different values depending on claims
// at higher levels in the game.
type TraceAccessor interface {
	// Get returns the claim value at the requested position, evaluated in the context of the specified claim (ref).
	Get(ctx context.Context, game Game, ref Claim, pos Position) (common.Hash, error)

	// GetStepData returns the data required to execute the step at the specified position,
	// evaluated in the context of the specified claim (ref).
	GetStepData(ctx context.Context, game Game, ref Claim, pos Position) (prestate []byte, proofData []byte, preimageData *PreimageOracleData, err error)
}

// TraceProvider is a generic way to get a claim value at a specific step in the trace.
type TraceProvider interface {
	// Get returns the claim value at the requested index.
	// Get(i) = Keccak256(GetPreimage(i))
	Get(ctx context.Context, i Position) (common.Hash, error)

	// GetStepData returns the data required to execute the step at the specified trace index.
	// This includes the pre-state of the step (not hashed), the proof data required during step execution
	// and any pre-image data that needs to be loaded into the oracle prior to execution (may be nil)
	// The prestate returned from GetStepData for trace 10 should be the pre-image of the claim from trace 9
	GetStepData(ctx context.Context, i Position) (prestate []byte, proofData []byte, preimageData *PreimageOracleData, err error)

	// AbsolutePreStateCommitment is the commitment of the pre-image value of the trace that transitions to the trace value at index 0
	AbsolutePreStateCommitment(ctx context.Context) (hash common.Hash, err error)
}

// ClaimData is the core of a claim. It must be unique inside a specific game.
type ClaimData struct {
	Value common.Hash
	Position
}

func (c *ClaimData) ValueBytes() [32]byte {
	responseBytes := c.Value.Bytes()
	var responseArr [32]byte
	copy(responseArr[:], responseBytes[:32])
	return responseArr
}

// Claim extends ClaimData with information about the relationship between two claims.
// It uses ClaimData to break cyclicity without using pointers.
// If the position of the game is Depth 0, IndexAtDepth 0 it is the root claim
// and the Parent field is empty & meaningless.
type Claim struct {
	ClaimData
	// WARN: Countered is a mutable field in the FaultDisputeGame contract
	//       and rely on it for determining whether to step on leaf claims.
	//       When caching is implemented for the Challenger, this will need
	//       to be changed/removed to avoid invalid/stale contract state.
	Countered bool
	Clock     uint64
	// Location of the claim & it's parent inside the contract. Does not exist
	// for claims that have not made it to the contract.
	ContractIndex       int
	ParentContractIndex int
}

// IsRoot returns true if this claim is the root claim.
func (c *Claim) IsRoot() bool {
	return c.Position.IsRootPosition()
}
