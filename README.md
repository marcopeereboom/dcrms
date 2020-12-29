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
$ dcrms createmultisigtx address="publickey" to="toaddr" amount="1.0" confirmations="6"
```

```
$ dcrms signmultisigtx tx="hextx"
```

```
$ dcrms broadcastmultisigtx tx="hextx"
```

```
$ dcrms sweepmultisig address="publickey"
```

```
$ dcrms multisiginfo address="publickey"
```

```
$ dcrms importredeemscript script="script"
```

## Example workflow for a 2 of 2

Alice obtains a public key:
```
$ dcrms --net=testnet3 getnewkey
TkQ4HxtQMvWQ8VypqyR6nj4xxwxEU667ceEH2EfyEZajhCLZFwkwx
```

Bob obtains a public key:
```
$ dcrms --net=testnet3 getnewkey
TkKnHcV4PEd3Cz9NvAV9hCAqAiW2zyWBsgMvEcou78KtrNKBKmJU7
```

Alice creates a multisig address:
```
$ dcrms --net=testnet3 createmultisigaddress n=2 keys=TkQ4HxtQMvWQ8VypqyR6nj4xxwxEU667ceEH2EfyEZajhCLZFwkwx,TkKnHcV4PEd3Cz9NvAV9hCAqAiW2zyWBsgMvEcou78KtrNKBKmJU7
Tcgeou3sAxjvyFecHcLDaQwJ59gTHZ53Fsi
5221037a0f5e5616bf12e395d3cef6d65ea38bf3bc570fc2b353a169ac30bda84f63932102e4a131b2f1dce7177b18a97265d99b91aed29cbdbac9142e2539dcd7fb01429852ae
```

Alice imports redeem script:
```
$ dcrms --net=testnet3 importredeemscript script=5221037a0f5e5616bf12e395d3cef6d65ea38bf3bc570fc2b353a169ac30bda84f63932102e4a131b2f1dce7177b18a97265d99b91aed29cbdbac9142e2539dcd7fb01429852ae
```

Bob imports redeem script:
```
$ dcrms --net=testnet3 importredeemscript script=5221037a0f5e5616bf12e395d3cef6d65ea38bf3bc570fc2b353a169ac30bda84f63932102e4a131b2f1dce7177b18a97265d99b91aed29cbdbac9142e2539dcd7fb01429852ae
```

Alice funds multisig address with 100 DCR (must unlock wallet):
```
$ dcrms --net=testnet3 sendtomultisig address=Tcgeou3sAxjvyFecHcLDaQwJ59gTHZ53Fsi amount=100.0
fd8fbc1a116a6883c001491f884f3488ae1f6ffcd468738ae302785a6b8b147f
```

Alice verifies multisig balance (after confirmation):
```
$ dcrms --net=testnet3 getmultisigbalance address=Tcgeou3sAxjvyFecHcLDaQwJ59gTHZ53Fsi
100
```

Alice verifies that the multisig adress is functional (must be after funding):
```
$ dcrms --net=testnet3 multisiginfo address=Tcgeou3sAxjvyFecHcLDaQwJ59gTHZ53Fsi
Address      : Tcgeou3sAxjvyFecHcLDaQwJ59gTHZ53Fsi
M            : 2
N            : 2
Public key   : TkQ4HxtQMvWQ8VypqyR6nj4xxwxEU667ceEH2EfyEZajhCLZFwkwx
Public key   : TkKnHcV4PEd3Cz9NvAV9hCAqAiW2zyWBsgMvEcou78KtrNKBKmJU7
Redeem script: 5221037a0f5e5616bf12e395d3cef6d65ea38bf3bc570fc2b353a169ac30bda84f63932102e4a131b2f1dce7177b18a97265d99b91aed29cbdbac9142e2539dcd7fb01429852ae
```

Alice sends 5 DCR to Bob (TsUjJFG66ycJ5qVrBTRGVpgqqJNrgN4jBUc)
```
$ dcrms --net=testnet3 createmultisigtx address=Tcgeou3sAxjvyFecHcLDaQwJ59gTHZ53Fsi to=TsUjJFG66ycJ5qVrBTRGVpgqqJNrgN4jBUc amount=5 confirmations=2
01000000017f148b6b5a7802e38a7368d4fc6f1fae88344f881f4901c083686a111abc8ffd0100000000ffffffff020065cd1d0000000000001976a91428b19684953cc68d4e1c9367e70d937bfad6d0b988acf06b3e3602000000000017a914643c566846dab8ba0e28220fe3b5fd06b472ec518700000000000000000100e40b540200000000000000ffffffff475221037a0f5e5616bf12e395d3cef6d65ea38bf3bc570fc2b353a169ac30bda84f63932102e4a131b2f1dce7177b18a97265d99b91aed29cbdbac9142e2539dcd7fb01429852ae
```

Alice signs transaction (wallet figures out privkey):
```
$ dcrms --net=testnet3 signmultisigtx tx=01000000017f148b6b5a7802e38a7368d4fc6f1fae88344f881f4901c083686a111abc8ffd0100000000ffffffff020065cd1d0000000000001976a91428b19684953cc68d4e1c9367e70d937bfad6d0b988acf06b3e3602000000000017a914643c566846dab8ba0e28220fe3b5fd06b472ec518700000000000000000100e40b540200000000000000ffffffff475221037a0f5e5616bf12e395d3cef6d65ea38bf3bc570fc2b353a169ac30bda84f63932102e4a131b2f1dce7177b18a97265d99b91aed29cbdbac9142e2539dcd7fb01429852ae
TRANSACTION SIGNING *NOT* COMPLETE
01000000017f148b6b5a7802e38a7368d4fc6f1fae88344f881f4901c083686a111abc8ffd0100000000ffffffff020065cd1d0000000000001976a91428b19684953cc68d4e1c9367e70d937bfad6d0b988acf06b3e3602000000000017a914643c566846dab8ba0e28220fe3b5fd06b472ec518700000000000000000100e40b540200000000000000ffffffff91473044022017e94f0f74581b2e1fd0759fb10ee11a5d2c64d8df90989ab4b65f61015a018a02204b6381e82a5b6237b22beec5541f95ef0be259c5e96d767efc173b6ac43aa2d30100475221037a0f5e5616bf12e395d3cef6d65ea38bf3bc570fc2b353a169ac30bda84f63932102e4a131b2f1dce7177b18a97265d99b91aed29cbdbac9142e2539dcd7fb01429852ae
```

Bob signs transaction (wallet figures out privkey):
```
$ dcrms --net=testnet3 signmultisigtx tx=01000000017f148b6b5a7802e38a7368d4fc6f1fae88344f881f4901c083686a111abc8ffd0100000000ffffffff020065cd1d0000000000001976a91428b19684953cc68d4e1c9367e70d937bfad6d0b988acf06b3e3602000000000017a914643c566846dab8ba0e28220fe3b5fd06b472ec518700000000000000000100e40b540200000000000000ffffffff91473044022017e94f0f74581b2e1fd0759fb10ee11a5d2c64d8df90989ab4b65f61015a018a02204b6381e82a5b6237b22beec5541f95ef0be259c5e96d767efc173b6ac43aa2d30100475221037a0f5e5616bf12e395d3cef6d65ea38bf3bc570fc2b353a169ac30bda84f63932102e4a131b2f1dce7177b18a97265d99b91aed29cbdbac9142e2539dcd7fb01429852ae
TRANSACTION SIGNING *NOT* COMPLETE
01000000017f148b6b5a7802e38a7368d4fc6f1fae88344f881f4901c083686a111abc8ffd0100000000ffffffff020065cd1d0000000000001976a91428b19684953cc68d4e1c9367e70d937bfad6d0b988acf06b3e3602000000000017a914643c566846dab8ba0e28220fe3b5fd06b472ec518700000000000000000100e40b540200000000000000ffffffff91473044022017e94f0f74581b2e1fd0759fb10ee11a5d2c64d8df90989ab4b65f61015a018a02204b6381e82a5b6237b22beec5541f95ef0be259c5e96d767efc173b6ac43aa2d30100475221037a0f5e5616bf12e395d3cef6d65ea38bf3bc570fc2b353a169ac30bda84f63932102e4a131b2f1dce7177b18a97265d99b91aed29cbdbac9142e2539dcd7fb01429852ae
```

^^^---- Bug right here, should have completed signing.

Broadcast signed transaction to network:
```
$ dcrms --net=testnet3 broadcastmultisigtx tx=
```

## Todo

* Make utxo picker add some guestimate of fees in case the amounts add up exactly
* implement sweep
