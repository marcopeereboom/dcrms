package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"decred.org/dcrwallet/rpc/jsonrpc/types"
	jt "decred.org/dcrwallet/rpc/jsonrpc/types"
	"decred.org/dcrwallet/wallet/txrules"
	"decred.org/dcrwallet/wallet/txsizes"
	"github.com/davecgh/go-spew/spew"
	"github.com/decred/dcrd/blockchain/stake/v3"
	"github.com/decred/dcrd/chaincfg/chainhash"
	"github.com/decred/dcrd/dcrutil/v3"
	"github.com/decred/dcrd/txscript/v3"
	"github.com/decred/dcrd/wire"
	it "github.com/decred/dcrdata/api/types"
	"github.com/jrick/wsrpc/v2"
	"github.com/juju/loggo"
)

const (
	defaultConfirmations = 6
)

var (
	log = loggo.GetLogger("dcrms")
)

type client struct {
	cfg *config
}

// httpRequest send an HTTP request to the provided URL.
// XXX add tor
func (c *client) httpRequest(ctx context.Context, url string, timeout time.Duration) ([]byte, error) {
	log.Debugf("httpRequest: %v", url)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to create request: %v", err)
	}

	client := &http.Client{
		Timeout: timeout * time.Second,
	}
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return nil, fmt.Errorf("dcrdata error: %v %v %v",
				response.StatusCode, url, err)
		}
		return nil, fmt.Errorf("dcrdata error: %v %v %s",
			response.StatusCode, url, body)
	}

	return ioutil.ReadAll(response.Body)
}

func (c *client) walletCall(ctx context.Context, method string, res interface{}, params ...interface{}) error {
	tc := &tls.Config{RootCAs: x509.NewCertPool()}
	tc.RootCAs.AppendCertsFromPEM(c.cfg.ca)
	wc, err := wsrpc.Dial(ctx, c.cfg.wallet,
		wsrpc.WithBasicAuth(c.cfg.User, c.cfg.Pass), wsrpc.WithTLSConfig(tc))
	if err != nil {
		return err
	}
	err = wc.Call(ctx, method, res, params...)
	if err != nil {
		return err
	}
	return nil
}

func (c *client) getMultiSigBalance(ctx context.Context, a map[string]string) error {
	address, err := ArgAsString("address", a)
	if err != nil {
		return err
	}

	var addr it.InsightAddressInfo
	url := c.cfg.insight + "/addr/" + address

	resp, err := c.httpRequest(ctx, url, 5*time.Second)
	if err != nil {
		return err
	}
	err = json.Unmarshal(resp, &addr)
	if err != nil {
		return err
	}
	log.Tracef("%v", spew.Sdump(addr))
	fmt.Printf("%v\n", addr.Balance)

	return nil
}

func (c *client) getWalletBalance(ctx context.Context, a map[string]string) error {
	var balance jt.GetBalanceResult
	err := c.walletCall(ctx, "getbalance", &balance)
	if err != nil {
		return err
	}
	log.Tracef("%v", spew.Sdump(balance))
	fmt.Printf("%v\n", balance.TotalSpendable)

	return nil
}

func (c *client) getNewKey(ctx context.Context, a map[string]string) error {
	var address string
	err := c.walletCall(ctx, "getnewaddress", &address, "default", "wrap")
	if err != nil {
		return err
	}
	log.Tracef("%v", address)

	var va jt.ValidateAddressResult
	err = c.walletCall(ctx, "validateaddress", &va, address)
	if err != nil {
		return err
	}
	log.Tracef("%v", spew.Sdump(va))
	if !va.IsValid {
		return fmt.Errorf("address is not valid: %v", address)
	}
	if !va.IsMine {
		return fmt.Errorf("we don't control this address: %v", address)
	}

	fmt.Printf("%v\n", va.PubKeyAddr)

	return nil
}

func (c *client) createMultisigAddress(ctx context.Context, a map[string]string) error {
	n, err := ArgAsUint("n", a)
	if err != nil {
		return err
	}
	keys, err := ArgAsStringSlice("keys", a)
	if err != nil {
		return err
	}

	var msa jt.CreateMultiSigResult
	err = c.walletCall(ctx, "createmultisig", &msa, n, keys)
	if err != nil {
		return err
	}
	log.Tracef("%v", msa)
	fmt.Printf("%v\n", msa.Address)
	// Don't think we need to print redeem script.
	fmt.Printf("%v\n", msa.RedeemScript)

	return nil
}

