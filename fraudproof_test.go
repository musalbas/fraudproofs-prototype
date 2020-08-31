package fraudproofs

import (
	"bytes"
	"crypto/sha512"
	"fmt"
	"github.com/NebulousLabs/merkletree"
	"github.com/lazyledger/smt"
	"math/rand"
	"testing"
	"time"
)

func TestTransaction(test *testing.T) {
	// create good transaction
	_, err :=  NewTransaction(generateCorruptedTransactionInput())
	if err == nil {
		test.Error("should return an error")
	}

	// create bad transaction
	goodT, err :=  NewTransaction(generateTransactionInput())
	if err != nil {
		test.Error(err)
	}

	// serialize and deserialize
	buff := goodT.Serialize()
	t, err := Deserialize(buff)
	if err != nil {
		test.Error(err)
	} else if bytes.Compare(t.Serialize(), buff) != 0 {
		test.Error("transaction not serialized and deserialize correctly")
	}
}


func TestBlock(test *testing.T) {
	// create bad block (corrupted transactions)
	_, err :=  NewBlock(generateCorruptedBlockInput())
	if err == nil {
		test.Error("should return an error")
	}

	// create good block
	goodTransaction, stateTree := generateBlockInput(1000000)
	goodBlock, err :=  NewBlock(goodTransaction, stateTree)
	if err != nil {
		test.Error(err)
	}

	// check good block
	_, err = goodBlock.CheckBlock(stateTree)
	if err != nil {
		test.Error(err)
	}

	// check a bad block (corrupted transactions)
	badBlock := generateBlockWithCorruptedTransactions()
	_, err = badBlock.CheckBlock(stateTree)
	if err == nil {
		test.Error("should return an error")
	}

	// check bad block (corrupted intermediate state)
	badBlock = corruptBlockInterStates(goodBlock)
	goodFp, err := badBlock.CheckBlock(stateTree)
	if err != nil {
		test.Error(err)
	} else if goodFp == nil {
		test.Error("should return a fraud proof")
	}

	// verify fraud proof of bad block
	ret := badBlock.VerifyFraudProof(*goodFp)
	if ret != true {
		test.Error("fraud proof does not check")
	}

	// verify corrupted fraud proof (corrupted chunks proof)
	corruptedFp := corruptFraudproofChunks(goodFp)
	ret = badBlock.VerifyFraudProof(*corruptedFp)
	if ret != false {
		test.Error("invalid fraud proof should not check")
	}

	// verify corrupted fraud proof (corrupted state proof)
	corruptedFp = corruptFraudproofState(goodFp)
	ret = badBlock.VerifyFraudProof(*corruptedFp)
	if ret != false {
		test.Error("invalid fraud proof should not check")
	}
}

func TestBlockchain(test *testing.T) {
	// add good blocks to blockchain
	blockchain := NewBlockchain()
	goodBlock, _ := NewBlock(generateBlockInput(1000000))
	blockchain.Append(goodBlock) // add a first block
	fp, err := blockchain.Append(goodBlock) // add a second block
	if err != nil {
		test.Error(err)
	} else if fp != nil {
		test.Error("should not return a fraud proof")
	}

	// add bad block to blockchain (corrupted intermediate state)
	fp, err = blockchain.Append(corruptBlockInterStates(goodBlock))
	if err != nil {
		test.Error(err)
	} else if fp == nil {
		test.Error("should return a fraud proof")
	}

	// add bad block to blockchain (corrupted transactions)
	_, err = blockchain.Append(generateBlockWithCorruptedTransactions())
	if err == nil {
		test.Error("should return an error")
	}
}

func TestTiming(test *testing.T) {
	runs := 10
	blockSize := 1000000 // in bytes
	fmt.Println("Number of runs: ", runs)
	fmt.Println("Block size: ", blockSize, "Bytes")

	// create good block
	goodTransaction, stateTree := generateBlockInput(blockSize)
	goodBlock, err :=  NewBlock(goodTransaction, stateTree)
	if err != nil {
		test.Error(err)
	}

	// check bad block (corrupted intermediate state)
	goodBlock = corruptBlockInterStates(goodBlock)
	start := time.Now()
	for i := 0; i < runs; i++ {
		goodFp, err := goodBlock.CheckBlock(stateTree)
		if err != nil {
			test.Error(err)
		} else if goodFp == nil {
			test.Error("should return a fraud proof")
		}
	}
	t := time.Now()
	elapsed := t.Sub(start)
	fmt.Println("generate proof (average): ", int64(elapsed / time.Millisecond) / int64(runs), "ms")

	// verify fraud proof of bad block
	goodFp, err := goodBlock.CheckBlock(stateTree)
	start = time.Now()
	for i := 0; i < runs; i++ {
		ret := goodBlock.VerifyFraudProof(*goodFp)
		if ret != true {
			test.Error("fraud proof does not check")
		}
	}
	t = time.Now()
	elapsed = t.Sub(start)
	fmt.Println("verify proof (average): ", int64(elapsed / time.Microsecond) / int64(runs), "us")
}


// ------------------ helpers ------------------ //


