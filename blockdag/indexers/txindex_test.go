package indexers

import (
	"bytes"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/daglabs/btcd/blockdag"
	"github.com/daglabs/btcd/dagconfig"
	"github.com/daglabs/btcd/dagconfig/daghash"
	"github.com/daglabs/btcd/util"
	"github.com/daglabs/btcd/wire"
)

func TestTxIndexConnectBlock(t *testing.T) {
	blocks := make(map[daghash.Hash]*util.Block)
	processBlock := func(t *testing.T, dag *blockdag.BlockDAG, msgBlock *wire.MsgBlock, blockName string) {
		block := util.NewBlock(msgBlock)
		blocks[*block.Hash()] = block
		isOrphan, err := dag.ProcessBlock(block, blockdag.BFNone)
		if err != nil {
			t.Fatalf("TestTxIndexConnectBlock: dag.ProcessBlock got unexpected error for block %v: %v", blockName, err)
		}
		if isOrphan {
			t.Fatalf("TestTxIndexConnectBlock: block %v was unexpectedly orphan", blockName)
		}
	}

	txIndex := NewTxIndex()
	indexManager := NewManager([]Indexer{txIndex})

	params := dagconfig.SimNetParams
	params.CoinbaseMaturity = 1
	params.K = 1

	config := blockdag.Config{
		IndexManager: indexManager,
		DAGParams:    &params,
	}

	dag, teardown, err := blockdag.DAGSetup("TestTxIndexConnectBlock", config)
	if err != nil {
		t.Fatalf("TestTxIndexConnectBlock: Failed to setup DAG instance: %v", err)
	}
	if teardown != nil {
		defer teardown()
	}

	processBlock(t, dag, &block1, "1")
	processBlock(t, dag, &block2, "2")
	processBlock(t, dag, &block3, "3")

	block3TxHash := block3Tx.TxHash()
	block3TxNewAcceptedBlock, err := txIndex.BlockThatAcceptedTx(dag, &block3TxHash)
	if err != nil {
		t.Errorf("TestTxIndexConnectBlock: TxAcceptedInBlock: %v", err)
	}
	block3Hash := block3.Header.BlockHash()
	if !block3TxNewAcceptedBlock.IsEqual(&block3Hash) {
		t.Errorf("TestTxIndexConnectBlock: block3Tx should've "+
			"been accepted in block %v but instead got accepted in block %v", block3Hash, block3TxNewAcceptedBlock)
	}

	processBlock(t, dag, &block3A, "3A")
	processBlock(t, dag, &block4, "4")
	processBlock(t, dag, &block5, "5")

	block3TxAcceptedBlock, err := txIndex.BlockThatAcceptedTx(dag, &block3TxHash)
	if err != nil {
		t.Errorf("TestTxIndexConnectBlock: TxAcceptedInBlock: %v", err)
	}
	block3AHash := block3A.Header.BlockHash()
	if !block3TxAcceptedBlock.IsEqual(&block3AHash) {
		t.Errorf("TestTxIndexConnectBlock: block3Tx should've "+
			"been accepted in block %v but instead got accepted in block %v", block3AHash, block3TxAcceptedBlock)
	}

	region, err := txIndex.TxFirstBlockRegion(&block3TxHash)
	if err != nil {
		t.Fatalf("TestTxIndexConnectBlock: no block region was found for block3Tx")
	}
	regionBlock, ok := blocks[*region.Hash]
	if !ok {
		t.Fatalf("TestTxIndexConnectBlock: couldn't find block with hash %v", region.Hash)
	}

	regionBlockBytes, err := regionBlock.Bytes()
	if err != nil {
		t.Fatalf("TestTxIndexConnectBlock: Couldn't serialize block to bytes")
	}
	block3TxInBlock := regionBlockBytes[region.Offset : region.Offset+region.Len]

	block3TxBuf := bytes.NewBuffer(make([]byte, 0, block3Tx.SerializeSize()))
	block3Tx.BtcEncode(block3TxBuf, 0)
	blockTxBytes := block3TxBuf.Bytes()

	if !reflect.DeepEqual(blockTxBytes, block3TxInBlock) {
		t.Errorf("TestTxIndexConnectBlock: the block region that was in the bucket doesn't match block3Tx")
	}

}

