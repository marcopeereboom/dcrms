## Decred Multi Signature tool - dcrms.

The `dcrms` tool aids in dealing with multi signature adresses and balances on
the decred blockchain. Managing multisig contracts is extremely painful and
error prone and this tool aims at simplifying the steps.

This tool can only offer so many checks and balances. The author does not
assume any responsibility for lost or locked funds. Use at your own risk.

Do note that this tool does NOT use new addresses for deposits and change. It
uses the most basic escrow type mechanism.

The typical workflow for 2:3 keys is as follows:
1. Alice collects public keys addresses from Bob and Charlie
1. Alice creates a multisig address
1. Someone, or someones, fund the multisig address
1. Diana provides Bob an address to receive funds
1. Bob creates a partially signed transaction
1. Bob asks Alice and Charlie to sign transaction
1. Either Alice or Charlie sign the transaction
1. Someone broadcasts the signed transaction to the network

This tool helps simplifying the process quite a bit but in order for this to be
better infrastructure is required to perform simpler key exchanges, backup
information etc.

Required commands:
* createmultisigaddress - Create a multisg address
* getmultisigbalance - Retrieve current multisig address balance
* getwalletbalance - Retrieve wallet total spendable amount
* getnewkey - Get a new public key address from the wallet
* sendtomultisig - Send funds to any address
* createmultisigtx - Create an unsigned multisig transaction
* signmultisigtx - Sign multisig transaction
* broadcastmultisigtx - Broadcast signed multisig transaction to network

Extra commands, for convenience:
* sweepmultisig - Create an unsigned multisig transaction that sweeps the entire multisig address balance.
* multisiginfo - Print multisig address information

```
$ dcrms getnewkey
```

```
$ dcrms createmultisigaddress n=2 keys="xx,yy,zz"
```

```
$ dcrms getmultisigbalance address="publickey"
```

```
$ dcrms getwalletbalance
```

```
$ dcrms sendtomultisig address="addr" amount="1.0"
```

```
$ dcrms createmultisigtx address="publickey" to="toaddr" amount="1.0"
```

```
$ dcrms signmultisigtx tx="hextx"
```

```
$ dcrms broadcastmultisigtx signedtx="hextx"
```

```
$ dcrms sweepmultisig address="publickey"
```

```
$ dcrms multisiginfo address="publickey"
```

## Example workflow

Alice obtains a public key:
```
$ dcrms --net=testnet3 getnewkey
TkKmCGq5rjhgecymkseKC7SoAeUjynXL3naxrzGzbnUurvrXpWwU1
```

Bob obtains a public key:
```
$ dcrms --net=testnet3 getnewkey
TkKmwJxm5axvRekvQR82HHr7BPiUz7pGmUqBP7ez28CVw9cv8y9mB
```

Charlie obtains a public key:
```
$ dcrms --net=testnet3 getnewkey
TkQ4bvLP1uKTwHUuNtrTxrwLBsNaQmfRFGFgHiftv1k16C1Jvcgpy
```

Create a multisig address:
```
$ dcrms --net=testnet3 createmultisigaddress n=2 keys=TkKmCGq5rjhgecymkseKC7SoAeUjynXL3naxrzGzbnUurvrXpWwU1,TkKmwJxm5axvRekvQR82HHr7BPiUz7pGmUqBP7ez28CVw9cv8y9mB,TkQ4bvLP1uKTwHUuNtrTxrwLBsNaQmfRFGFgHiftv1k16C1Jvcgpy
TcerhCZvVVzjYKQoKUybohE75ZxPgPqManG
```

Fund multisig address with 100 DCR (must unlock wallet):
```
$ dcrms --net=testnet3 sendtomultisig address=TcerhCZvVVzjYKQoKUybohE75ZxPgPqManG amount=100.0
ce365de0a58a8ad89d5fd173c0d6191bf4c111448ea112661c8200eb5ca0fb67
```

Verify balance:
```
dcrms --net=testnet3 getmultisigbalance address=TcerhCZvVVzjYKQoKUybohE75ZxPgPqManG
100.0
```

