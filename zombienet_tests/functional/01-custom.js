async function run(nodeName, networkInfo, args) {
    const {wsUri, userDefinedTypes} = networkInfo.nodesByName[nodeName];
    const api = await zombie.connect(wsUri, userDefinedTypes);
    console.log("user types " + JSON.stringify(userDefinedTypes))
    console.log("babe " + api.consts.babe.epochDuration.toNumber());
    const now = await api.query.timestamp.now();
    console.log("now " + now)
    console.log("wsUri " + wsUri)

    console.log("node metadata " + JSON.stringify(api.rpc.state.getMetadata()))
    const chain = await api.rpc.system.chain();
    console.log("chain " + chain);
    const lastHeader = await api.rpc.chain.getHeader();
    console.log("header " + lastHeader)
    // console.log("node api " + JSON.stringify(api))
    const version = await api.rpc.system.version();
    console.log("node version " + version)
    return 2;
}

module.exports = { run }