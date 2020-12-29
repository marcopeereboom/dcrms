package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/decred/dcrd/chaincfg/v3"
	"github.com/decred/dcrd/dcrutil"
	"github.com/jrick/flagfile"
)

const (
	defaultLogging = "dcrms=INFO"
)

var (
	defaultHomeDir    = dcrutil.AppDataDir("dcrms", false)
	defaultConfigFile = filepath.Join(defaultHomeDir, "dcrms.conf")

	dcrwalletFlags = flag.NewFlagSet("dcrwallet.conf flags",
		flag.ContinueOnError)
	dcrwalletHomeDir = dcrutil.AppDataDir("dcrwallet", false)
	dcrwalletConfig  = filepath.Join(dcrwalletHomeDir, "dcrwallet.conf")
	dcrwalletCert    = filepath.Join(dcrwalletHomeDir, "rpc.cert")
)

func versionString() string {
	return "1.0.0"
}

type config struct {
	Config      flag.Value
	ShowVersion bool
	Cert        string
	Wallet      string
	User        string
	Pass        string
	Net         string
	Log         string

	ca      []byte // wallet cert
	wallet  string // wallet websocke
	dcrdata string
	insight string
	params  *chaincfg.Params
}

func usage() {
	fmt.Fprintf(os.Stderr, `Usage of dcrms:
  dcrms [flags] action <args...>
Flags:
  -C <filename>
        Configuration file
  -v	Show version and exit
  -cert <certificate>
	Wallet certificate (uses ~/dcrwallet/rpc.cert by default)
  -wallet <websocket>
	Wallet websocket URL. Default wss://localhost:9110/ws
  -user <username>
	RPC user (reads dcrwallet.conf for defaults)
  -pass <password>
	RPC password (reads dcrwallet.conf for defaults)
  -net <network>
	Network, mainnet or testnet3, default mainnet
  -log	default logging level, default: dcrms=INFO
Actions:
  getmultisigbalance address=<address>
	Print the balance of the multisig address.
  getwalletbalance
	Print total spendable wallet amount
  getnewkey
	Obtain a new public key for a multisig contract
  createmultisigaddress n=<number of signatures required> keys=<public key>,<...>
	Create a multisig address that requires n signatures out of number of keys
  sendtomultisig address=<address> amount=<amount>
	Send funds to an address; wallet must be unlocked
  createmultisigtx address=<address> to=<address> amount=<amount> confirmations=<number>
	Create an unsigned multisig transaction
  signmultisigtx tx=<partially signed transaction>
	Partially, or fully, sign, a multisig transation
  broadcastmultisigtx tx=<signed multisig tx>
	Broadcast multi signature transaction to the network
  multisiginfo address=<public key>
	Print information about the multisg address
  importredeemscript script=<hex>
	Import a redeemscript into the wallet. It takes a few minutes for the wallet to recognize the script.
  sweepmultisig address=<publickey>
	Not implemented yet.
`)
	os.Exit(2)
}

func (c *config) FlagSet() *flag.FlagSet {
	fs := flag.NewFlagSet("dcrms", flag.ExitOnError)
	configParser := flagfile.Parser{AllowUnknown: false}
	c.Config = configParser.ConfigFlag(fs)
	fs.Var(c.Config, "C", "config file")
	fs.BoolVar(&c.ShowVersion, "v", false, "")
	fs.StringVar(&c.Cert, "cert", dcrwalletCert, "")
	fs.StringVar(&c.Wallet, "wallet", "", "")
	fs.StringVar(&c.User, "user", "", "")
	fs.StringVar(&c.Pass, "key", "", "")
	fs.StringVar(&c.Net, "net", "mainnet", "")
	fs.StringVar(&c.Log, "log", defaultLogging, "")
	fs.Usage = usage
	return fs
}

// fileExists returns true if a file exists.
func fileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// cleanAndExpandPath expands environment variables and leading ~ in the
// passed path, cleans the result, and returns it.
func cleanAndExpandPath(path string) string {
	// Expand initial ~ to OS specific home directory.
	if strings.HasPrefix(path, "~") {
		homeDir := filepath.Dir(defaultHomeDir)
		path = strings.Replace(path, "~", homeDir, 1)
	}

	// NOTE: The os.ExpandEnv doesn't work with Windows-style %VARIABLE%,
	// but they variables can still be expanded via POSIX-style $VARIABLE.
	return filepath.Clean(os.ExpandEnv(path))
}