var block1 = wire.MsgBlock{
	Header: wire.BlockHeader{
		Version: 1,
		ParentHashes: []daghash.Hash{
			[32]byte{ // Make go vet happy.
				0xca, 0xd9, 0x5f, 0x65, 0x44, 0xd4, 0x2f, 0x08,
				0x23, 0x22, 0x93, 0x4c, 0x07, 0xd9, 0xa4, 0xc0,
				0x1a, 0x51, 0x77, 0xf6, 0x13, 0x7c, 0x06, 0x8b,
				0xd2, 0x6d, 0xe1, 0x38, 0xea, 0x12, 0xcd, 0x4a,
			},
		},
		MerkleRoot: daghash.Hash([32]byte{ // Make go vet happy.
			0x80, 0x57, 0x44, 0xf9, 0xee, 0xb7, 0x14, 0x05,
			0x8c, 0x37, 0x2e, 0x41, 0x82, 0x98, 0xcd, 0x0d,
			0xc8, 0xd1, 0xd1, 0x11, 0x9b, 0xe2, 0xc1, 0x4e,
			0x4b, 0x7c, 0x02, 0xd1, 0x11, 0xe0, 0x50, 0x11,
		}),
		Timestamp: time.Unix(0x5c34c291, 0),
		Bits:      0x207fffff,
		Nonce:     0xdffffffffffffff9,
	},
	Transactions: []*wire.MsgTx{
		{
			Version: 1,
			TxIn: []*wire.TxIn{
				{
					PreviousOutPoint: wire.OutPoint{
						Hash:  daghash.Hash{},
						Index: 0xffffffff,
					},
					SignatureScript: []byte{
						0x51, 0x00, 0x0b, 0x2f, 0x50, 0x32, 0x53, 0x48,
						0x2f, 0x62, 0x74, 0x63, 0x64, 0x2f,
					},
					Sequence: math.MaxUint64,
				},
			},
			TxOut: []*wire.TxOut{
				{
					Value: 5000000000,
					PkScript: []byte{
						0x76, 0xa9, 0x14, 0x3d, 0xee, 0x47, 0x71, 0x6e,
						0x3c, 0xfa, 0x57, 0xdf, 0x45, 0x11, 0x34, 0x73,
						0xa6, 0x31, 0x2e, 0xbe, 0xae, 0xf3, 0x11, 0x88,
						0xac,
					},
				},
			},
			LockTime:     0,
			SubnetworkID: wire.SubnetworkDAGCoin,
		},
	},
}

