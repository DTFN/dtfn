'use strict';

let chai = require('chai');
let assert = chai.assert;
let fs = require('fs');
let config = require('config');
let solc = require('solc');
let Web3 = require('web3');
let web3 = new Web3(new Web3.providers.HttpProvider("http://" + config.host + ":" + config.port));

const account = web3.eth.accounts[0];

console.log('Block number: ' + web3.eth.blockNumber)
console.log('Account: ' + account)
console.log('Account balance: ' + web3.eth.getBalance(account))

//unlock account
web3.personal.unlockAccount(account, "1234");
let contractSource = fs.readFileSync(__dirname + '/test.sol', 'utf-8');

console.log("Solc version is " + solc.version());

describe('gasLimit', function () {
    it('should return gas too low error', function (done) {
        // mocha timeout
        this.timeout(120 * 1000);

        const contractCompiled = solc.compile(contractSource);

        const bytecode = contractCompiled.contracts[':Test'].bytecode;
        const abi = JSON.parse(contractCompiled.contracts[':Test'].interface);

        // const estimateGas = web3.eth.estimateGas({data: '0x' + bytecode,});
        // console.log('Gas needed: ' + estimateGas);

        const testContract = web3.eth.contract(abi);
        testContract.new({
            from: account,
            data: '0x' + bytecode,
            gas: '100' //set low gas
        }, function (error) {
            console.log(error);
            assert.equal(error, "Error: intrinsic gas too low");
            done();
        });
    });

    it('should send tx', function (done) {
        this.timeout(60 * 1000);
        web3.eth.sendTransaction({
            from: web3.eth.accounts[0],
            to: "0x231dD21555C6D905ce4f2AafDBa0C01aF89Db0a0",
            value: "11123123"
        }, function (err, transactionHash) {
            if (!err){
                console.log(transactionHash + " success");
                console.log(web3.eth.getBalance("0x231dD21555C6D905ce4f2AafDBa0C01aF89Db0a0"))
                done()
	    } else console.log(err)
        });
    });
});