// loadConfig initializes and parses the config using a config file and command
// line options.
//
// The configuration proceeds as follows:
// 	1) Start with a default config with sane settings
// 	2) Pre-parse the command line to check for an alternative config file
// 	3) Load configuration file overwriting defaults with any specified options
// 	4) Parse CLI options and overwrite/add any specified options
//
// The above results in functioning properly without any config settings
// while still allowing the user to override settings with config files and
// command line options.  Command line options always take precedence.
func loadConfig() (*config, []string, error) {
	// Default config.
	cfg := &config{}
	fs := cfg.FlagSet()
	args := os.Args[1:]

	// Determine config file to read (if any).  When -C is the first
	// parameter, configure flags from the specified config file rather than
	// using the application default path.  Otherwise the default config
	// will be parsed if the file exists.
	//
	// If further -C options are specified in later arguments, the config
	// file parameter is used to modify the current state of the config.
	//
	// If you want to read the application default config first, and other
	// configs later, explicitly specify the default path with the first
	// flag argument.
	if len(args) >= 2 && args[0] == "-C" {
		err := cfg.Config.Set(args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid value %q for flag "+
				"-C: %s\n", args[1], err)
			os.Exit(1)
		}
		args = args[2:]
	} else if fileExists(defaultConfigFile) {
		err := cfg.Config.Set(defaultConfigFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	fs.Parse(args)

	// Show the version and exit if the version flag was specified.
	appName := filepath.Base(os.Args[0])
	appName = strings.TrimSuffix(appName, filepath.Ext(appName))
	if cfg.ShowVersion {
		fmt.Printf("%s version %s (Go version %s %s/%s)\n", appName,
			versionString(), runtime.Version(), runtime.GOOS,
			runtime.GOARCH)
		os.Exit(0)
	}

	switch cfg.Net {
	case "mainnet":
		cfg.dcrdata = "https://explorer.dcrdata.org/api"
		cfg.insight = "https://explorer.dcrdata.org/insight/api"
		cfg.params = chaincfg.MainNetParams()
		if cfg.Wallet == "" {
			cfg.wallet = "wss://localhost:9110/ws"
		}
	case "testnet3":
		cfg.dcrdata = "https://testnet.dcrdata.org/api"
		cfg.insight = "https://testnet.dcrdata.org/insight/api"
		cfg.params = chaincfg.TestNet3Params()
		if cfg.Wallet == "" {
			cfg.wallet = "wss://localhost:19110/ws"
		}
	default:
		return nil, nil, fmt.Errorf("invalid net: %v", cfg.Net)
	}

	if cfg.User == "" {
		dcrwalletFlags.StringVar(&cfg.User, "username", "", "rpc user")
	}
	if cfg.Pass == "" {
		dcrwalletFlags.StringVar(&cfg.Pass, "password", "", "rpc pass")
	}
	if cfg.User == "" || cfg.Pass == "" {
		cfgPath := cleanAndExpandPath(dcrwalletConfig)
		cfg, err := os.Open(cfgPath)
		if err != nil {
			return nil, nil, fmt.Errorf("opening config for "+
				"user/pass: %v", err)
		}
		parser := flagfile.Parser{AllowUnknown: true}
		err = parser.Parse(cfg, dcrwalletFlags)
		if err != nil {
			return nil, nil,
				fmt.Errorf("parsing config for user/pass: %v",
					err)
		}
	}
	if cfg.User == "" || cfg.Pass == "" {
		return nil, nil, fmt.Errorf("user or pass unset and not found" +
			" in dcrwallet config file")
	}

	var err error
	cfg.ca, err = ioutil.ReadFile(cfg.Cert)
	if err != nil {
		return nil, nil, fmt.Errorf("can't read wallet certificate: %v",
			err)
	}

	return cfg, fs.Args(), nil
}
