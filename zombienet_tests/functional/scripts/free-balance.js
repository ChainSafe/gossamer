async function run(nodeName, networkInfo, args) {
    const {wsUri, userDefinedTypes} = networkInfo.nodesByName[nodeName];
    const api = await zombie.connect(wsUri, userDefinedTypes);

    const {nonce, data: balance} = await api.query.system.account(args[0]);

    return balance.free;
}

module.exports = { run }