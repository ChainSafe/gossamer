// Copyright 2017-2022 @polkadot/types-create authors & contributors
// SPDX-License-Identifier: Apache-2.0
export let TypeDefInfo;

(function (TypeDefInfo) {
  TypeDefInfo[TypeDefInfo["BTreeMap"] = 0] = "BTreeMap";
  TypeDefInfo[TypeDefInfo["BTreeSet"] = 1] = "BTreeSet";
  TypeDefInfo[TypeDefInfo["Compact"] = 2] = "Compact";
  TypeDefInfo[TypeDefInfo["DoNotConstruct"] = 3] = "DoNotConstruct";
  TypeDefInfo[TypeDefInfo["Enum"] = 4] = "Enum";
  TypeDefInfo[TypeDefInfo["HashMap"] = 5] = "HashMap";
  TypeDefInfo[TypeDefInfo["Int"] = 6] = "Int";
  TypeDefInfo[TypeDefInfo["Linkage"] = 7] = "Linkage";
  TypeDefInfo[TypeDefInfo["Null"] = 8] = "Null";
  TypeDefInfo[TypeDefInfo["Option"] = 9] = "Option";
  TypeDefInfo[TypeDefInfo["Plain"] = 10] = "Plain";
  TypeDefInfo[TypeDefInfo["Range"] = 11] = "Range";
  TypeDefInfo[TypeDefInfo["RangeInclusive"] = 12] = "RangeInclusive";
  TypeDefInfo[TypeDefInfo["Result"] = 13] = "Result";
  TypeDefInfo[TypeDefInfo["Set"] = 14] = "Set";
  TypeDefInfo[TypeDefInfo["Si"] = 15] = "Si";
  TypeDefInfo[TypeDefInfo["Struct"] = 16] = "Struct";
  TypeDefInfo[TypeDefInfo["Tuple"] = 17] = "Tuple";
  TypeDefInfo[TypeDefInfo["UInt"] = 18] = "UInt";
  TypeDefInfo[TypeDefInfo["Vec"] = 19] = "Vec";
  TypeDefInfo[TypeDefInfo["VecFixed"] = 20] = "VecFixed";
  TypeDefInfo[TypeDefInfo["WrapperKeepOpaque"] = 21] = "WrapperKeepOpaque";
  TypeDefInfo[TypeDefInfo["WrapperOpaque"] = 22] = "WrapperOpaque";
})(TypeDefInfo || (TypeDefInfo = {}));