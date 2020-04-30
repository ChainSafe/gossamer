// Copyright 2019 ChainSafe Systems (ON) Corp.
// This file is part of gossamer.
//
// The gossamer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The gossamer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the gossamer library. If not, see <http://www.gnu.org/licenses/>.

package modules

import (
	"encoding/hex"
	"net/http"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/runtime"
)

// StateCallRequest holds json fields
type StateCallRequest struct {
	Method string      `json:"method"`
	Data   []byte      `json:"data"`
	Block  common.Hash `json:"block"`
}

// StateChildStorageRequest holds json fields
type StateChildStorageRequest struct {
	ChildStorageKey []byte      `json:"childStorageKey"`
	Key             []byte      `json:"key"`
	Block           common.Hash `json:"block"`
}

// StateStorageKeyRequest holds json fields
type StateStorageKeyRequest struct {
	Key   []byte      `json:"key"`
	Block common.Hash `json:"block"`
}

// StateStorageQueryRequest holds json fields
type StateStorageQueryRequest struct {
	Key   []byte      `json:"key"`
	Block common.Hash `json:"block"`
}

// StateBlockHashQuery is a hash value
type StateBlockHashQuery common.Hash

// StateRuntimeMetadataQuery is a hash value
type StateRuntimeMetadataQuery common.Hash

// StateStorageQueryRangeRequest holds json fields
type StateStorageQueryRangeRequest struct {
	Keys       []byte      `json:"keys"`
	StartBlock common.Hash `json:"startBlock"`
	Block      common.Hash `json:"block"`
}

// StateStorageKeysQuery field to store storage keys
type StateStorageKeysQuery [][]byte

// StateCallResponse holds json fields
type StateCallResponse struct {
	StateCallResponse []byte `json:"stateCallResponse"`
}

// StateKeysResponse field to store the state keys
type StateKeysResponse [][]byte

// StateStorageDataResponse field to store data response
type StateStorageDataResponse []byte

// StateStorageHashResponse is a hash value
type StateStorageHashResponse common.Hash

// StateStorageSizeResponse the default size for response
type StateStorageSizeResponse uint64

// StateStorageKeysResponse field for storage keys
type StateStorageKeysResponse [][]byte

// StateMetadataResponse holds the metadata
//TODO: Determine actual type
type StateMetadataResponse string

// StorageChangeSetResponse is the struct that holds the block and changes
type StorageChangeSetResponse struct {
	Block   common.Hash
	Changes []KeyValueOption
}

// KeyValueOption struct holds json fields
type KeyValueOption struct {
	StorageKey  []byte `json:"storageKey"`
	StorageData []byte `json:"storageData"`
}

// StorageKey is the key for the storage
type StorageKey []byte

// StateRuntimeVersionResponse is the runtime version response
type StateRuntimeVersionResponse struct {
	SpecName         string        `json:"specName"`
	ImplName         string        `json:"implName"`
	AuthoringVersion int32         `json:"authoringVersion"`
	SpecVersion      int32         `json:"specVersion"`
	ImplVersion      int32         `json:"implVersion"`
	Apis             []interface{} `json:"apis"`
}

// StateModule is an RPC module providing access to storage API points.
type StateModule struct {
	networkAPI NetworkAPI
	storageAPI StorageAPI
	coreAPI    CoreAPI
}

// NewStateModule creates a new State module.
func NewStateModule(net NetworkAPI, storage StorageAPI, core CoreAPI) *StateModule {
	return &StateModule{
		networkAPI: net,
		storageAPI: storage,
		coreAPI:    core,
	}
}

// Call isn't implemented properly yet.
func (sm *StateModule) Call(r *http.Request, req *StateCallRequest, res *StateCallResponse) {
	_ = sm.networkAPI
	_ = sm.storageAPI
}

// GetChildKeys isn't implemented properly yet.
func (sm *StateModule) GetChildKeys(r *http.Request, req *StateChildStorageRequest, res *StateKeysResponse) {
}

// GetChildStorage isn't implemented properly yet.
func (sm *StateModule) GetChildStorage(r *http.Request, req *StateChildStorageRequest, res *StateStorageDataResponse) {
}

// GetChildStorageHash isn't implemented properly yet.
func (sm *StateModule) GetChildStorageHash(r *http.Request, req *StateChildStorageRequest, res *StateStorageHashResponse) {
}

// GetChildStorageSize isn't implemented properly yet.
func (sm *StateModule) GetChildStorageSize(r *http.Request, req *StateChildStorageRequest, res *StateStorageSizeResponse) {
}

