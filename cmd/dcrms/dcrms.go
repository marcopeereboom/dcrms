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
	"github.com/decred/dcrd/chaincfg/chainhash"
	"github.com/decred/dcrd/dcrutil/v3"
	"github.com/decred/dcrd/txscript/v3"
	"github.com/decred/dcrd/wire"
	it "github.com/decred/dcrdata/api/types"
	"github.com/jrick/wsrpc/v2"
	"github.com/juju/loggo"
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
	//fmt.Printf("%v\n", msa.RedeemScript)

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

func (c *client) createMultisigTx(ctx context.Context, a map[string]string) error {
	address, err := ArgAsString("address", a)
	if err != nil {
		return err
	}
	change, err := dcrutil.DecodeAddress(address, c.cfg.params)
	if err != nil {
		return err
	}

	to, err := ArgAsString("to", a)
	if err != nil {
		return err
	}
	toAddress, err := dcrutil.DecodeAddress(to, c.cfg.params)
	if err != nil {
		return err
	}

	amount, err := ArgAsFloat("amount", a)
	if err != nil {
		return err
	}
	outValue, err := dcrutil.NewAmount(amount)
	if err != nil {
		return fmt.Errorf("NewAmount: %v", err)
	}

	// See if we have enough balance
	// XXX

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
	log.Tracef("%v", spew.Sdump(utxos))

	lookingFor := amount * 1.05 // Just use 5% for now
	log.Tracef("Looking for amount %v\n", lookingFor)
	for k := range utxos {
		if utxos[k].Amount < lookingFor {
			continue
		}
		log.Tracef("tx %v:%v amount %v\n", utxos[k].TxnID,
			utxos[k].Vout, utxos[k].Amount)

		prevHash, err := chainhash.NewHashFromStr(utxos[k].TxnID)
		if err != nil {
			return fmt.Errorf("decode tx: %v", err)
		}

		// Find the tree, decred specific
		//var dcrouts []it.TxOut
		//url := c.cfg.insight + "/addr/" + address + "/utxo"

		//resp, err := c.httpRequest(ctx, url, 5*time.Second)
		//if err != nil {
		//	return err
		//}
		//err = json.Unmarshal(resp, &utxos)
		//if err != nil {
		//	return err
		//}
		// XXX find tree, getrawtransaction, decode that, get tree
		tree := int8(0)

		// Get redeem script
		moir, err := c.getMultisigOutInfo(ctx, utxos[k].TxnID,
			utxos[k].Vout)
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

		// Inputs
		outPoint := wire.NewOutPoint(prevHash, utxos[k].Vout, tree)
		txIn := wire.NewTxIn(outPoint, utxos[k].Satoshis, redeemScript)
		unsignedTx.AddTxIn(txIn)

		// Outputs
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
		txOutChange := wire.NewTxOut(utxos[k].Satoshis-int64(outValue+fee),
			changeScript)
		unsignedTx.AddTxOut(txOutChange)

		// Dump
		log.Tracef("%v", spew.Sdump(unsignedTx))
		serializedTX, err := unsignedTx.Bytes()
		if err != nil {
			return fmt.Errorf("serialize: %v", err)
		}
		fmt.Printf("%x\n", serializedTX)

		return nil
	}

	return fmt.Errorf("no suitable utxo found for amount: %v", lookingFor)
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

	unsignedTX := wire.NewMsgTx()
	if err != nil {
		return fmt.Errorf("NewMsgTx: %v", err)
	}
	err = unsignedTX.FromBytes(utxb)
	if err != nil {
		return fmt.Errorf("FromBytes: %v", err)
	}

	var srtr types.SignRawTransactionResult
	err = c.walletCall(ctx, "signrawtransaction", &srtr, unsignedTXS)
	if err != nil {
		return err
	}
	log.Tracef("%v", spew.Sdump(srtr))

	if len(srtr.Errors) > 0 {
		return fmt.Errorf("SignRawTransaction: %v", srtr)
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

func (c *client) sweepMultisig(ctx context.Context, a map[string]string) error {
	return fmt.Errorf("not implemented yet")
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