var block2 = wire.MsgBlock{
	Header: wire.BlockHeader{
		Version: 1,
		ParentHashes: []daghash.Hash{
			[32]byte{ // Make go vet happy.
				0xf1, 0x15, 0xa7, 0xd8, 0x0e, 0xb6, 0x88, 0x25,
				0x1a, 0x9b, 0xc8, 0x6f, 0x1f, 0x71, 0x79, 0xc9,
				0x33, 0xca, 0xd7, 0x79, 0xe5, 0x40, 0x98, 0xd6,
				0x1b, 0x0b, 0x59, 0x3b, 0x98, 0x35, 0x7a, 0x1f,
			},
		},
		MerkleRoot: daghash.Hash([32]byte{ // Make go vet happy.
			0x22, 0x71, 0xda, 0xba, 0x9d, 0x3c, 0xc8, 0xea,
			0xc7, 0x54, 0x26, 0x11, 0x31, 0x1c, 0x1a, 0x09,
			0x70, 0xde, 0x53, 0x6d, 0xaa, 0x32, 0xa6, 0x00,
			0x7a, 0x6b, 0xc4, 0x61, 0x3b, 0xc7, 0x1e, 0x13,
		}),
		Timestamp: time.Unix(0x5c34c292, 0),
		Bits:      0x207fffff,
		Nonce:     0xdffffffffffffffc,
	},
	Transactions: []*wire.MsgTx{
		{
			Version: 1,
			TxIn: []*wire.TxIn{
				{
					PreviousOutPoint: wire.OutPoint{
						Hash:  daghash.Hash{},
						Index: 0xffffffff,
					},
					SignatureScript: []byte{
						0x52, 0x00, 0x0b, 0x2f, 0x50, 0x32, 0x53, 0x48,
						0x2f, 0x62, 0x74, 0x63, 0x64, 0x2f,
					},
					Sequence: math.MaxUint64,
				},
			},
			TxOut: []*wire.TxOut{
				{
					Value: 5000000000,
					PkScript: []byte{
						0x76, 0xa9, 0x14, 0x3d, 0xee, 0x47, 0x71, 0x6e,
						0x3c, 0xfa, 0x57, 0xdf, 0x45, 0x11, 0x34, 0x73,
						0xa6, 0x31, 0x2e, 0xbe, 0xae, 0xf3, 0x11, 0x88,
						0xac,
					},
				},
			},
			LockTime:     0,
			SubnetworkID: wire.SubnetworkDAGCoin,
		},
		{
			Version: 1,
			TxIn: []*wire.TxIn{
				{
					PreviousOutPoint: wire.OutPoint{
						Hash: daghash.Hash{
							0x80, 0x57, 0x44, 0xf9, 0xee, 0xb7, 0x14, 0x05,
							0x8c, 0x37, 0x2e, 0x41, 0x82, 0x98, 0xcd, 0x0d,
							0xc8, 0xd1, 0xd1, 0x11, 0x9b, 0xe2, 0xc1, 0x4e,
							0x4b, 0x7c, 0x02, 0xd1, 0x11, 0xe0, 0x50, 0x11,
						},
						Index: 0,
					},
					SignatureScript: []byte{
						0x47, 0x30, 0x44, 0x02, 0x20, 0x08, 0x3e, 0x75,
						0x3e, 0x0a, 0xbc, 0x0b, 0x39, 0x06, 0xf2, 0x2c,
						0x99, 0x85, 0xf2, 0xde, 0xa7, 0x83, 0x3e, 0x6b,
						0x5a, 0x69, 0x37, 0x51, 0x4c, 0xf8, 0x40, 0x59,
						0x4c, 0x2f, 0x50, 0x1c, 0x04, 0x02, 0x20, 0x06,
						0x21, 0xd9, 0xde, 0x0c, 0x10, 0xca, 0x9d, 0xa4,
						0x5f, 0xe0, 0xfe, 0x3b, 0x33, 0x1d, 0x92, 0x6e,
						0xc4, 0x02, 0xe4, 0x3c, 0xd4, 0x3c, 0xea, 0xf8,
						0xd8, 0xe5, 0x14, 0x3f, 0x56, 0xe9, 0x5b, 0x01,
						0x21, 0x02, 0xa6, 0x73, 0x63, 0x8c, 0xb9, 0x58,
						0x7c, 0xb6, 0x8e, 0xa0, 0x8d, 0xbe, 0xf6, 0x85,
						0xc6, 0xf2, 0xd2, 0xa7, 0x51, 0xa8, 0xb3, 0xc6,
						0xf2, 0xa7, 0xe9, 0xa4, 0x99, 0x9e, 0x6e, 0x4b,
						0xfa, 0xf5,
					},
					Sequence: math.MaxUint64,
				},
			},
			TxOut: []*wire.TxOut{
				{
					Value: 5000000000,
					PkScript: []byte{
						0x76, 0xa9, 0x14, 0x3d, 0xee, 0x47, 0x71, 0x6e,
						0x3c, 0xfa, 0x57, 0xdf, 0x45, 0x11, 0x34, 0x73,
						0xa6, 0x31, 0x2e, 0xbe, 0xae, 0xf3, 0x11, 0x88,
						0xac,
					},
				},
			},
			LockTime:     0,
			SubnetworkID: wire.SubnetworkDAGCoin,
		},
	},
}