func (c *client) sendToMultisig(ctx context.Context, a map[string]string) error {
	address, err := ArgAsString("address", a)
	if err != nil {
		return err
	}
	amount, err := ArgAsFloat("amount", a)
	if err != nil {
		return err
	}

	var txHash string
	err = c.walletCall(ctx, "sendtoaddress", &txHash, address, amount)
	if err != nil {
		return err
	}
	fmt.Printf("%v\n", txHash)

	return nil
}

func (c *client) getMultisigOutInfo(ctx context.Context, tx string, vout uint32) (*jt.GetMultisigOutInfoResult, error) {
	var moir jt.GetMultisigOutInfoResult
	err := c.walletCall(ctx, "getmultisigoutinfo", &moir, tx, vout)
	if err != nil {
		return nil, err
	}
	return &moir, nil
}

// getUtxos returns a map of utxos that have had enough confirmations.
func (c *client) getUtxos(ctx context.Context, address string, confirmations int64) (map[string]it.AddressTxnOutput, error) {
	var utxos []it.AddressTxnOutput
	url := c.cfg.insight + "/addr/" + address + "/utxo"

	resp, err := c.httpRequest(ctx, url, 5*time.Second)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(resp, &utxos)
	if err != nil {
		return nil, err
	}
	log.Tracef("%v", spew.Sdump(utxos))

	u := make(map[string]it.AddressTxnOutput, len(utxos))
	for k := range utxos {
		if utxos[k].Confirmations < confirmations {
			continue
		}
		txID := utxos[k].TxnID
		if _, ok := u[txID]; ok {
			return nil, fmt.Errorf("duplicate tx ud: %v", txID)
		}
		u[txID] = utxos[k]

	}

	return u, nil
}

func (c *client) assembleRawTxIns(ctx context.Context, redeemScript []byte, utxos []it.AddressTxnOutput) ([]types.RawTxInput, error) {
	txIns := make([]types.RawTxInput, 0, len(utxos))
	for k := range utxos {
		prevHash, err := chainhash.NewHashFromStr(utxos[k].TxnID)
		if err != nil {
			return nil, fmt.Errorf("decode tx: %v", err)
		}

		// Find the tree, decred specific
		url := c.cfg.dcrdata + "/tx/hex/" + prevHash.String()
		rawTxS, err := c.httpRequest(ctx, url, 5*time.Second)
		if err != nil {
			return nil, fmt.Errorf("get hex tx: %v", err)
		}
		log.Tracef("%v", spew.Sdump(rawTxS))
		rawTx, err := hex.DecodeString(string(rawTxS))
		if err != nil {
			return nil, fmt.Errorf("decode raw hex: %v", err)
		}
		prevTx := wire.NewMsgTx()
		err = prevTx.FromBytes(rawTx)
		if err != nil {
			return nil, fmt.Errorf("decode raw tx: %v", err)
		}
		tree := wire.TxTreeRegular
		st := stake.DetermineTxType(prevTx, true)
		if st != stake.TxTypeRegular {
			tree = wire.TxTreeStake
		}
		//fmt.Printf("prevTX: %v\n", spew.Sdump(prevTx))

		// Add input
		//outPoint := wire.NewOutPoint(prevHash, utxos[k].Vout, tree)
		//txIn := wire.NewTxIn(outPoint, utxos[k].Satoshis, rs)
		txIns = append(txIns, types.RawTxInput{
			Txid:         utxos[k].TxnID,
			Vout:         utxos[k].Vout,
			Tree:         tree,
			ScriptPubKey: utxos[k].ScriptPubKey,
			RedeemScript: hex.EncodeToString(redeemScript),
		})
	}
	return txIns, nil
}

