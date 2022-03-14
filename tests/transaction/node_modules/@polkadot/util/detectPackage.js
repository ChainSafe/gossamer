// Copyright 2017-2022 @polkadot/util authors & contributors
// SPDX-License-Identifier: Apache-2.0
import others from "./detectOther.js";
import { packageInfo } from "./packageInfo.js";
import { detectPackage } from "./versionDetect.js";
detectPackage(packageInfo, null, others);