var block3Tx = &wire.MsgTx{
	Version: 1,
	TxIn: []*wire.TxIn{
		{
			PreviousOutPoint: wire.OutPoint{
				Hash: daghash.Hash{
					0x65, 0x63, 0x9f, 0x61, 0x7c, 0xaa, 0xc1, 0x4a,
					0x96, 0x7d, 0x8a, 0xc0, 0x4b, 0x97, 0xc5, 0xf3,
					0x86, 0xbe, 0x54, 0x03, 0x26, 0x00, 0x0c, 0xc5,
					0xd8, 0xbb, 0x75, 0x96, 0x1b, 0xdb, 0xa7, 0x5b,
				},
				Index: 0,
			},
			SignatureScript: []byte{
				0x48, 0x30, 0x45, 0x02, 0x21, 0x00, 0x94, 0x6a,
				0x03, 0xb4, 0xab, 0xc3, 0xce, 0x5f, 0xc9, 0x85,
				0xbd, 0xb1, 0xdf, 0x94, 0x26, 0xd0, 0x27, 0x20,
				0x63, 0xdd, 0xd6, 0xd6, 0xce, 0x29, 0xb5, 0xae,
				0x91, 0x50, 0x57, 0x18, 0xc3, 0x26, 0x02, 0x20,
				0x56, 0x99, 0xa2, 0x8a, 0xbb, 0x2f, 0xfe, 0x09,
				0x11, 0x54, 0x42, 0xa7, 0xb3, 0x52, 0x35, 0xf8,
				0xa4, 0x3e, 0x01, 0x61, 0xfa, 0xb9, 0x09, 0x6d,
				0x48, 0x38, 0xa7, 0xc1, 0xfd, 0x6f, 0x9e, 0x5b,
				0x01, 0x21, 0x02, 0xa6, 0x73, 0x63, 0x8c, 0xb9,
				0x58, 0x7c, 0xb6, 0x8e, 0xa0, 0x8d, 0xbe, 0xf6,
				0x85, 0xc6, 0xf2, 0xd2, 0xa7, 0x51, 0xa8, 0xb3,
				0xc6, 0xf2, 0xa7, 0xe9, 0xa4, 0x99, 0x9e, 0x6e,
				0x4b, 0xfa, 0xf5,
			},
			Sequence: math.MaxUint64,
		},
	},
	TxOut: []*wire.TxOut{
		{
			Value: 5000000000,
			PkScript: []byte{
				0x76, 0xa9, 0x14, 0x3d, 0xee, 0x47, 0x71, 0x6e,
				0x3c, 0xfa, 0x57, 0xdf, 0x45, 0x11, 0x34, 0x73,
				0xa6, 0x31, 0x2e, 0xbe, 0xae, 0xf3, 0x11, 0x88,
				0xac,
			},
		},
	},
	LockTime:     0,
	SubnetworkID: wire.SubnetworkDAGCoin,
}

var block3 = wire.MsgBlock{
	Header: wire.BlockHeader{
		Version: 1,
		ParentHashes: []daghash.Hash{
			[32]byte{ // Make go vet happy.
				0x41, 0x27, 0x85, 0x25, 0x0e, 0x8e, 0xdb, 0xf3,
				0xb5, 0xdd, 0xfa, 0xb9, 0x75, 0xc0, 0x4f, 0xe8,
				0x88, 0xff, 0x04, 0x08, 0xe9, 0x0a, 0x93, 0x8f,
				0x45, 0x04, 0x03, 0x73, 0xc6, 0x24, 0x08, 0x72,
			},
		},
		MerkleRoot: daghash.Hash([32]byte{ // Make go vet happy.
			0x93, 0x9b, 0x93, 0x78, 0x9f, 0xca, 0x8c, 0xab,
			0x73, 0x04, 0x64, 0x01, 0xc9, 0x4f, 0x67, 0xf4,
			0xb7, 0x6f, 0x0f, 0xd4, 0x0a, 0xe9, 0x77, 0x81,
			0xa7, 0x18, 0xf8, 0x60, 0xe8, 0x20, 0x45, 0xf2,
		}),
		Timestamp: time.Unix(0x5c34c293, 0),
		Bits:      0x207fffff,
		Nonce:     0xdffffffffffffff9,
	},
	Transactions: []*wire.MsgTx{
		{
			Version: 1,
			TxIn: []*wire.TxIn{
				{
					PreviousOutPoint: wire.OutPoint{
						Hash:  daghash.Hash{},
						Index: 0xffffffff,
					},
					SignatureScript: []byte{
						0x53, 0x00, 0x0b, 0x2f, 0x50, 0x32, 0x53, 0x48,
						0x2f, 0x62, 0x74, 0x63, 0x64, 0x2f,
					},
					Sequence: math.MaxUint64,
				},
			},
			TxOut: []*wire.TxOut{
				{
					Value: 5000000000,
					PkScript: []byte{
						0x76, 0xa9, 0x14, 0x3d, 0xee, 0x47, 0x71, 0x6e,
						0x3c, 0xfa, 0x57, 0xdf, 0x45, 0x11, 0x34, 0x73,
						0xa6, 0x31, 0x2e, 0xbe, 0xae, 0xf3, 0x11, 0x88,
						0xac,
					},
				},
			},
			LockTime:     0,
			SubnetworkID: wire.SubnetworkDAGCoin,
		},
		block3Tx,
	},
}

