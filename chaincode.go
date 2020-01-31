/*
 * SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	sc "github.com/hyperledger/fabric/protos/peer"
)

// ERC20Chaincode is the definition of the chaincode structure.
type ERC20Chaincode struct {
}

// ERC20Metadata is the definition of Token Meta Info
type ERC20Metadata struct {
	Name        string `json:"name"`
	Symbol      string `json:"symbol"`
	Owner       string `json:"owner"`
	TotalSupply uint64 `json:"totalSupply"`
}

// TransferEvent is the event definition of Transfer
type TransferEvent struct {
	Sender    string `json:"sender"`
	Recipient string `json:"recipient"`
	Amount    int    `json:"amount"`
}

// Init is called when the chaincode is instantiated by the blockchain network.
// params - tokenName, symbol, owner(address), amount
func (cc *ERC20Chaincode) Init(stub shim.ChaincodeStubInterface) sc.Response {
	_, params := stub.GetFunctionAndParameters()
	fmt.Println("Init called with params: ", params)
	if len(params) != 4 {
		return shim.Error("incorrect number of parameter")
	}

	tokenName, symbol, owner, amount := params[0], params[1], params[2], params[3]

	// check amount is unsigned int
	amountUint, err := strconv.ParseUint(string(amount), 10, 64)
	if err != nil {
		return shim.Error("amount must be a number or amount cannot be negative")
	}

	// tokenName & symbol & owner cannot be empty
	if len(tokenName) == 0 || len(symbol) == 0 || len(owner) == 0 {
		return shim.Error("tokenName or symbol or owner cannot be emtpy")
	}

	// make metadata
	erc20 := &ERC20Metadata{Name: tokenName, Symbol: symbol, Owner: owner, TotalSupply: amountUint}
	erc20Bytes, err := json.Marshal(erc20)
	if err != nil {
		return shim.Error("failed to Marshal erc20, error: " + err.Error())
	}

	// save token meta data
	err = stub.PutState(tokenName, erc20Bytes)
	if err != nil {
		return shim.Error("failed to PutState, error: " + err.Error())
	}

	// save owner balance
	err = stub.PutState(owner, []byte(amount))
	if err != nil {
		return shim.Error("failed to PutState, error: " + err.Error())
	}

	// response
	return shim.Success(nil)
}

// Invoke is called as a result of an application request to run the chaincode.
func (cc *ERC20Chaincode) Invoke(stub shim.ChaincodeStubInterface) sc.Response {
	fcn, params := stub.GetFunctionAndParameters()

	switch fcn {
	case "totalSupply":
		return cc.totalSupply(stub, params)
	case "balanceOf":
		return cc.balanceOf(stub, params)
	case "transfer":
		return cc.transfer(stub, params)
	case "allowance":
		return cc.allowance(stub, params)
	case "approve":
		return cc.approve(stub, params)
	case "transferFrom":
		return cc.transferFrom(stub, params)
	case "increaseAllowance":
		return cc.increaseAllowance(stub, params)
	case "decreaseAllowance":
		return cc.decreaseAllowance(stub, params)
	case "mint":
		return cc.mint(stub, params)
	case "burn":
		return cc.burn(stub, params)
	default:
		return sc.Response{Status: 404, Message: "404 Not Found", Payload: nil}
	}
}

// totalSuuply is query function
// params - tokenName
// Returns the amount of token in existence
func (cc *ERC20Chaincode) totalSupply(stub shim.ChaincodeStubInterface, params []string) sc.Response {

	// check the number of params is one
	if len(params) != 1 {
		return shim.Error("incorrect number of parameter")
	}

	tokenName := params[0]

	// Get ERC20 Metadata
	erc20 := ERC20Metadata{}
	erc20Bytes, err := stub.GetState(tokenName)
	if err != nil {
		return shim.Error("failed to GetState, error: " + err.Error())
	}
	err = json.Unmarshal(erc20Bytes, &erc20)
	if err != nil {
		return shim.Error("failed to Unmarshal, error: " + err.Error())
	}

	// Convert TotalSupply to Bytes
	totalSupplyBytes, err := json.Marshal(erc20.TotalSupply)
	if err != nil {
		return shim.Error("failed to Marshal totalSupply, error: " + err.Error())
	}
	fmt.Println(tokenName + "'s totalSupply is " + string(totalSupplyBytes))

	return shim.Success(totalSupplyBytes)
}

// balanceOf is query function
// params - address
// Returns the amount of tokens owned by addresss
func (cc *ERC20Chaincode) balanceOf(stub shim.ChaincodeStubInterface, params []string) sc.Response {

	// check the number of params is one
	if len(params) != 1 {
		return shim.Error("incorrect number of parameters")
	}

	address := params[0]

	// get Balance
	amountBytes, err := stub.GetState(address)
	if err != nil {
		return shim.Error("failed to GetState, error: " + err.Error())
	}

	fmt.Println(address + "'s balance is " + string(amountBytes))

	if amountBytes == nil {
		return shim.Success([]byte("0"))
	}
	return shim.Success(amountBytes)
}

// transfer is invoke function that moves amount token
// from the caller's address to recipient
// params - caller's address, recipient's address, amount of token
func (cc *ERC20Chaincode) transfer(stub shim.ChaincodeStubInterface, params []string) sc.Response {

	// check the number of params is 3
	if len(params) != 3 {
		return shim.Error("incorrect number of parameters")
	}

	callerAddress, recipientAddress, transferAmount := params[0], params[1], params[2]

	// check amount is integer & positive
	transferAmountInt, err := strconv.Atoi(transferAmount)
	if err != nil {
		return shim.Error("transfer amount must be integer")
	}
	if transferAmountInt <= 0 {
		return shim.Error("transfer amount must be positive")
	}

	// get caller amount
	callerAmount, err := stub.GetState(callerAddress)
	if err != nil {
		return shim.Error("failed to GetState, error: " + err.Error())
	}
	callerAmountInt, err := strconv.Atoi(string(callerAmount))
	if err != nil {
		return shim.Error("caller amount must be integer")
	}

	// get recipient amount
	recipientAmount, err := stub.GetState(recipientAddress)
	if err != nil {
		return shim.Error("failed to GetState, error: " + err.Error())
	}
	if recipientAmount == nil {
		recipientAmount = []byte("0")
	}
	recipientAmountInt, err := strconv.Atoi(string(recipientAmount))
	if err != nil {
		return shim.Error("caller amount must be integer")
	}

	// calculate amount
	callerResultAmount := callerAmountInt - transferAmountInt
	recipientResultAmount := recipientAmountInt + transferAmountInt

	// check callerReuslt Amount is positive
	if callerResultAmount < 0 {
		return shim.Error("caller's balance is not sufficient")
	}

	// save the caller's & recipient's amount
	err = stub.PutState(callerAddress, []byte(strconv.Itoa(callerResultAmount)))
	if err != nil {
		return shim.Error("failed to PutState of caller, error: " + err.Error())
	}
	err = stub.PutState(recipientAddress, []byte(strconv.Itoa(recipientResultAmount)))
	if err != nil {
		return shim.Error("failed to PutState of caller, error: " + err.Error())
	}

	// emit transfer event
	transferEvent := TransferEvent{Sender: callerAddress, Recipient: recipientAddress, Amount: transferAmountInt}
	transferEventBytes, err := json.Marshal(transferEvent)
	if err != nil {
		return shim.Error("failed to Marshal transferEvent, error: " + err.Error())
	}
	err = stub.SetEvent("transferEvent", transferEventBytes)
	if err != nil {
		return shim.Error("failed to SetEvent of TransferEvent, error: " + err.Error())
	}

	fmt.Println(callerAddress + " send " + transferAmount + " to " + recipientAddress)

	return shim.Success([]byte("transfer Success"))
}

func (cc *ERC20Chaincode) allowance(stub shim.ChaincodeStubInterface, params []string) sc.Response {

	id, name, amount := params[0], params[1], params[2]

	insuranceKey, _ := stub.CreateCompositeKey("insurance", []string{id, name})

	fmt.Println("insuranceKey: " + insuranceKey)

	stub.PutState(insuranceKey, []byte(amount))

	return shim.Success(nil)

}

func (cc *ERC20Chaincode) approve(stub shim.ChaincodeStubInterface, params []string) sc.Response {

	id := params[0]

	type Insurance struct {
		Name   string `json:"name"`
		Amount string `json:"amount"`
	}
	result := []Insurance{}
	insuranceIterator, err := stub.GetStateByPartialCompositeKey("insurance", []string{id})
	if err != nil {
		return shim.Error("error: " + err.Error())
	}
	for insuranceIterator.HasNext() {
		insuranceKeyValue, _ := insuranceIterator.Next()

		fmt.Println("key: " + insuranceKeyValue.GetKey())
		fmt.Println("value: " + string(insuranceKeyValue.GetValue()))

		objectType, attrs, _ := stub.SplitCompositeKey(insuranceKeyValue.GetKey())
		fmt.Println("objectType: " + objectType)
		fmt.Println("attrs: " + attrs[0] + " // " + attrs[1])
		insurance := Insurance{Name: attrs[1], Amount: string(insuranceKeyValue.GetValue())}

		result = append(result, insurance)
	}

	resultBytes, _ := json.Marshal(result)

	return shim.Success(resultBytes)
}

func (cc *ERC20Chaincode) transferFrom(stub shim.ChaincodeStubInterface, params []string) sc.Response {
	return shim.Success(nil)
}

func (cc *ERC20Chaincode) increaseAllowance(stub shim.ChaincodeStubInterface, params []string) sc.Response {
	return shim.Success(nil)
}

func (cc *ERC20Chaincode) decreaseAllowance(stub shim.ChaincodeStubInterface, params []string) sc.Response {
	return shim.Success(nil)
}

func (cc *ERC20Chaincode) mint(stub shim.ChaincodeStubInterface, params []string) sc.Response {
	return shim.Success(nil)
}

func (cc *ERC20Chaincode) burn(stub shim.ChaincodeStubInterface, params []string) sc.Response {
	return shim.Success(nil)
}