// GetKeys isn't implemented properly yet.
func (sm *StateModule) GetKeys(r *http.Request, req *StateStorageKeyRequest, res *StateStorageKeysResponse) {
}

// GetMetadata isn't implemented properly yet.
func (sm *StateModule) GetMetadata(r *http.Request, req *StateRuntimeMetadataQuery, res *StateMetadataResponse) error {
	// TODO implement call to runtime for getMetadata
	*res = "0x6d6574610b241853797374656d011853797374656d341c4163636f756e7401010130543a3a4163636f756e744964944163636f756e74496e666f3c543a3a496e6465782c20543a3a4163636f756e74446174613e00150100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000004e8205468652066756c6c206163636f756e7420696e666f726d6174696f6e20666f72206120706172746963756c6172206163636f756e742049442e3845787472696e736963436f756e7400000c753332040004b820546f74616c2065787472696e7369637320636f756e7420666f72207468652063757272656e7420626c6f636b2e4c416c6c45787472696e73696373576569676874000018576569676874040004150120546f74616c2077656967687420666f7220616c6c2065787472696e736963732070757420746f6765746865722c20666f72207468652063757272656e7420626c6f636b2e40416c6c45787472696e736963734c656e00000c753332040004410120546f74616c206c656e6774682028696e2062797465732920666f7220616c6c2065787472696e736963732070757420746f6765746865722c20666f72207468652063757272656e7420626c6f636b2e24426c6f636b4861736801010138543a3a426c6f636b4e756d6265721c543a3a48617368008000000000000000000000000000000000000000000000000000000000000000000498204d6170206f6620626c6f636b206e756d6265727320746f20626c6f636b206861736865732e3445787472696e736963446174610101050c7533321c5665633c75383e000400043d012045787472696e73696373206461746120666f72207468652063757272656e7420626c6f636b20286d61707320616e2065787472696e736963277320696e64657820746f206974732064617461292e184e756d626572010038543a3a426c6f636b4e756d6265721000000000040901205468652063757272656e7420626c6f636b206e756d626572206265696e672070726f6365737365642e205365742062792060657865637574655f626c6f636b602e28506172656e744861736801001c543a3a4861736880000000000000000000000000000000000000000000000000000000000000000004702048617368206f66207468652070726576696f757320626c6f636b2e3845787472696e73696373526f6f7401001c543a3a486173688000000000000000000000000000000000000000000000000000000000000000000415012045787472696e7369637320726f6f74206f66207468652063757272656e7420626c6f636b2c20616c736f2070617274206f662074686520626c6f636b206865616465722e1844696765737401002c4469676573744f663c543e040004f020446967657374206f66207468652063757272656e7420626c6f636b2c20616c736f2070617274206f662074686520626c6f636b206865616465722e184576656e747301008c5665633c4576656e745265636f72643c543a3a4576656e742c20543a3a486173683e3e040004a0204576656e7473206465706f736974656420666f72207468652063757272656e7420626c6f636b2e284576656e74436f756e740100284576656e74496e646578100000000004b820546865206e756d626572206f66206576656e747320696e2074686520604576656e74733c543e60206c6973742e2c4576656e74546f706963730101011c543a3a48617368845665633c28543a3a426c6f636b4e756d6265722c204576656e74496e646578293e000400282501204d617070696e67206265747765656e206120746f7069632028726570726573656e74656420627920543a3a486173682920616e64206120766563746f72206f6620696e646578657394206f66206576656e747320696e2074686520603c4576656e74733c543e3e60206c6973742e00510120416c6c20746f70696320766563746f727320686176652064657465726d696e69737469632073746f72616765206c6f636174696f6e7320646570656e64696e67206f6e2074686520746f7069632e2054686973450120616c6c6f7773206c696768742d636c69656e747320746f206c6576657261676520746865206368616e67657320747269652073746f7261676520747261636b696e67206d656368616e69736d20616e64e420696e2063617365206f66206368616e67657320666574636820746865206c697374206f66206576656e7473206f6620696e7465726573742e004d01205468652076616c756520686173207468652074797065206028543a3a426c6f636b4e756d6265722c204576656e74496e646578296020626563617573652069662077652075736564206f6e6c79206a7573744d012074686520604576656e74496e64657860207468656e20696e20636173652069662074686520746f70696320686173207468652073616d6520636f6e74656e7473206f6e20746865206e65787420626c6f636b0101206e6f206e6f74696669636174696f6e2077696c6c20626520747269676765726564207468757320746865206576656e74206d69676874206265206c6f73742e01282866696c6c5f626c6f636b04185f726174696f1c50657262696c6c040901204120646973706174636820746861742077696c6c2066696c6c2074686520626c6f636b2077656967687420757020746f2074686520676976656e20726174696f2e1872656d61726b041c5f72656d61726b1c5665633c75383e046c204d616b6520736f6d65206f6e2d636861696e2072656d61726b2e387365745f686561705f7061676573041470616765730c75363404fc2053657420746865206e756d626572206f6620706167657320696e2074686520576562417373656d626c7920656e7669726f6e6d656e74277320686561702e207365745f636f64650410636f64651c5665633c75383e04682053657420746865206e65772072756e74696d6520636f64652e5c7365745f636f64655f776974686f75745f636865636b730410636f64651c5665633c75383e041d012053657420746865206e65772072756e74696d6520636f646520776974686f757420646f696e6720616e7920636865636b73206f662074686520676976656e2060636f6465602e5c7365745f6368616e6765735f747269655f636f6e666967044c6368616e6765735f747269655f636f6e666967804f7074696f6e3c4368616e67657354726965436f6e66696775726174696f6e3e04a02053657420746865206e6577206368616e676573207472696520636f6e66696775726174696f6e2e2c7365745f73746f7261676504146974656d73345665633c4b657956616c75653e046c2053657420736f6d65206974656d73206f662073746f726167652e306b696c6c5f73746f7261676504106b657973205665633c4b65793e0478204b696c6c20736f6d65206974656d732066726f6d2073746f726167652e2c6b696c6c5f70726566697804187072656669780c4b6579041501204b696c6c20616c6c2073746f72616765206974656d7320776974682061206b657920746861742073746172747320776974682074686520676976656e207072656669782e1c7375696369646500086501204b696c6c207468652073656e64696e67206163636f756e742c20617373756d696e6720746865726520617265206e6f207265666572656e636573206f75747374616e64696e6720616e642074686520636f6d706f7369746590206461746120697320657175616c20746f206974732064656661756c742076616c75652e01144045787472696e7369635375636365737304304469737061746368496e666f049420416e2065787472696e73696320636f6d706c65746564207375636365737366756c6c792e3c45787472696e7369634661696c6564083444697370617463684572726f72304469737061746368496e666f045420416e2065787472696e736963206661696c65642e2c436f64655570646174656400045420603a636f6465602077617320757064617465642e284e65774163636f756e7404244163636f756e744964046c2041206e6577206163636f756e742077617320637265617465642e344b696c6c65644163636f756e7404244163636f756e744964045c20416e206163636f756e7420776173207265617065642e001c3c496e76616c6964537065634e616d6508150120546865206e616d65206f662073706563696669636174696f6e20646f6573206e6f74206d61746368206265747765656e207468652063757272656e742072756e74696d655420616e6420746865206e65772072756e74696d652e7c5370656356657273696f6e4e6f74416c6c6f776564546f4465637265617365084501205468652073706563696669636174696f6e2076657273696f6e206973206e6f7420616c6c6f77656420746f206465637265617365206265747765656e207468652063757272656e742072756e74696d655420616e6420746865206e65772072756e74696d652e7c496d706c56657273696f6e4e6f74416c6c6f776564546f44656372656173650849012054686520696d706c656d656e746174696f6e2076657273696f6e206973206e6f7420616c6c6f77656420746f206465637265617365206265747765656e207468652063757272656e742072756e74696d655420616e6420746865206e65772072756e74696d652e7c537065634f72496d706c56657273696f6e4e656564546f496e637265617365083501205468652073706563696669636174696f6e206f722074686520696d706c656d656e746174696f6e2076657273696f6e206e65656420746f20696e637265617365206265747765656e20746865942063757272656e742072756e74696d6520616e6420746865206e65772072756e74696d652e744661696c6564546f4578747261637452756e74696d6556657273696f6e0cf0204661696c656420746f2065787472616374207468652072756e74696d652076657273696f6e2066726f6d20746865206e65772072756e74696d652e000d01204569746865722063616c6c696e672060436f72655f76657273696f6e60206f72206465636f64696e67206052756e74696d6556657273696f6e60206661696c65642e4c4e6f6e44656661756c74436f6d706f7369746504010120537569636964652063616c6c6564207768656e20746865206163636f756e7420686173206e6f6e2d64656661756c7420636f6d706f7369746520646174612e3c4e6f6e5a65726f526566436f756e740439012054686572652069732061206e6f6e2d7a65726f207265666572656e636520636f756e742070726576656e74696e6720746865206163636f756e742066726f6d206265696e67207075726765642e6052616e646f6d6e657373436f6c6c656374697665466c6970016052616e646f6d6e657373436f6c6c656374697665466c6970043852616e646f6d4d6174657269616c0100305665633c543a3a486173683e04000c610120536572696573206f6620626c6f636b20686561646572732066726f6d20746865206c61737420383120626c6f636b73207468617420616374732061732072616e646f6d2073656564206d6174657269616c2e2054686973610120697320617272616e67656420617320612072696e672062756666657220776974682060626c6f636b5f6e756d626572202520383160206265696e672074686520696e64657820696e746f20746865206056656360206f664420746865206f6c6465737420686173682e01000000002454696d657374616d70012454696d657374616d70080c4e6f77010024543a3a4d6f6d656e7420000000000000000004902043757272656e742074696d6520666f72207468652063757272656e7420626c6f636b2e24446964557064617465010010626f6f6c040004b420446964207468652074696d657374616d7020676574207570646174656420696e207468697320626c6f636b3f01040c736574040c6e6f7748436f6d706163743c543a3a4d6f6d656e743e245820536574207468652063757272656e742074696d652e00590120546869732063616c6c2073686f756c6420626520696e766f6b65642065786163746c79206f6e63652070657220626c6f636b2e2049742077696c6c2070616e6963206174207468652066696e616c697a6174696f6ed82070686173652c20696620746869732063616c6c206861736e2774206265656e20696e766f6b656420627920746861742074696d652e004501205468652074696d657374616d702073686f756c642062652067726561746572207468616e207468652070726576696f7573206f6e652062792074686520616d6f756e74207370656369666965642062794420604d696e696d756d506572696f64602e00d820546865206469737061746368206f726967696e20666f7220746869732063616c6c206d7573742062652060496e686572656e74602e0004344d696e696d756d506572696f6424543a3a4d6f6d656e7420b80b00000000000010690120546865206d696e696d756d20706572696f64206265747765656e20626c6f636b732e204265776172652074686174207468697320697320646966666572656e7420746f20746865202a65787065637465642a20706572696f64690120746861742074686520626c6f636b2070726f64756374696f6e206170706172617475732070726f76696465732e20596f75722063686f73656e20636f6e73656e7375732073797374656d2077696c6c2067656e6572616c6c79650120776f726b2077697468207468697320746f2064657465726d696e6520612073656e7369626c6520626c6f636b2074696d652e20652e672e20466f7220417572612c2069742077696c6c20626520646f75626c6520746869737020706572696f64206f6e2064656661756c742073657474696e67732e00104175726100000000001c4772616e647061013c4772616e64706146696e616c6974791c2c417574686f726974696573010034417574686f726974794c6973740400102c20444550524543415445440061012054686973207573656420746f2073746f7265207468652063757272656e7420617574686f72697479207365742c20776869636820686173206265656e206d6967726174656420746f207468652077656c6c2d6b6e6f776e98204752414e4450415f415554484f5249544945535f4b455920756e686173686564206b65792e14537461746501006c53746f72656453746174653c543a3a426c6f636b4e756d6265723e04000490205374617465206f66207468652063757272656e7420617574686f72697479207365742e3450656e64696e674368616e676500008c53746f72656450656e64696e674368616e67653c543a3a426c6f636b4e756d6265723e040004c42050656e64696e67206368616e67653a20287369676e616c65642061742c207363686564756c6564206368616e6765292e284e657874466f72636564000038543a3a426c6f636b4e756d626572040004bc206e65787420626c6f636b206e756d6265722077686572652077652063616e20666f7263652061206368616e67652e1c5374616c6c656400008028543a3a426c6f636b4e756d6265722c20543a3a426c6f636b4e756d626572290400049020607472756560206966207765206172652063757272656e746c79207374616c6c65642e3043757272656e7453657449640100145365744964200000000000000000085d0120546865206e756d626572206f66206368616e6765732028626f746820696e207465726d73206f66206b65797320616e6420756e6465726c79696e672065636f6e6f6d696320726573706f6e736962696c697469657329c420696e20746865202273657422206f66204772616e6470612076616c696461746f72732066726f6d2067656e657369732e30536574496453657373696f6e0001011453657449643053657373696f6e496e64657800040004c1012041206d617070696e672066726f6d206772616e6470612073657420494420746f2074686520696e646578206f6620746865202a6d6f737420726563656e742a2073657373696f6e20666f7220776869636820697473206d656d62657273207765726520726573706f6e7369626c652e0104487265706f72745f6d69736265686176696f72041c5f7265706f72741c5665633c75383e0464205265706f727420736f6d65206d69736265686176696f722e010c384e6577417574686f7269746965730434417574686f726974794c6973740490204e657720617574686f726974792073657420686173206265656e206170706c6965642e1850617573656400049c2043757272656e7420617574686f726974792073657420686173206265656e207061757365642e1c526573756d65640004a02043757272656e7420617574686f726974792073657420686173206265656e20726573756d65642e00102c50617573654661696c656408090120417474656d707420746f207369676e616c204752414e445041207061757365207768656e2074686520617574686f72697479207365742069736e2774206c697665a8202865697468657220706175736564206f7220616c72656164792070656e64696e67207061757365292e30526573756d654661696c656408150120417474656d707420746f207369676e616c204752414e44504120726573756d65207768656e2074686520617574686f72697479207365742069736e277420706175736564a42028656974686572206c697665206f7220616c72656164792070656e64696e6720726573756d65292e344368616e676550656e64696e6704ec20417474656d707420746f207369676e616c204752414e445041206368616e67652077697468206f6e6520616c72656164792070656e64696e672e1c546f6f536f6f6e04c02043616e6e6f74207369676e616c20666f72636564206368616e676520736f20736f6f6e206166746572206c6173742e2042616c616e636573012042616c616e6365731034546f74616c49737375616e6365010028543a3a42616c616e6365400000000000000000000000000000000004982054686520746f74616c20756e6974732069737375656420696e207468652073797374656d2e1c4163636f756e7401010130543a3a4163636f756e7449645c4163636f756e74446174613c543a3a42616c616e63653e00010100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000186c205468652062616c616e6365206f6620616e206163636f756e742e005901204e4f54453a2054484953204d4159204e4556455220424520494e204558495354454e434520414e4420594554204841564520412060746f74616c28292e69735f7a65726f2829602e2049662074686520746f74616cc02069732065766572207a65726f2c207468656e2074686520656e747279202a4d5553542a2062652072656d6f7665642e004101204e4f54453a2054686973206973206f6e6c79207573656420696e20746865206361736520746861742074686973206d6f64756c65206973207573656420746f2073746f72652062616c616e6365732e144c6f636b7301010130543a3a4163636f756e744964705665633c42616c616e63654c6f636b3c543a3a42616c616e63653e3e00040008b820416e79206c6971756964697479206c6f636b73206f6e20736f6d65206163636f756e742062616c616e6365732e2501204e4f54453a2053686f756c64206f6e6c79206265206163636573736564207768656e2073657474696e672c206368616e67696e6720616e642066726565696e672061206c6f636b2e2849735570677261646564010010626f6f6c04000ccc2054727565206966206e6574776f726b20686173206265656e20757067726164656420746f20746869732076657273696f6e2e005c205472756520666f72206e6577206e6574776f726b732e0110207472616e736665720810646573748c3c543a3a4c6f6f6b7570206173205374617469634c6f6f6b75703e3a3a536f757263651476616c75654c436f6d706163743c543a3a42616c616e63653e60d8205472616e7366657220736f6d65206c697175696420667265652062616c616e636520746f20616e6f74686572206163636f756e742e00090120607472616e73666572602077696c6c207365742074686520604672656542616c616e636560206f66207468652073656e64657220616e642072656365697665722e21012049742077696c6c2064656372656173652074686520746f74616c2069737375616e6365206f66207468652073797374656d2062792074686520605472616e73666572466565602e1501204966207468652073656e6465722773206163636f756e742069732062656c6f7720746865206578697374656e7469616c206465706f736974206173206120726573756c74b4206f6620746865207472616e736665722c20746865206163636f756e742077696c6c206265207265617065642e00190120546865206469737061746368206f726967696e20666f7220746869732063616c6c206d75737420626520605369676e65646020627920746865207472616e736163746f722e002c2023203c7765696768743e3101202d20446570656e64656e74206f6e20617267756d656e747320627574206e6f7420637269746963616c2c20676976656e2070726f70657220696d706c656d656e746174696f6e7320666f72cc202020696e70757420636f6e6669672074797065732e205365652072656c617465642066756e6374696f6e732062656c6f772e6901202d20497420636f6e7461696e732061206c696d69746564206e756d626572206f6620726561647320616e642077726974657320696e7465726e616c6c7920616e64206e6f20636f6d706c657820636f6d7075746174696f6e2e004c2052656c617465642066756e6374696f6e733a0051012020202d2060656e737572655f63616e5f77697468647261776020697320616c776179732063616c6c656420696e7465726e616c6c792062757420686173206120626f756e64656420636f6d706c65786974792e2d012020202d205472616e7366657272696e672062616c616e63657320746f206163636f756e7473207468617420646964206e6f74206578697374206265666f72652077696c6c206361757365d420202020202060543a3a4f6e4e65774163636f756e743a3a6f6e5f6e65775f6163636f756e746020746f2062652063616c6c65642e61012020202d2052656d6f76696e6720656e6f7567682066756e64732066726f6d20616e206163636f756e742077696c6c20747269676765722060543a3a4475737452656d6f76616c3a3a6f6e5f756e62616c616e636564602e49012020202d20607472616e736665725f6b6565705f616c6976656020776f726b73207468652073616d652077617920617320607472616e73666572602c206275742068617320616e206164646974696f6e616cf82020202020636865636b207468617420746865207472616e736665722077696c6c206e6f74206b696c6c20746865206f726967696e206163636f756e742e00302023203c2f7765696768743e2c7365745f62616c616e63650c0c77686f8c3c543a3a4c6f6f6b7570206173205374617469634c6f6f6b75703e3a3a536f75726365206e65775f667265654c436f6d706163743c543a3a42616c616e63653e306e65775f72657365727665644c436f6d706163743c543a3a42616c616e63653e349420536574207468652062616c616e636573206f66206120676976656e206163636f756e742e00210120546869732077696c6c20616c74657220604672656542616c616e63656020616e642060526573657276656442616c616e63656020696e2073746f726167652e2069742077696c6c090120616c736f2064656372656173652074686520746f74616c2069737375616e6365206f66207468652073797374656d202860546f74616c49737375616e636560292e190120496620746865206e65772066726565206f722072657365727665642062616c616e63652069732062656c6f7720746865206578697374656e7469616c206465706f7369742c01012069742077696c6c20726573657420746865206163636f756e74206e6f6e63652028606672616d655f73797374656d3a3a4163636f756e744e6f6e636560292e00b420546865206469737061746368206f726967696e20666f7220746869732063616c6c2069732060726f6f74602e002c2023203c7765696768743e80202d20496e646570656e64656e74206f662074686520617267756d656e74732ec4202d20436f6e7461696e732061206c696d69746564206e756d626572206f6620726561647320616e64207772697465732e302023203c2f7765696768743e38666f7263655f7472616e736665720c18736f757263658c3c543a3a4c6f6f6b7570206173205374617469634c6f6f6b75703e3a3a536f7572636510646573748c3c543a3a4c6f6f6b7570206173205374617469634c6f6f6b75703e3a3a536f757263651476616c75654c436f6d706163743c543a3a42616c616e63653e0851012045786163746c7920617320607472616e73666572602c2065786365707420746865206f726967696e206d75737420626520726f6f7420616e642074686520736f75726365206163636f756e74206d61792062652c207370656369666965642e4c7472616e736665725f6b6565705f616c6976650810646573748c3c543a3a4c6f6f6b7570206173205374617469634c6f6f6b75703e3a3a536f757263651476616c75654c436f6d706163743c543a3a42616c616e63653e1851012053616d6520617320746865205b607472616e73666572605d2063616c6c2c206275742077697468206120636865636b207468617420746865207472616e736665722077696c6c206e6f74206b696c6c2074686540206f726967696e206163636f756e742e00bc20393925206f66207468652074696d6520796f752077616e74205b607472616e73666572605d20696e73746561642e00c4205b607472616e73666572605d3a207374727563742e4d6f64756c652e68746d6c236d6574686f642e7472616e7366657201141c456e646f77656408244163636f756e7449641c42616c616e636504bc20416e206163636f756e74207761732063726561746564207769746820736f6d6520667265652062616c616e63652e20447573744c6f737408244163636f756e7449641c42616c616e636508410120416e206163636f756e74207761732072656d6f7665642077686f73652062616c616e636520776173206e6f6e2d7a65726f206275742062656c6f77204578697374656e7469616c4465706f7369742c7c20726573756c74696e6720696e20616e206f75747269676874206c6f73732e205472616e736665720c244163636f756e744964244163636f756e7449641c42616c616e63650498205472616e7366657220737563636565646564202866726f6d2c20746f2c2076616c7565292e2842616c616e63655365740c244163636f756e7449641c42616c616e63651c42616c616e636504c420412062616c616e6365207761732073657420627920726f6f74202877686f2c20667265652c207265736572766564292e1c4465706f73697408244163636f756e7449641c42616c616e636504dc20536f6d6520616d6f756e7420776173206465706f73697465642028652e672e20666f72207472616e73616374696f6e2066656573292e04484578697374656e7469616c4465706f73697428543a3a42616c616e636540f401000000000000000000000000000004d420546865206d696e696d756d20616d6f756e7420726571756972656420746f206b65657020616e206163636f756e74206f70656e2e203856657374696e6742616c616e6365049c2056657374696e672062616c616e636520746f6f206869676820746f2073656e642076616c7565544c69717569646974795265737472696374696f6e7304c8204163636f756e74206c6971756964697479207265737472696374696f6e732070726576656e74207769746864726177616c204f766572666c6f77047420476f7420616e206f766572666c6f7720616674657220616464696e674c496e73756666696369656e7442616c616e636504782042616c616e636520746f6f206c6f7720746f2073656e642076616c7565484578697374656e7469616c4465706f73697404ec2056616c756520746f6f206c6f7720746f20637265617465206163636f756e742064756520746f206578697374656e7469616c206465706f736974244b656570416c6976650490205472616e736665722f7061796d656e7420776f756c64206b696c6c206163636f756e745c4578697374696e6756657374696e675363686564756c6504cc20412076657374696e67207363686564756c6520616c72656164792065786973747320666f722074686973206163636f756e742c446561644163636f756e74048c2042656e6566696369617279206163636f756e74206d757374207072652d6578697374485472616e73616374696f6e5061796d656e74012042616c616e63657304444e6578744665654d756c7469706c6965720100284d756c7469706c69657220000000000000000000000008485472616e73616374696f6e426173654665653042616c616e63654f663c543e400000000000000000000000000000000004dc205468652066656520746f206265207061696420666f72206d616b696e672061207472616e73616374696f6e3b2074686520626173652e485472616e73616374696f6e427974654665653042616c616e63654f663c543e4001000000000000000000000000000000040d01205468652066656520746f206265207061696420666f72206d616b696e672061207472616e73616374696f6e3b20746865207065722d6279746520706f7274696f6e2e00105375646f01105375646f040c4b6579010030543a3a4163636f756e74496480000000000000000000000000000000000000000000000000000000000000000004842054686520604163636f756e74496460206f6620746865207375646f206b65792e010c107375646f041063616c6c5c426f783c3c542061732054726169743e3a3a43616c6c3e2839012041757468656e7469636174657320746865207375646f206b657920616e64206469737061746368657320612066756e6374696f6e2063616c6c20776974682060526f6f7460206f726967696e2e00d020546865206469737061746368206f726967696e20666f7220746869732063616c6c206d757374206265205f5369676e65645f2e002c2023203c7765696768743e20202d204f2831292e64202d204c696d697465642073746f726167652072656164732e60202d204f6e6520444220777269746520286576656e74292ec8202d20576569676874206f662064657269766174697665206063616c6c6020657865637574696f6e202b2031302c3030302e302023203c2f7765696768743e1c7365745f6b6579040c6e65778c3c543a3a4c6f6f6b7570206173205374617469634c6f6f6b75703e3a3a536f757263652475012041757468656e74696361746573207468652063757272656e74207375646f206b657920616e6420736574732074686520676976656e204163636f756e7449642028606e6577602920617320746865206e6577207375646f206b65792e00d020546865206469737061746368206f726967696e20666f7220746869732063616c6c206d757374206265205f5369676e65645f2e002c2023203c7765696768743e20202d204f2831292e64202d204c696d697465642073746f726167652072656164732e44202d204f6e65204442206368616e67652e302023203c2f7765696768743e1c7375646f5f6173080c77686f8c3c543a3a4c6f6f6b7570206173205374617469634c6f6f6b75703e3a3a536f757263651063616c6c5c426f783c3c542061732054726169743e3a3a43616c6c3e2c51012041757468656e7469636174657320746865207375646f206b657920616e64206469737061746368657320612066756e6374696f6e2063616c6c207769746820605369676e656460206f726967696e2066726f6d44206120676976656e206163636f756e742e00d020546865206469737061746368206f726967696e20666f7220746869732063616c6c206d757374206265205f5369676e65645f2e002c2023203c7765696768743e20202d204f2831292e64202d204c696d697465642073746f726167652072656164732e60202d204f6e6520444220777269746520286576656e74292ec8202d20576569676874206f662064657269766174697665206063616c6c6020657865637574696f6e202b2031302c3030302e302023203c2f7765696768743e010c1453756469640410626f6f6c04602041207375646f206a75737420746f6f6b20706c6163652e284b65794368616e67656404244163636f756e74496404f020546865207375646f6572206a757374207377697463686564206964656e746974793b20746865206f6c64206b657920697320737570706c6965642e285375646f4173446f6e650410626f6f6c04602041207375646f206a75737420746f6f6b20706c6163652e00042c526571756972655375646f04802053656e646572206d75737420626520746865205375646f206163636f756e743854656d706c6174654d6f64756c65013854656d706c6174654d6f64756c650424536f6d657468696e6700000c753332040000010830646f5f736f6d657468696e670424736f6d657468696e670c7533320c68204a75737420612064756d6d7920656e74727920706f696e742e21012066756e6374696f6e20746861742063616e2062652063616c6c6564206279207468652065787465726e616c20776f726c6420617320616e2065787472696e736963732063616c6c25012074616b6573206120706172616d65746572206f6620746865207479706520604163636f756e744964602c2073746f7265732069742c20616e6420656d69747320616e206576656e742c63617573655f6572726f7200086c20416e6f746865722064756d6d7920656e74727920706f696e742e5d012074616b6573206e6f20706172616d65746572732c20617474656d70747320746f20696e6372656d656e742073746f726167652076616c75652c20616e6420706f737369626c79207468726f777320616e206572726f7201043c536f6d657468696e6753746f726564080c753332244163636f756e7449640c50204a75737420612064756d6d79206576656e742e4501204576656e742060536f6d657468696e6760206973206465636c617265642077697468206120706172616d65746572206f6620746865207479706520607533326020616e6420604163636f756e74496460350120546f20656d69742074686973206576656e742c2077652063616c6c20746865206465706f7369742066756e6374696f6e2c2066726f6d206f75722072756e74696d652066756e6374696f6e730008244e6f6e6556616c7565043c2056616c756520776173204e6f6e653c53746f726167654f766572666c6f7704e02056616c75652072656163686564206d6178696d756d20616e642063616e6e6f7420626520696e6372656d656e7465642066757274686572041830436865636b56657273696f6e30436865636b47656e6573697320436865636b45726128436865636b4e6f6e63652c436865636b576569676874604368617267655472616e73616374696f6e5061796d656e74"
	return nil
}

