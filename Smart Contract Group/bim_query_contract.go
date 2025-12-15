package chaincode

import (
    "encoding/json"
    "fmt"

    "github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// QueryContract provides functions to query BIM model update history
type QueryContract struct {
    contractapi.Contract
}

// BIMHistoryRecord combines initialization + approval info for query output
type BIMHistoryRecord struct {
    UpdateID   string      `json:"UpdateID"`
    InitRecord *BIMUpdate  `json:"InitRecord"`
    Approval   *BIMApproval `json:"ApprovalRecord"`
}

// QueryUpdate returns full information of a BIM update transaction
// Includes initialization info + approval record (if exists)
func (qc *QueryContract) QueryUpdate(ctx contractapi.TransactionContextInterface, updateID string) (*BIMHistoryRecord, error) {
    if updateID == "" {
        return nil, fmt.Errorf("updateID required")
    }

    // --- Query initialization record ---
    initBytes, err := ctx.GetStub().GetState(updateID)
    if err != nil {
        return nil, fmt.Errorf("failed to read init record: %v", err)
    }
    if initBytes == nil {
        return nil, fmt.Errorf("no record for update %s", updateID)
    }
    var initRec BIMUpdate
    if err := json.Unmarshal(initBytes, &initRec); err != nil {
        return nil, fmt.Errorf("failed to parse init record: %v", err)
    }

    // --- Query approval record (may not exist yet) ---
    compKey, err := ctx.GetStub().CreateCompositeKey("BIMApproval", []string{updateID})
    if err != nil {
        return nil, fmt.Errorf("failed composite key: %v", err)
    }

    apprBytes, err := ctx.GetStub().GetState(compKey)
    if err != nil {
        return nil, fmt.Errorf("failed to read approval record: %v", err)
    }

    var approvalRec *BIMApproval = nil
    if apprBytes != nil { // approval record exists
        var tmp BIMApproval
        if err := json.Unmarshal(apprBytes, &tmp); err != nil {
            return nil, fmt.Errorf("failed to parse approval record: %v", err)
        }
        approvalRec = &tmp
    }

    // --- Build output structure ---
    history := BIMHistoryRecord{
        UpdateID:   updateID,
        InitRecord: &initRec,
        Approval:   approvalRec,
    }

    return &history, nil
}

// QueryModelHistory lists all updates associated with a BIM model
// Returns a list of init+approval combined results
func (qc *QueryContract) QueryModelHistory(ctx contractapi.TransactionContextInterface, modelID string) ([]*BIMHistoryRecord, error) {
    if modelID == "" {
        return nil, fmt.Errorf("modelID required")
    }

    // Scan all updates (prefix-scan all updates stored directly under keys)
    iterator, err := ctx.GetStub().GetStateByRange("", "")
    if err != nil {
        return nil, err
    }
    defer iterator.Close()

    var result []*BIMHistoryRecord

    for iterator.HasNext() {
        kv, err := iterator.Next()
        if err != nil {
            return nil, err
        }

        // Skip approval composite keys (they include null bytes, filtered out)
        if _, comp := ctx.GetStub().SplitCompositeKey(kv.Key); comp != nil {
            continue
        }

        var maybeInit BIMUpdate
        if err := json.Unmarshal(kv.Value, &maybeInit); err != nil {
            continue // skip invalid JSON
        }

        if maybeInit.ModelID != modelID {
            continue
        }

        // fetch approval if exists
        rec, err := qc.QueryUpdate(ctx, maybeInit.UpdateID)
        if err != nil {
            continue
        }
        result = append(result, rec)
    }

    return result, nil
}

// QueryAllUpdates returns all BIM updates
func (qc *QueryContract) QueryAllUpdates(ctx contractapi.TransactionContextInterface) ([]*BIMHistoryRecord, error) {
    iterator, err := ctx.GetStub().GetStateByRange("", "")
    if err != nil {
        return nil, err
    }
    defer iterator.Close()

    var result []*BIMHistoryRecord

    for iterator.HasNext() {
        kv, err := iterator.Next()
        if err != nil {
            return nil, err
        }

        // skip approval keys
        if _, comp := ctx.GetStub().SplitCompositeKey(kv.Key); comp != nil {
            continue
        }

        var initRec BIMUpdate
        if err := json.Unmarshal(kv.Value, &initRec); err != nil {
            continue
        }

        // full record
        rec, err := qc.QueryUpdate(ctx, initRec.UpdateID)
        if err != nil {
            continue
        }

        result = append(result, rec)
    }

    return result, nil
}