func (c *client) assembleTxIns(ctx context.Context, redeemScript []byte, utxos []it.AddressTxnOutput) ([]*wire.TxIn, error) {
	rs, err := txscript.NewScriptBuilder().AddData(redeemScript).Script()
	if err != nil {
		return nil, err
	}

	txIns := make([]*wire.TxIn, 0, len(utxos))
	for k := range utxos {
		prevHash, err := chainhash.NewHashFromStr(utxos[k].TxnID)
		if err != nil {
			return nil, fmt.Errorf("decode tx: %v", err)
		}

		// Find the tree, decred specific
		url := c.cfg.dcrdata + "/tx/hex/" + prevHash.String()
		rawTxS, err := c.httpRequest(ctx, url, 5*time.Second)
		if err != nil {
			return nil, fmt.Errorf("get hex tx: %v", err)
		}
		log.Tracef("%v", spew.Sdump(rawTxS))
		rawTx, err := hex.DecodeString(string(rawTxS))
		if err != nil {
			return nil, fmt.Errorf("decode raw hex: %v", err)
		}
		prevTx := wire.NewMsgTx()
		err = prevTx.FromBytes(rawTx)
		if err != nil {
			return nil, fmt.Errorf("decode raw tx: %v", err)
		}
		tree := wire.TxTreeRegular
		st := stake.DetermineTxType(prevTx, true)
		if st != stake.TxTypeRegular {
			tree = wire.TxTreeStake
		}

		// Add input
		outPoint := wire.NewOutPoint(prevHash, utxos[k].Vout, tree)
		//txIn := wire.NewTxIn(outPoint, utxos[k].Satoshis, rs)
		_ = rs
		txIn := wire.NewTxIn(outPoint, utxos[k].Satoshis, []byte{})
		txIns = append(txIns, txIn)
	}
	return txIns, nil
}

func (c *client) createMultisigTx(ctx context.Context, a map[string]string) error {
	// Multisig address
	address, err := ArgAsString("address", a)
	if err != nil {
		return err
	}
	change, err := dcrutil.DecodeAddress(address, c.cfg.params)
	if err != nil {
		return err
	}

	// Destination
	to, err := ArgAsString("to", a)
	if err != nil {
		return err
	}
	toAddress, err := dcrutil.DecodeAddress(to, c.cfg.params)
	if err != nil {
		return err
	}

	// Amount
	amount, err := ArgAsFloat("amount", a)
	if err != nil {
		return err
	}
	outValue, err := dcrutil.NewAmount(amount)
	if err != nil {
		return fmt.Errorf("NewAmount: %v", err)
	}

	confirmations, err := ArgAsInt("confirmations", a)
	if err != nil {
		confirmations = defaultConfirmations
	}
	// See if we have enough balance
	var balance jt.GetBalanceResult
	err = c.walletCall(ctx, "getbalance", &balance)
	if err != nil {
		return fmt.Errorf("getbalance: %v", err)
	}
	if balance.TotalSpendable < amount {
		return fmt.Errorf("balance too low: available %v",
			balance.TotalSpendable)
	}

	// Find all utxos
	utxos, err := c.getUtxos(ctx, address, int64(confirmations))
	if err != nil {
		return fmt.Errorf("getUtxos: %v", err)
	}

	// Select utxos
	utxoList := make([]it.AddressTxnOutput, 0, len(utxos))
	foundAmount := 0.0
	for k := range utxos {
		utxoList = append(utxoList, utxos[k])
		foundAmount += utxos[k].Amount
		//fmt.Printf("foundAmount %v\n", foundAmount)
		if foundAmount > amount {
			break
		}
	}
	if len(utxoList) == 0 {
		return fmt.Errorf("0 utxos found to assemble transaction")
	}
	if foundAmount <= amount {
		return fmt.Errorf("not enough total value: %v", foundAmount)
	}
	foundAtoms, err := dcrutil.NewAmount(foundAmount)
	if err != nil {
		return fmt.Errorf("foundAtoms: %v", err)
	}

	//spew.Dump(utxoList)

	// Get redeem script and signers
	moir, err := c.getMultisigOutInfo(ctx, utxoList[0].TxnID,
		utxoList[0].Vout)
	if err != nil {
		return fmt.Errorf("getMultisigOutInfo: %v", err)
	}
	redeemScript, err := hex.DecodeString(moir.RedeemScript)
	if err != nil {
		return fmt.Errorf("decode string: %v", err)
	}
	signers := moir.M

	// Assemble tx
	unsignedTx := wire.NewMsgTx()

	// Output
	script, err := txscript.PayToAddrScript(toAddress)
	if err != nil {
		return fmt.Errorf("DecodeAddress: %v", err)
	}
	txOut := wire.NewTxOut(int64(outValue), script)
	unsignedTx.AddTxOut(txOut)

	// Change
	changeScript, err := txscript.PayToAddrScript(change)
	if err != nil {
		return fmt.Errorf("PayToAddrScript: %v", err)
	}

	// Note that size * signers slightly overpays
	signersSize := int(txsizes.RedeemP2PKHSigScriptSize * signers)
	inputSizes := []int{signersSize, len(redeemScript)}
	outputSizes := []int{txsizes.P2PKHPkScriptSize}
	changeSize := txsizes.P2SHPkScriptSize
	sz := txsizes.EstimateSerializeSizeFromScriptSizes(inputSizes,
		outputSizes, changeSize)
	fee := txrules.FeeForSerializeSize(txrules.DefaultRelayFeePerKb,
		sz)
	txOutChange := wire.NewTxOut(int64(foundAtoms)-int64(outValue+fee),
		changeScript)
	unsignedTx.AddTxOut(txOutChange)

	// Get previous outpoints
	raw := false
	if raw {
		txIns, err := c.assembleRawTxIns(ctx, redeemScript, utxoList)
		if err != nil {
			return fmt.Errorf("getPrevOutpoints: %v", err)
		}
		_ = txIns
	} else {
		txIns, err := c.assembleTxIns(ctx, redeemScript, utxoList)
		if err != nil {
			return fmt.Errorf("getPrevOutpoints: %v", err)
		}
		for k := range txIns {
			unsignedTx.AddTxIn(txIns[k])
		}
	}

	// Dump
	log.Tracef("%v", spew.Sdump(unsignedTx))
	serializedTX, err := unsignedTx.Bytes()
	if err != nil {
		return fmt.Errorf("serialize: %v", err)
	}
	fmt.Printf("%x\n", serializedTX)

	return nil
}

