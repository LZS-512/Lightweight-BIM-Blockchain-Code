package mapping

import (
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "errors"
    "fmt"
    "time"
)

// -------------------------------
//  数据结构定义
// -------------------------------

// UserInfo 用户信息（工号、部门等）
type UserInfo struct {
    UserID     string `json:"userId"`
    Department string `json:"department"`
    Role       string `json:"role"`
}

// BIMInitInfo 经过 IPFS 处理得到的 BIM 文件初始信息
type BIMInitInfo struct {
    FileName string `json:"fileName"`
    CID      string `json:"cid"` // 来自 IPFS
    FileHash string `json:"fileHash"`
}

// Transaction 封装后的完整交易结构
type Transaction struct {
    TxID      string      `json:"txId"`
    Timestamp int64       `json:"timestamp"`
    User      UserInfo    `json:"user"`
    BIM       BIMInitInfo `json:"bim"`
}

// NodeMapping 区块链节点映射结果
type NodeMapping struct {
    NodeURL string `json:"nodeUrl"`
    OrgName string `json:"orgName"`
}

// -------------------------------
//  IPFS 模拟接口（可替换为实际 IPFS SDK）
// -------------------------------

// SimulateIPFSUpload 模拟上传 BIM 文件到 IPFS，返回 CID
func SimulateIPFSUpload(fileContent []byte) (string, string) {
    hash := sha256.Sum256(fileContent)
    fileHash := hex.EncodeToString(hash[:])

    cid := "CID-" + fileHash[:16]
    return cid, fileHash
}

// -------------------------------
// 1. 初始信息处理功能（调用 IPFS）
// -------------------------------

func ProcessInitialInfo(fileName string, content []byte) (*BIMInitInfo, error) {
    cid, fileHash := SimulateIPFSUpload(content)

    initInfo := BIMInitInfo{
        FileName: fileName,
        CID:      cid,
        FileHash: fileHash,
    }

    return &initInfo, nil
}

// -------------------------------
// 2. 获取用户信息功能
// -------------------------------

// GetUserInfo 模拟从企业系统获取用户信息
func GetUserInfo(userID string) (*UserInfo, error) {
    // 模拟部门映射规则
    mapping := map[string]string{
        "1001": "architecture",
        "1002": "structure",
        "1003": "me",
        "2001": "management",
    }

    dept, ok := mapping[userID]
    if !ok {
        return nil, errors.New("用户不存在")
    }

    info := UserInfo{
        UserID:     userID,
        Department: dept,
        Role:       "designer", // 默认角色
    }
    return &info, nil
}

// -------------------------------
// 3. 交易封装功能（用户信息 + 初始信息）
// -------------------------------

func PackageTransaction(user *UserInfo, bim *BIMInitInfo) (*Transaction, error) {
    if user == nil || bim == nil {
        return nil, errors.New("用户信息或 BIM 信息为空")
    }

    tx := Transaction{
        TxID:      fmt.Sprintf("TX-%d", time.Now().UnixNano()),
        Timestamp: time.Now().Unix(),
        User:      *user,
        BIM:       *bim,
    }
    return &tx, nil
}

// -------------------------------
// 4. 映射到区块链节点功能
// -------------------------------

// MapToBlockchainNode 根据部门选择对应区块链节点
func MapToBlockchainNode(department string) (*NodeMapping, error) {
    nodeMap := map[string]NodeMapping{
        "architecture": {NodeURL: "grpc://node1.arch.example.com:7051", OrgName: "Org1"},
        "structure":    {NodeURL: "grpc://node2.struct.example.com:7051", OrgName: "Org2"},
        "me":           {NodeURL: "grpc://node3.me.example.com:7051", OrgName: "Org3"},
        "management":   {NodeURL: "grpc://node4.mgmt.example.com:7051", OrgName: "Org4"},
    }

    node, ok := nodeMap[department]
    if !ok {
        return nil, errors.New("部门未映射至任何区块链节点")
    }
    return &node, nil
}

// -------------------------------
// 工具包入口函数：完整一对多映射流程
// -------------------------------

func RunOneToManyMapping(fileName string, fileContent []byte, userID string) (string, error) {
    // 1. 处理 BIM 初始信息
    bimInfo, err := ProcessInitialInfo(fileName, fileContent)
    if err != nil {
        return "", err
    }

    // 2. 获取用户信息
    user, err := GetUserInfo(userID)
    if err != nil {
        return "", err
    }

    // 3. 封装交易
    tx, err := PackageTransaction(user, bimInfo)
    if err != nil {
        return "", err
    }

    // 4. 映射节点
    node, err := MapToBlockchainNode(user.Department)
    if err != nil {
        return "", err
    }

    // 模拟把交易发往区块链
    payload, _ := json.MarshalIndent(tx, "", "  ")

    result := fmt.Sprintf("发送至节点: %s (组织: %s)\n交易内容:\n%s", node.NodeURL, node.OrgName, string(payload))

    return result, nil
}