var block3A = wire.MsgBlock{
	Header: wire.BlockHeader{
		Version: 1,
		ParentHashes: []daghash.Hash{
			[32]byte{ // Make go vet happy.
				0x41, 0x27, 0x85, 0x25, 0x0e, 0x8e, 0xdb, 0xf3,
				0xb5, 0xdd, 0xfa, 0xb9, 0x75, 0xc0, 0x4f, 0xe8,
				0x88, 0xff, 0x04, 0x08, 0xe9, 0x0a, 0x93, 0x8f,
				0x45, 0x04, 0x03, 0x73, 0xc6, 0x24, 0x08, 0x72,
			},
		},
		MerkleRoot: daghash.Hash([32]byte{ // Make go vet happy.
			0x47, 0xb6, 0x23, 0x3a, 0x59, 0xf7, 0x51, 0x40,
			0x41, 0x2e, 0xf1, 0xa3, 0x35, 0xa6, 0x19, 0xa1,
			0x89, 0x33, 0x0b, 0x02, 0x29, 0x3f, 0x8f, 0x35,
			0x92, 0x75, 0x80, 0x61, 0x37, 0x3e, 0x6e, 0x54,
		}),
		Timestamp: time.Unix(0x5c34c293, 0),
		Bits:      0x207fffff,
		Nonce:     0xdffffffffffffffc,
	},
	Transactions: []*wire.MsgTx{
		{
			Version: 1,
			TxIn: []*wire.TxIn{
				{
					PreviousOutPoint: wire.OutPoint{
						Hash:  daghash.Hash{},
						Index: 0xffffffff,
					},
					SignatureScript: []byte{
						0x53, 0x51, 0x0b, 0x2f, 0x50, 0x32, 0x53, 0x48,
						0x2f, 0x62, 0x74, 0x63, 0x64, 0x2f,
					},
					Sequence: math.MaxUint64,
				},
			},
			TxOut: []*wire.TxOut{
				{
					Value: 5000000000,
					PkScript: []byte{
						0x76, 0xa9, 0x14, 0x3d, 0xee, 0x47, 0x71, 0x6e,
						0x3c, 0xfa, 0x57, 0xdf, 0x45, 0x11, 0x34, 0x73,
						0xa6, 0x31, 0x2e, 0xbe, 0xae, 0xf3, 0x11, 0x88,
						0xac,
					},
				},
			},
			LockTime:     0,
			SubnetworkID: wire.SubnetworkDAGCoin,
		},
		block3Tx,
	},
}

var block4 = wire.MsgBlock{
	Header: wire.BlockHeader{
		Version: 1,
		ParentHashes: []daghash.Hash{
			[32]byte{ // Make go vet happy.
				0xf9, 0x3e, 0x6e, 0x3f, 0x22, 0x4b, 0x36, 0xfc,
				0x9b, 0xb4, 0xd1, 0x44, 0xbc, 0x62, 0x78, 0xa0,
				0x2f, 0xef, 0xcc, 0x16, 0xc5, 0x42, 0xbe, 0x59,
				0x22, 0xfe, 0xec, 0x01, 0x55, 0x03, 0x34, 0x62,
			},
		},
		MerkleRoot: daghash.Hash([32]byte{ // Make go vet happy.
			0x79, 0xa0, 0x0e, 0xd0, 0xaa, 0x17, 0x4e, 0xec,
			0x73, 0xd3, 0xcf, 0x13, 0x7f, 0x0d, 0x1d, 0xee,
			0x63, 0x56, 0x3c, 0x2e, 0x17, 0x19, 0x5a, 0x3e,
			0x8b, 0xd2, 0x99, 0xa4, 0xaf, 0xf9, 0xe6, 0x1e,
		}),
		Timestamp: time.Unix(0x5c34c294, 0),
		Bits:      0x207fffff,
		Nonce:     0xdffffffffffffffa,
	},
	Transactions: []*wire.MsgTx{
		{
			Version: 1,
			TxIn: []*wire.TxIn{
				{
					PreviousOutPoint: wire.OutPoint{
						Hash:  daghash.Hash{},
						Index: 0xffffffff,
					},
					SignatureScript: []byte{
						0x54, 0x00, 0x0b, 0x2f, 0x50, 0x32, 0x53, 0x48,
						0x2f, 0x62, 0x74, 0x63, 0x64, 0x2f,
					},
					Sequence: math.MaxUint64,
				},
			},
			TxOut: []*wire.TxOut{
				{
					Value: 5000000000,
					PkScript: []byte{
						0x76, 0xa9, 0x14, 0x3d, 0xee, 0x47, 0x71, 0x6e,
						0x3c, 0xfa, 0x57, 0xdf, 0x45, 0x11, 0x34, 0x73,
						0xa6, 0x31, 0x2e, 0xbe, 0xae, 0xf3, 0x11, 0x88,
						0xac,
					},
				},
			},
			LockTime:     0,
			SubnetworkID: wire.SubnetworkDAGCoin,
		},
	},
}