func (c *client) signMultiSigTx(ctx context.Context, a map[string]string) error {
	unsignedTXS, err := ArgAsString("tx", a)
	if err != nil {
		return err
	}
	utxb, err := hex.DecodeString(unsignedTXS)
	if err != nil {
		return fmt.Errorf("DecodeString %v", err)
	}
	//inputs, err := ArgAsString("inputs", a)
	//if err != nil {
	//	return err
	//}
	//var ins []types.RawTxInput
	//err = json.Unmarshal([]byte(inputs), &ins)
	//if err != nil {
	//	return fmt.Errorf("unmarshal: %v", err)
	//}
	//spew.Dump(ins)
	//panic("x")

	unsignedTX := wire.NewMsgTx()
	if err != nil {
		return fmt.Errorf("NewMsgTx: %v", err)
	}
	err = unsignedTX.FromBytes(utxb)
	if err != nil {
		return fmt.Errorf("FromBytes: %v", err)
	}

	var srtr types.SignRawTransactionResult
	err = c.walletCall(ctx, "signrawtransaction", &srtr, unsignedTXS) //, ins)
	if err != nil {
		return err
	}
	log.Tracef("%v", spew.Sdump(srtr))
	fmt.Printf("%v", spew.Sdump(srtr))

	if srtr.Complete {
		fmt.Printf("TRANSACTION SIGNING COMPLETE\n")
	} else {
		fmt.Printf("TRANSACTION SIGNING *NOT* COMPLETE\n")
	}
	fmt.Printf("%v\n", srtr.Hex)

	return nil
}

func (c *client) broadcastMultisigTx(ctx context.Context, a map[string]string) error {
	signedTXS, err := ArgAsString("tx", a)
	if err != nil {
		return err
	}
	utxb, err := hex.DecodeString(signedTXS)
	if err != nil {
		return fmt.Errorf("DecodeString %v", err)
	}

	signedTX := wire.NewMsgTx()
	if err != nil {
		return fmt.Errorf("NewMsgTx: %v", err)
	}
	err = signedTX.FromBytes(utxb)
	if err != nil {
		return fmt.Errorf("FromBytes: %v", err)
	}

	var txHash string
	err = c.walletCall(ctx, "sendrawtransaction", &txHash, signedTXS)
	if err != nil {
		return err
	}
	fmt.Printf("%v\n", txHash)

	return nil
}

