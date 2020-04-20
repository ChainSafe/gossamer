package extrinsic

import (
	"encoding/binary"
	"errors"
	"io"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/optional"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/scale"
)

const (
	AuthoritiesChangeType = 0
	TransferType          = 1
	IncludeDataType       = 2
	StorageChangeType     = 3
	// TODO: implement when storage changes trie is completed
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
	enc, err := scale.Encode(e.authorityIDs)
	if err != nil {
		return nil, err
	}

	return append([]byte{AuthoritiesChangeType}, enc...), nil
}

func (e *AuthoritiesChangeExt) Decode(r io.Reader) error {
	sd := &scale.Decoder{Reader: r}
	d, err := sd.Decode(e.authorityIDs)
	if err != nil {
		return err
	}

	e.authorityIDs = d.([][32]byte)
	return nil
}

type Transfer struct {
	from   [32]byte
	to     [32]byte
	amount uint64
	nonce  uint64
}

func NewTransfer(from, to [32]byte, amount, nonce uint64) *Transfer {
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

	enc = append(enc, t.from[:]...)
	enc = append(enc, t.to[:]...)

	binary.LittleEndian.PutUint64(buf, t.amount)
	enc = append(enc, buf...)

	binary.LittleEndian.PutUint64(buf, t.nonce)
	enc = append(enc, buf...)

	return enc, nil
}

func (t *Transfer) Decode(r io.Reader) (err error) {
	t.from, err = common.ReadHash(r)
	if err != nil {
		return err
	}

	t.to, err = common.ReadHash(r)
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
	enc, err := scale.Encode(e.data)
	if err != nil {
		return nil, err
	}

	return append([]byte{IncludeDataType}, enc...), nil
}

func (e *IncludeDataExt) Decode(r io.Reader) error {
	sd := &scale.Decoder{Reader: r}
	d, err := sd.Decode(e.data)
	if err != nil {
		return err
	}

	e.data = d.([]byte)
	return nil
}

type StorageChangeExt struct {
	key   []byte
	value *optional.Bytes
}

func NewStorageChangeExt(key []byte, value *optional.Bytes) *StorageChangeExt {
	return &StorageChangeExt{
		key:   key,
		value: value,
	}
}

func (e *StorageChangeExt) Type() int {
	return StorageChangeType
}

func (e *StorageChangeExt) Encode() ([]byte, error) {
	enc := []byte{StorageChangeType}

	d, err := scale.Encode(e.key)
	if err != nil {
		return nil, err
	}

	enc = append(enc, d...)

	if e.value.Exists() {
		enc = append(enc, 1)
		d, err = scale.Encode(e.value.Value())
		if err != nil {
			return nil, err
		}

		enc = append(enc, d...)
	} else {
		enc = append(enc, 0)
	}

	return enc, nil
}

func (e *StorageChangeExt) Decode(r io.Reader) error {
	sd := &scale.Decoder{Reader: r}
	d, err := sd.Decode([]byte{})
	if err != nil {
		return err
	}

	e.key = d.([]byte)

	exists, err := common.ReadByte(r)
	if err != nil {
		return err
	}

	if exists == 1 {
		d, err = sd.Decode([]byte{})
		if err != nil {
			return err
		}

		e.value = optional.NewBytes(true, d.([]byte))
	} else {
		e.value = optional.NewBytes(false, nil)
	}

	return nil
}
