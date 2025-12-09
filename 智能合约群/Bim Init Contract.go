package chaincode

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	"github.com/hyperledger/fabric-chaincode-go/pkg/cid"
)

// SmartContract implements chaincode for BIM model update initialization
type SmartContract struct {
	contractapi.Contract
}

// BIMUpdate represents an initialization request for a BIM model update
type BIMUpdate struct {
	UpdateID    string            `json:"UpdateID"`
	ModelID     string            `json:"ModelID"`
	Version     string            `json:"Version"`
	Description string            `json:"Description"`
	Initiator   string            `json:"Initiator"`
	Timestamp   string            `json:"Timestamp"`
	Signatures  map[string]string `json:"Signatures"` // map[endorserID]signaturePlaceholder
	Status      string            `json:"Status"`     // e.g., "INITIALIZED", "APPROVED", "REJECTED", "PUBLISHED"
}

// Role constants (these should match attributes set in certificates)
const (
	RoleAttrName       = "role"
	RoleModeler        = "modeler"
	RoleProfessional   = "professional"
	RoleBIMLead        = "bim_lead"
	EventBIMInit       = "BIMUpdateInitialized"
	StatusInitialized  = "INITIALIZED"
)

// InitLedger optional: add demo data
func (s *SmartContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
	return nil
}

// InitBIMUpdate initializes a BIM model update transaction on the ledger.
// - caller must have role=modeler (or other allowed roles per policy)
// - validates incoming payload
// - collects creator identity and a placeholder for endorsement/signature
// - stores the BIMUpdate with status INITIALIZED and emits an event
func (s *SmartContract) InitBIMUpdate(ctx contractapi.TransactionContextInterface, updateJSON string) error {
	// permission check
	if err := authorizeCallerRole(ctx, RoleModeler); err != nil {
		return fmt.Errorf("authorization failed: %v", err)
	}

	// parse input
	var input BIMUpdate
	if err := json.Unmarshal([]byte(updateJSON), &input); err != nil {
		return fmt.Errorf("failed to parse update JSON: %v", err)
	}

	// basic validation
	if input.UpdateID == "" {
		return fmt.Errorf("UpdateID is required")
	}
	if input.ModelID == "" {
		return fmt.Errorf("ModelID is required")
	}
	if input.Version == "" {
		return fmt.Errorf("Version is required")
	}

	// check existence
	exists, err := s.UpdateExists(ctx, input.UpdateID)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("update %s already exists", input.UpdateID)
	}

	// capture creator identity
	creatorID, err := getSubmittingClientID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get creator identity: %v", err)
	}

	// attach initiator and timestamp
	input.Initiator = creatorID
	input.Timestamp = time.Now().UTC().Format(time.RFC3339)
	input.Status = StatusInitialized

	// gather endorsement signatures placeholder
	// In Fabric chaincode we cannot directly collect peer endorsements; however,
	// we can capture the client's identity and signed proposal metadata as a placeholder.
	sigMap := map[string]string{}
	sigMap[creatorID] = fmt.Sprintf("sig:%s", ctx.GetStub().GetTxID())
	input.Signatures = sigMap

	// write to world state
	data, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("failed to marshal BIMUpdate: %v", err)
	}

	if err := ctx.GetStub().PutState(input.UpdateID, data); err != nil {
		return fmt.Errorf("failed to put BIMUpdate to world state: %v", err)
	}

	// emit event so off-chain components (endorsement collectors, UI) can react
	if err := ctx.GetStub().SetEvent(EventBIMInit, data); err != nil {
		return fmt.Errorf("failed to set event: %v", err)
	}

	return nil
}

// UpdateExists returns whether a BIMUpdate with given id exists
func (s *SmartContract) UpdateExists(ctx contractapi.TransactionContextInterface, id string) (bool, error) {
	b, err := ctx.GetStub().GetState(id)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state: %v", err)
	}
	return b != nil, nil
}

// ReadUpdate retrieves a BIMUpdate from world state
func (s *SmartContract) ReadUpdate(ctx contractapi.TransactionContextInterface, id string) (*BIMUpdate, error) {
	data, err := ctx.GetStub().GetState(id)
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %v", err)
	}
	if data == nil {
		return nil, fmt.Errorf("the update %s does not exist", id)
	}
	var update BIMUpdate
	if err := json.Unmarshal(data, &update); err != nil {
		return nil, err
	}
	return &update, nil
}

// Helper: getSubmittingClientID returns a human-readable ID for the transaction submitter
func getSubmittingClientID(ctx contractapi.TransactionContextInterface) (string, error) {
	ci, err := cid.New(ctx.GetStub())
	if err != nil {
		return "", err
	}
	id, err := ci.GetID()
	if err != nil {
		return "", err
	}
	return id, nil
}

// authorizeCallerRole checks the caller's certificate attribute 'role' equals expected
func authorizeCallerRole(ctx contractapi.TransactionContextInterface, expected string) error {
	ci, err := cid.New(ctx.GetStub())
	if err != nil {
		return fmt.Errorf("failed to create client identity: %v", err)
	}
	role, found, err := ci.GetAttributeValue(RoleAttrName)
	if err != nil {
		return fmt.Errorf("failed to read attribute '%s': %v", RoleAttrName, err)
	}
	if !found {
		return fmt.Errorf("attribute '%s' not found in identity", RoleAttrName)
	}
	if role != expected {
		return fmt.Errorf("caller role '%s' not authorized (expected '%s')", role, expected)
	}
	return nil
}

// Note:
// - This contract focuses on initialization. Approval, proof-generation, and query contracts
//   should be implemented separately as described in your design.
// - Endorsement collection in Fabric happens at the client/peer level. Chaincode can store
//   endorsement metadata or placeholders; actual peer signatures are available in the
//   proposal/endorsement objects but are typically processed outside chaincode logic.
// - For production, extend validation (schema, business rules), better error handling,
//   and logging, and integrate with off-chain services (notification/consensus helpers).