var block5 = wire.MsgBlock{
	Header: wire.BlockHeader{
		Version: 1,
		ParentHashes: []daghash.Hash{
			[32]byte{ // Make go vet happy.
				0xe6, 0x08, 0x3c, 0x96, 0x4f, 0x4c, 0xb5, 0x37,
				0x2d, 0xd6, 0xe0, 0xe0, 0x85, 0x1a, 0x97, 0x0b,
				0x22, 0x91, 0x13, 0x80, 0x3b, 0xd1, 0xc8, 0x3d,
				0x8f, 0x77, 0xd5, 0xd4, 0x39, 0xc4, 0x9a, 0x09,
			},
			[32]byte{ // Make go vet happy.
				0xfd, 0x28, 0x66, 0x62, 0x56, 0x3e, 0xf0, 0x33,
				0x85, 0xca, 0xf6, 0x96, 0x0d, 0x3a, 0x73, 0xd1,
				0x3b, 0xb8, 0xa0, 0xda, 0xae, 0x4d, 0xdc, 0xa6,
				0x56, 0x82, 0xfd, 0x3b, 0xa0, 0x92, 0x27, 0x38,
			},
		},
		MerkleRoot: daghash.Hash([32]byte{ // Make go vet happy.
			0x29, 0xc6, 0xbc, 0xd9, 0xac, 0x1d, 0x4a, 0x5e,
			0xb0, 0x71, 0xfd, 0xac, 0xde, 0x39, 0xc0, 0x9c,
			0x90, 0xb8, 0x22, 0xde, 0x2d, 0x76, 0x49, 0xab,
			0x80, 0xdc, 0x77, 0xa8, 0xd7, 0x75, 0x40, 0x18,
		}),
		Timestamp: time.Unix(0x5c34c295, 0),
		Bits:      0x207fffff,
		Nonce:     0xdffffffffffffffa,
	},
	Transactions: []*wire.MsgTx{
		{
			Version: 1,
			TxIn: []*wire.TxIn{
				{
					PreviousOutPoint: wire.OutPoint{
						Hash:  daghash.Hash{},
						Index: 0xffffffff,
					},
					SignatureScript: []byte{
						0x55, 0x00, 0x0b, 0x2f, 0x50, 0x32, 0x53, 0x48,
						0x2f, 0x62, 0x74, 0x63, 0x64, 0x2f,
					},
					Sequence: math.MaxUint64,
				},
			},
			TxOut: []*wire.TxOut{
				{
					Value: 5000000000,
					PkScript: []byte{
						0x76, 0xa9, 0x14, 0x3d, 0xee, 0x47, 0x71, 0x6e,
						0x3c, 0xfa, 0x57, 0xdf, 0x45, 0x11, 0x34, 0x73,
						0xa6, 0x31, 0x2e, 0xbe, 0xae, 0xf3, 0x11, 0x88,
						0xac,
					},
				},
			},
			LockTime:     0,
			SubnetworkID: wire.SubnetworkDAGCoin,
		},
	},
}