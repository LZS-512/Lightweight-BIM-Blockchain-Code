package chaincode

import (
    "encoding/json"
    "fmt"
    "time"

    "github.com/hyperledger/fabric-contract-api-go/contractapi"
    "github.com/hyperledger/fabric-chaincode-go/pkg/cid"
)

// ApprovalContract handles BIM model update approval workflow
type ApprovalContract struct {
    contractapi.Contract
}

// BIMApproval extends update info with approval data
type BIMApproval struct {
    UpdateID      string            `json:"UpdateID"`
    ModelID       string            `json:"ModelID"`
    Version       string            `json:"Version"`
    Approver      string            `json:"Approver"`
    ApproveResult string            `json:"ApproveResult"` // APPROVED / REJECTED
    Comment       string            `json:"Comment"`
    Timestamp     string            `json:"Timestamp"`
    Proof         map[string]string `json:"Proof"` // map[approverID]signaturePlaceholder
}

const (
    StatusApproved = "APPROVED"
    StatusRejected = "REJECTED"
    EventBIMApprove = "BIMUpdateApproved"
)

// ApproveBIMUpdate performs approval or rejection of an update
// - Caller must have role=professional
// - Requires UpdateID and approval decision
// - Writes approval proof and updates status
func (c *ApprovalContract) ApproveBIMUpdate(ctx contractapi.TransactionContextInterface,
    updateID string, approveResult string, comment string) error {

    // --- Permission Check: only professional roles allowed ---
    if err := authorizeCallerRole(ctx, RoleProfessional); err != nil {
        return fmt.Errorf("authorization failed: %v", err)
    }

    if updateID == "" {
        return fmt.Errorf("updateID required")
    }
    if approveResult != StatusApproved && approveResult != StatusRejected {
        return fmt.Errorf("invalid approveResult: must be APPROVED or REJECTED")
    }

    // --- Load existing update ---
    updateBytes, err := ctx.GetStub().GetState(updateID)
    if err != nil {
        return fmt.Errorf("failed to read update: %v", err)
    }
    if updateBytes == nil {
        return fmt.Errorf("update %s does not exist", updateID)
    }

    var initUpdate BIMUpdate
    if err := json.Unmarshal(updateBytes, &initUpdate); err != nil {
        return fmt.Errorf("failed to parse update: %v", err)
    }

    // --- Approver identity ---
    approverID, err := getSubmittingClientID(ctx)
    if err != nil {
        return fmt.Errorf("failed to get approver ID: %v", err)
    }

    // --- Build approval record ---
    approval := BIMApproval{
        UpdateID:      updateID,
        ModelID:       initUpdate.ModelID,
        Version:       initUpdate.Version,
        Approver:      approverID,
        ApproveResult: approveResult,
        Comment:       comment,
        Timestamp:     time.Now().UTC().Format(time.RFC3339),
        Proof:         map[string]string{},
    }

    // --- Add approval signature placeholder ---
    approval.Proof[approverID] = fmt.Sprintf("sig:%s", ctx.GetStub().GetTxID())

    // --- Update original update status ---
    initUpdate.Status = approveResult
    merged, err := json.Marshal(initUpdate)
    if err != nil {
        return fmt.Errorf("failed to marshal updated update: %v", err)
    }

    if err := ctx.GetStub().PutState(updateID, merged); err != nil {
        return fmt.Errorf("failed to write updated update: %v", err)
    }

    // --- Store approval record under composite key ---
    key, err := ctx.GetStub().CreateCompositeKey("BIMApproval", []string{updateID})
    if err != nil {
        return fmt.Errorf("failed to create composite key: %v", err)
    }
    approvalBytes, _ := json.Marshal(approval)

    if err := ctx.GetStub().PutState(key, approvalBytes); err != nil {
        return fmt.Errorf("failed to save approval record: %v", err)
    }

    // --- Emit approval event ---
    if err := ctx.GetStub().SetEvent(EventBIMApprove, approvalBytes); err != nil {
        return fmt.Errorf("failed to set event: %v", err)
    }

    return nil
}

// QueryApproval returns approval record for an updateID
func (c *ApprovalContract) QueryApproval(ctx contractapi.TransactionContextInterface, updateID string) (*BIMApproval, error) {
    key, err := ctx.GetStub().CreateCompositeKey("BIMApproval", []string{updateID})
    if err != nil {
        return nil, err
    }
    data, err := ctx.GetStub().GetState(key)
    if err != nil {
        return nil, err
    }
    if data == nil {
        return nil, fmt.Errorf("no approval record for %s", updateID)
    }
    var approval BIMApproval
    if err := json.Unmarshal(data, &approval); err != nil {
        return nil, err
    }
    return &approval, nil
}
