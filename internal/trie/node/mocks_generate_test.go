// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package node

//go:generate mockgen -destination=buffer_mock_test.go -package $GOPACKAGE . Buffer
//go:generate mockgen -destination=writer_mock_test.go -package $GOPACKAGE io Writer
//go:generate mockgen -destination=reader_mock_test.go -package $GOPACKAGE io Reader