Send 5 DCR to Diane (TsoD8TRGwJdQ3DrxFaV537ffDHnoW3bfD5B)
```
$ dcrms --net=testnet3 createmultisigtx address=TcerhCZvVVzjYKQoKUybohE75ZxPgPqManG to=TsoD8TRGwJdQ3DrxFaV537ffDHnoW3bfD5B amount=5
010000000167fba05ceb00821c6612a18e4411c1f41b19d6c073d15f9dd88a8aa5e05d36ce0000000000ffffffff020065cd1d0000000000001976a914f367538f6c8748c0ddb3f7709070d1b4a977528688ac9c6a3e3602000000000017a914508b7c7fd8e2a2fd49bacf6483b14ebce49fbc238700000000000000000100e40b540200000000000000ffffffff6952210254cf9dc4798eabd6dd1e34a6ea2a4d387bc6b766b1c73609a27d12da3ab9d9772102b687ff58749bd90dd50b37776312d73e91549dccdf81327bda9cb42df855f2652103a2d4d194f1369e147dc88bbc5d7c280ca323da1b660cc7e7782db59db491fd2e53ae
```

Alice signs transaction (wallet figures out privkey):
```
$ dcrms --net=testnet3 signmultisigtx tx=010000000167fba05ceb00821c6612a18e4411c1f41b19d6c073d15f9dd88a8aa5e05d36ce0000000000ffffffff020065cd1d0000000000001976a914f367538f6c8748c0ddb3f7709070d1b4a977528688ac9c6a3e3602000000000017a914508b7c7fd8e2a2fd49bacf6483b14ebce49fbc238700000000000000000100e40b540200000000000000ffffffff6952210254cf9dc4798eabd6dd1e34a6ea2a4d387bc6b766b1c73609a27d12da3ab9d9772102b687ff58749bd90dd50b37776312d73e91549dccdf81327bda9cb42df855f2652103a2d4d194f1369e147dc88bbc5d7c280ca323da1b660cc7e7782db59db491fd2e53ae
010000000167fba05ceb00821c6612a18e4411c1f41b19d6c073d15f9dd88a8aa5e05d36ce0000000000ffffffff020065cd1d0000000000001976a914f367538f6c8748c0ddb3f7709070d1b4a977528688ac9c6a3e3602000000000017a914508b7c7fd8e2a2fd49bacf6483b14ebce49fbc238700000000000000000100e40b540200000000000000fffffffffb47304402200af59a4fba575417a95224c1c323d6d5ac956a75ffb4a6396a4820f36e75c0f3022061f4112da22dae706f7ea3f11c503c7a46b53d8c21b15f8bd5f96859a2622c8601473044022030ba6f87340dfd99378e0d420b04bbbd1b79b674301cfb9743f58ff95174e2c902203f0bb50acecf702de16987bdab7147e2f002ce9f2e92b2b92107b2d11fc9cd08014c6952210254cf9dc4798eabd6dd1e34a6ea2a4d387bc6b766b1c73609a27d12da3ab9d9772102b687ff58749bd90dd50b37776312d73e91549dccdf81327bda9cb42df855f2652103a2d4d194f1369e14
```

Bob signs transaction (wallet figures out privkey):
```
$ dcrms --net=testnet3 signmultisigtx tx=010000000167fba05ceb00821c6612a18e4411c1f41b19d6c073d15f9dd88a8aa5e05d36ce0000000000ffffffff020065cd1d0000000000001976a914f367538f6c8748c0ddb3f7709070d1b4a977528688ac9c6a3e3602000000000017a914508b7c7fd8e2a2fd49bacf6483b14ebce49fbc238700000000000000000100e40b540200000000000000fffffffffb47304402200af59a4fba575417a95224c1c323d6d5ac956a75ffb4a6396a4820f36e75c0f3022061f4112da22dae706f7ea3f11c503c7a46b53d8c21b15f8bd5f96859a2622c8601473044022030ba6f87340dfd99378e0d420b04bbbd1b79b674301cfb9743f58ff95174e2c902203f0bb50acecf702de16987bdab7147e2f002ce9f2e92b2b92107b2d11fc9cd08014c6952210254cf9dc4798eabd6dd1e34a6ea2a4d387bc6b766b1c73609a27d12da3ab9d9772102b687ff58749bd90dd50b37776312d73e91549dccdf81327bda9cb42df855f2652103a2d4d194f1369e14
hextxsigned
```

Broadcast signed transaction to network:
```
$ dcrms --net=testnet3 broadcastmultisigtx signedtx=hextxsigned
95c92b9da481ddf0520252833b0cfa5bb1897283127376c2fd4f310b67194f20
```

## Todo

* Make a better utxo picker
* implement sweep