func (c *client) multisigInfo(ctx context.Context, a map[string]string) error {
	address, err := ArgAsString("address", a)
	if err != nil {
		return err
	}

	// Find all utxos
	var utxos []it.AddressTxnOutput
	url := c.cfg.insight + "/addr/" + address + "/utxo"
	resp, err := c.httpRequest(ctx, url, 5*time.Second)
	if err != nil {
		return err
	}
	err = json.Unmarshal(resp, &utxos)
	if err != nil {
		return err
	}

	if len(utxos) == 0 {
		return fmt.Errorf("no information available for: %v", address)
	}

	// Fish out the first utxo to get to the multisig address info
	k := 0
	moir, err := c.getMultisigOutInfo(ctx, utxos[k].TxnID,
		utxos[k].Vout)
	if err != nil {
		return fmt.Errorf("getMultisigOutInfo: %v", err)
	}
	log.Tracef("%v", spew.Sdump(moir))
	fmt.Printf("Address      : %v\n", moir.Address)
	fmt.Printf("M            : %v\n", moir.M)
	fmt.Printf("N            : %v\n", moir.N)
	for k := range moir.Pubkeys {
		pubk, err := hex.DecodeString(moir.Pubkeys[k])
		if err != nil {
			fmt.Printf("Could not decode %v: %v\n",
				moir.Pubkeys[k], err)
			continue
		}
		a, err := dcrutil.NewAddressSecpPubKey(pubk, c.cfg.params)
		if err != nil {
			fmt.Printf("Could not decode %v: %v\n",
				moir.Pubkeys[k], err)
			continue
		}
		fmt.Printf("Public key   : %v\n", a)
	}
	fmt.Printf("Redeem script: %v\n", moir.RedeemScript)
	return nil
}

func (c *client) importRedeemScript(ctx context.Context, a map[string]string) error {
	script, err := ArgAsString("script", a)
	if err != nil {
		return err
	}

	err = c.walletCall(ctx, "importscript", nil, script, true)
	if err != nil {
		return err
	}
	return nil
}

func (c *client) sweepMultisig(ctx context.Context, a map[string]string) error {
	return fmt.Errorf("not implemented yet")
}

func (c *client) deserializeTx(ctx context.Context, a map[string]string) error {
	txHexS, err := ArgAsString("tx", a)
	if err != nil {
		return err
	}
	txHex, err := hex.DecodeString(txHexS)
	if err != nil {
		return err
	}

	tx := wire.NewMsgTx()
	err = tx.FromBytes(txHex)
	if err != nil {
		return err
	}
	spew.Dump(tx)

	for k := range tx.TxIn {
		fmt.Printf("%x\n", tx.TxIn[k].SignatureScript)
		disbuf, err := txscript.DisasmString(tx.TxIn[k].SignatureScript)
		if err != nil {
			fmt.Printf("could not decode script %v: %v\n", k, err)
			continue
		}
		fmt.Printf("%v: %v\n", k, disbuf)
	}

	return nil
}

func _main() error {
	cfg, args, err := loadConfig()
	if err != nil {
		return err
	}

	if len(args) == 0 {
		return fmt.Errorf("no action provided")
	}

	// Deal with command line
	a, err := ParseArgs(args)
	if err != nil {
		return err
	}

	// Initialize loggers
	loggo.ConfigureLoggers(cfg.Log)

	ctx := context.Background()

	c := &client{
		cfg: cfg,
	}

	// Handle actions
	if len(args) > 0 {
		switch args[0] {
		case "getmultisigbalance":
			return c.getMultiSigBalance(ctx, a)
		case "getwalletbalance":
			return c.getWalletBalance(ctx, a)
		case "getnewkey":
			return c.getNewKey(ctx, a)
		case "createmultisigaddress":
			return c.createMultisigAddress(ctx, a)
		case "sendtomultisig":
			return c.sendToMultisig(ctx, a)
		case "createmultisigtx":
			return c.createMultisigTx(ctx, a)
		case "signmultisigtx":
			return c.signMultiSigTx(ctx, a)
		case "broadcastmultisigtx":
			return c.broadcastMultisigTx(ctx, a)
		case "multisiginfo":
			return c.multisigInfo(ctx, a)
		case "importredeemscript":
			return c.importRedeemScript(ctx, a)
		case "deserializetx":
			return c.deserializeTx(ctx, a)
		case "sweepmultisig":
			return c.sweepMultisig(ctx, a)
		default:
			return fmt.Errorf("invalid action: %v", args[0])
		}
	}

	return nil
}

func main() {
	err := _main()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
