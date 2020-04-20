package extrinsic

import (
	"encoding/binary"
	"errors"
	"io"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
)

const (
	AuthoritiesChangeType = 0
	TransferType          = 1
	IncludeDataType       = 2
	StorageChangeType     = 3
	//ChangesTrieConfigUpdateType = 4
)

type Extrinsic interface {
	Type() int
	Encode() ([]byte, error)
	Decode(r io.Reader) error
}

func DecodeExtrinsic(r io.Reader) (Extrinsic, error) {
	typ, err := common.ReadByte(r)
	if err != nil {
		return nil, err
	}

	switch typ {
	case AuthoritiesChangeType:
		ext := new(AuthoritiesChangeExt)
		return ext, ext.Decode(r)
	case TransferType:
		ext := new(TransferExt)
		return ext, ext.Decode(r)
	case IncludeDataType:
		ext := new(IncludeDataExt)
		return ext, ext.Decode(r)
	case StorageChangeType:
		ext := new(StorageChangeExt)
		return ext, ext.Decode(r)
	default:
		return nil, errors.New("cannot decode invalid extrinsic type")
	}
}

type AuthoritiesChangeExt struct {
	authorityIDs [][32]byte
}

func NewAuthoritiesChangeExt(authorityIDs [][32]byte) *AuthoritiesChangeExt {
	return &AuthoritiesChangeExt{
		authorityIDs: authorityIDs,
	}
}

func (e *AuthoritiesChangeExt) Type() int {
	return AuthoritiesChangeType
}

func (e *AuthoritiesChangeExt) Encode() ([]byte, error) {
	return nil, nil
}

func (e *AuthoritiesChangeExt) Decode(r io.Reader) error {
	return nil
}

type AccountID = uint64

type Transfer struct {
	from   AccountID
	to     AccountID
	amount uint64
	nonce  uint64
}

func NewTransfer(from, to AccountID, amount, nonce uint64) *Transfer {
	return &Transfer{
		from:   from,
		to:     to,
		amount: amount,
		nonce:  nonce,
	}
}

func (t *Transfer) Encode() ([]byte, error) {
	enc := []byte{}

	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, t.from)
	enc = append(enc, buf...)

	binary.LittleEndian.PutUint64(buf, t.to)
	enc = append(enc, buf...)

	binary.LittleEndian.PutUint64(buf, t.amount)
	enc = append(enc, buf...)

	binary.LittleEndian.PutUint64(buf, t.nonce)
	enc = append(enc, buf...)

	return enc, nil
}

func (t *Transfer) Decode(r io.Reader) (err error) {
	t.from, err = common.ReadUint64(r)
	if err != nil {
		return err
	}

	t.to, err = common.ReadUint64(r)
	if err != nil {
		return err
	}

	t.amount, err = common.ReadUint64(r)
	if err != nil {
		return err
	}

	t.nonce, err = common.ReadUint64(r)
	if err != nil {
		return err
	}

	return nil
}

type TransferExt struct {
	transfer                     *Transfer
	signature                    [sr25519.SignatureLength]byte
	exhaustResourcesWhenNotFirst bool
}

func NewTransferExt(transfer *Transfer, signature [sr25519.SignatureLength]byte, exhaustResourcesWhenNotFirst bool) *TransferExt {
	return &TransferExt{
		transfer:                     transfer,
		signature:                    signature,
		exhaustResourcesWhenNotFirst: exhaustResourcesWhenNotFirst,
	}
}

func (e *TransferExt) Type() int {
	return TransferType
}

func (e *TransferExt) Encode() ([]byte, error) {
	enc := []byte{TransferType}

	tenc, err := e.transfer.Encode()
	if err != nil {
		return nil, err
	}

	enc = append(enc, tenc...)
	enc = append(enc, e.signature[:]...)

	if e.exhaustResourcesWhenNotFirst {
		enc = append(enc, 1)
	} else {
		enc = append(enc, 0)
	}

	return enc, nil
}

func (e *TransferExt) Decode(r io.Reader) error {
	e.transfer = new(Transfer)
	err := e.transfer.Decode(r)
	if err != nil {
		return err
	}

	_, err = r.Read(e.signature[:])
	if err != nil {
		return err
	}

	b, err := common.ReadByte(r)
	if err != nil {
		return err
	}

	e.exhaustResourcesWhenNotFirst = b == 1
	return nil
}

type IncludeDataExt struct {
	data []byte
}

func NewIncludeDataExt(data []byte) *IncludeDataExt {
	return &IncludeDataExt{
		data: data,
	}
}

func (e *IncludeDataExt) Type() int {
	return IncludeDataType
}

func (e *IncludeDataExt) Encode() ([]byte, error) {
	return nil, nil
}

func (e *IncludeDataExt) Decode(r io.Reader) error {
	return nil
}

type StorageChangeExt struct {
	key   []byte
	value []byte
}

func NewStorageChangeExt(key, value []byte) *StorageChangeExt {
	return &StorageChangeExt{
		key:   key,
		value: value,
	}
}

func (e *StorageChangeExt) Type() int {
	return StorageChangeType
}

func (e *StorageChangeExt) Encode() ([]byte, error) {
	return nil, nil
}

func (e *StorageChangeExt) Decode(r io.Reader) error {
	return nil
}