// GetRuntimeVersion Get the runtime version at a given block.
//  If no block hash is provided, the latest version gets returned.
// TODO currently only returns latest version, add functionality to lookup runtime by block hash
func (sm *StateModule) GetRuntimeVersion(r *http.Request, req *StateBlockHashQuery, res *StateRuntimeVersionResponse) error {
	rtVersion, err := sm.coreAPI.GetRuntimeVersion()
	res.SpecName = string(rtVersion.RuntimeVersion.Spec_name)
	res.ImplName = string(rtVersion.RuntimeVersion.Impl_name)
	res.AuthoringVersion = rtVersion.RuntimeVersion.Authoring_version
	res.SpecVersion = rtVersion.RuntimeVersion.Spec_version
	res.ImplVersion = rtVersion.RuntimeVersion.Impl_version
	res.Apis = convertAPIs(rtVersion.API)

	return err
}

// GetStorage isn't implemented properly yet.
func (sm *StateModule) GetStorage(r *http.Request, req *StateStorageQueryRequest, res *StateStorageDataResponse) {
}

// GetStorageHash isn't implemented properly yet.
func (sm *StateModule) GetStorageHash(r *http.Request, req *StateStorageQueryRequest, res *StateStorageHashResponse) {
}

// GetStorageSize isn't implemented properly yet.
func (sm *StateModule) GetStorageSize(r *http.Request, req *StateStorageQueryRequest, res *StateStorageSizeResponse) {
}

// QueryStorage isn't implemented properly yet.
func (sm *StateModule) QueryStorage(r *http.Request, req *StateStorageQueryRangeRequest, res *StorageChangeSetResponse) {
}

// SubscribeRuntimeVersion isn't implemented properly yet.
// TODO make this actually a subscription that pushes data
func (sm *StateModule) SubscribeRuntimeVersion(r *http.Request, req *StateStorageQueryRangeRequest, res *StateRuntimeVersionResponse) error {
	return sm.GetRuntimeVersion(r, nil, res)
}

// SubscribeStorage isn't implemented properly yet.
func (sm *StateModule) SubscribeStorage(r *http.Request, req *StateStorageQueryRangeRequest, res *StorageChangeSetResponse) error {
	return nil
}

func convertAPIs(in []*runtime.API_Item) []interface{} {
	ret := make([]interface{}, 0)
	for _, item := range in {
		encStr := hex.EncodeToString(item.Name)
		ret = append(ret, []interface{}{"0x" + encStr, item.Ver})
	}
	return ret
}