func generateTransactionInput() ([][]byte, [][]byte, [][]byte, [][]byte, [][]byte, []byte) {
	var writeKeys, newData, oldData, readKeys, readData [][]byte

	// average Ethereum transaction size (225B)
	const numWriteKeys = 1
	const numReadKeys = numWriteKeys
	const sizeKeys = 32
	const sizeData = 49

	for i := 0; i < numWriteKeys; i++ {
		token := make([]byte, sizeKeys)
		//rand.Read(token)
		for j := 0; j < len(token); j++ {
			token[j] = 1
		}
		writeKeys = append(writeKeys, token)

		token = make([]byte, sizeData)
		//rand.Read(token)
		for j := 0; j < len(token); j++ {
			token[j] = 2
		}
		newData = append(newData, token)

		token = make([]byte, sizeData)
		//rand.Read(token)
		for j := 0; j < len(token); j++ {
			token[j] = 3
		}
		oldData = append(oldData, token)
	}
	for i := 0; i < numReadKeys; i++ {
		token := make([]byte, sizeKeys)
		rand.Read(token)
		//fmt.Println(len(token), token)
		//for j := 0; j < len(token); j++ {
		//	token[j] = byte(i)
		//}
		//fmt.Println(len(token), token)
		readKeys = append(readKeys, token)

		token = make([]byte, sizeData)
		//rand.Read(token)
		for j := 0; j < len(token); j++ {
			token[j] = 5
		}
		//fmt.Println(token)
		readData = append(readData, token)
	}

	return writeKeys, newData, oldData, readKeys, readData, []byte{}
}

func generateCorruptedTransactionInput() ([][]byte, [][]byte, [][]byte, [][]byte, [][]byte, []byte) {
	writeKeys, newData, oldData, readKeys, readData, arbitrary := generateTransactionInput()
	writeKeys = writeKeys[1:]
	return writeKeys, newData, oldData, readKeys, readData, arbitrary
}

func corruptTransaction(t *Transaction) (*Transaction) {
	t.writeKeys = t.writeKeys[1:]
	return t
}

func generateBlockInput(blockSize int) ([]Transaction, *smt.SparseMerkleTree) {
	// average Ethereum transaction size (225B)
	numTransactions := blockSize / 225 // 4444 transactions for 1MB block
	t := make([]Transaction, numTransactions)
	for i := 0; i < len(t); i++ {
		tmp, _ := NewTransaction(generateTransactionInput())
		t[i] = *tmp
	}
	stateTree := smt.NewSparseMerkleTree(smt.NewSimpleMap(), sha512.New512_256())
	return t, stateTree
}

func generateCorruptedBlockInput() ([]Transaction, *smt.SparseMerkleTree) {
	t1, _ := NewTransaction(generateTransactionInput())
	t2, _ := NewTransaction(generateTransactionInput())

	t1 = corruptTransaction(t1)

	stateTree := smt.NewSparseMerkleTree(smt.NewSimpleMap(), sha512.New512_256())
	return []Transaction{*t1,*t2}, stateTree
}

func generateBlockWithCorruptedTransactions() (*Block) {
	block, _ := NewBlock(generateBlockInput(1000000))
	t := block.transactions[0]
	block.transactions[0] = *corruptTransaction(&t)
	return block
}

func corruptBlockInterStates(b *Block) (*Block) {
	h := sha512.New512_256()
	h.Write([]byte("random"))
	b.interStateRoots[0] = h.Sum(nil)

	dataTree := merkletree.New(sha512.New512_256())
	dataRoot, _ := fillDataTree(b.transactions, b.interStateRoots, dataTree)

	return &Block{
		dataRoot,
		b.stateRoot,
		b.transactions,
		nil,
		dataTree,
		b.interStateRoots}
}

func corruptFraudproofChunks(fp *FraudProof) (*FraudProof) {
	copyFp := copyFraudproof(fp)
	h := sha512.New512_256()
	h.Write([]byte("random"))
	copyFp.proofChunks[0] = [][]byte{h.Sum(nil), h.Sum(nil)}
	return copyFp
}

func corruptFraudproofState(fp *FraudProof) (*FraudProof) {
	copyFp := copyFraudproof(fp)
	h := sha512.New512_256()
	h.Write([]byte("random"))
	copyFp.writeKeys[0] = h.Sum(nil)
	return copyFp
}

// I didn't manage to make a deep copier work (.eg github.com/jinzhu/copier)
func copyFraudproof(fp *FraudProof) (*FraudProof) {
	copyFp := &FraudProof{
		make([][]byte, len(fp.writeKeys)), //writeKeys
		make([][]byte, len(fp.oldData)), //oldData
		make([][]byte, len(fp.readKeys)), //readKeys
		make([][]byte, len(fp.readData)), //readData
		make([]smt.SparseCompactMerkleProof, len(fp.proofState)), //proofState
		make([][]byte, len(fp.chunks)), // chunks
		make([][][]byte, len(fp.proofChunks)), //proofChunks
		make([]uint64, len(fp.chunksIndexes)), // chunksIndexes
		fp.numOfLeaves, // numOfLeaves
	}

	copy(copyFp.writeKeys, fp.writeKeys)
	copy(copyFp.oldData, fp.oldData)
	copy(copyFp.readKeys, fp.readKeys)
	copy(copyFp.readData, fp.readData)
	copy(copyFp.proofState, fp.proofState)
	copy(copyFp.chunks, fp.chunks)
	copy(copyFp.proofChunks, fp.proofChunks)
	copy(copyFp.chunksIndexes, fp.chunksIndexes)

	return copyFp
}
