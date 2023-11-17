package grandpa

//go:generate mockery --name=Backend --filename=mocks_backend_test.go --output=./ --structname=BackendMock --inpackage --with-expecter=true
//go:generate mockery --name=BlockchainBackend --filename=mocks_blockchainBackend_test.go --output=./ --structname=BlockchainBackendMock --inpackage --with-expecter=true
//go:generate mockery --name=HeaderBackend --filename=mocks_headerBackend_test.go --output=./ --structname=HeaderBackendMock --inpackage --with-expecter=true
//go:generate mockery --name=Header --filename=mocks_Header_test.go --output=./ --structname=HeaderMock --inpackage --with-expecter=true
