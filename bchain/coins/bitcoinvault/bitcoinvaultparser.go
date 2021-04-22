package bitcoinvault

import (
	"bytes"

	"github.com/martinboehm/btcd/wire"
	"github.com/martinboehm/btcutil/chaincfg"
	"github.com/trezor/blockbook/bchain"
	"github.com/trezor/blockbook/bchain/coins/btc"
	"github.com/trezor/blockbook/bchain/coins/utils"
)

// magic numbers
const (
	MainnetMagic wire.BitcoinNet = 0xc0c0c0c0
)

// chain parameters
var (
	MainNetParams chaincfg.Params
)

// From https://github.com/bitcoinvault/bitcoinvault/blob/a6ab4e7a6dda8e2b8fda4a6c4a558bd412f23d89/src/policy/auxpow.h#L11
// Except these are not block hashes, but their parent block hashes
var FAKE_AUXPOW_PREFORK_BLOCK_PARENTS = map[string]bool{
	"0000000000000000144c7fb7dad69be270035fa2d7f4652819369d9e3bb8023d": true,
	"000000000000000017b3b639f6f4ff8618c0fe5fbc44c39fa924a60974be6a2d": true,
	"0000000000000000022a3e19cd72f7012a2089fa5854ac42374bd48c010f1f4f": true,
	"00000000000000001e8818e5c81c2f224fbf21f7be90888f4c6314bb56f11d3f": true,
};

func isFakeAuxpowPreforkBlockParent(hash string) bool {
	return FAKE_AUXPOW_PREFORK_BLOCK_PARENTS[hash]
}

func init() {
	MainNetParams = chaincfg.MainNetParams
	MainNetParams.Net = MainnetMagic

	// From https://github.com/bitcoinvault/bitcoinvault/blob/a6ab4e7a6dda8e2b8fda4a6c4a558bd412f23d89/src/chainparams.cpp#L138-L144
	MainNetParams.PubKeyHashAddrID = []byte{78}
	MainNetParams.ScriptHashAddrID = []byte{60}
	MainNetParams.Bech32HRPSegwit = "royale"
}

// BitcoinvaultParser handle
type BitcoinvaultParser struct {
	*btc.BitcoinParser
}

// NewBitcoinvaultParser returns new BitcoinvaultParser instance
func NewBitcoinvaultParser(params *chaincfg.Params, c *btc.Configuration) *BitcoinvaultParser {
	return &BitcoinvaultParser{BitcoinParser: btc.NewBitcoinParser(params, c)}
}

// GetChainParams contains network parameters for the main Bitcoinvault network,
// and the test Bitcoinvault network
func GetChainParams(chain string) *chaincfg.Params {
	if !chaincfg.IsRegistered(&MainNetParams) {
		err := chaincfg.Register(&MainNetParams)
		if err != nil {
			panic(err)
		}
	}
	switch chain {
	default:
		return &MainNetParams
	}
}

// ParseBlock parses raw block to our Block struct
// it has special handling for Auxpow blocks that cannot be parsed by standard btc wire parser
func (p *BitcoinvaultParser) ParseBlock(b []byte) (*bchain.Block, error) {
	r := bytes.NewReader(b)
	w := wire.MsgBlock{}
	h := wire.BlockHeader{}
	err := h.Deserialize(r)

	if err != nil {
		return nil, err
	}

	if (h.Version & utils.VersionAuxpow) != 0 && !isFakeAuxpowPreforkBlockParent(h.PrevBlock.String()) {
		if err = utils.SkipAuxpow(r); err != nil {
			return nil, err
		}
	}

	err = utils.DecodeTransactions(r, 0, wire.WitnessEncoding, &w)
	if err != nil {
		return nil, err
	}

	txs := make([]bchain.Tx, len(w.Transactions))
	for ti, t := range w.Transactions {
		txs[ti] = p.TxFromMsgTx(t, false)
	}

	return &bchain.Block{
		BlockHeader: bchain.BlockHeader{
			Size: len(b),
			Time: h.Timestamp.Unix(),
		},
		Txs: txs,
	}, nil
}
