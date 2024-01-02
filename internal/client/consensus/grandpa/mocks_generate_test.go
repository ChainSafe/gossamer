// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package grandpa

//go:generate mockery --name=Backend --filename=mocks_backend_test.go --output=./ --structname=BackendMock --inpackage --with-expecter=true
//go:generate mockery --name=BlockchainBackend --filename=mocks_blockchainbackend_test.go --output=./ --structname=BlockchainBackendMock --inpackage --with-expecter=true
//go:generate mockery --name=HeaderBackend --filename=mocks_headerbackend_test.go --output=./ --structname=HeaderBackendMock --inpackage --with-expecter=true
