package genesis

import (
	"reflect"
	"testing"
)

var tcNextKey = []struct {
	name      string
	jsonValue []byte
	goValue   nextKey
}{
	{
		name: "test1",
		jsonValue: []byte{
			91, 34, 53, 71, 78, 74, 113, 84, 80, 121, 78, 113, 65, 78, 66, 107, 85, 86, 77, 78, 49,
			76, 80, 80, 114, 120, 88, 110, 70, 111, 117, 87, 88, 111, 101, 50, 119, 78, 83, 109, 109,
			69, 111, 76, 99, 116, 120, 105, 90, 89, 34, 44, 34, 53, 71, 78, 74, 113, 84, 80, 121, 78,
			113, 65, 78, 66, 107, 85, 86, 77, 78, 49, 76, 80, 80, 114, 120, 88, 110, 70, 111, 117, 87,
			88, 111, 101, 50, 119, 78, 83, 109, 109, 69, 111, 76, 99, 116, 120, 105, 90, 89, 34, 44,
			123, 34, 103, 114, 97, 110, 100, 112, 97, 34, 58, 34, 53, 70, 65, 57, 110, 81, 68, 86, 103,
			50, 54, 55, 68, 69, 100, 56, 109, 49, 90, 121, 112, 88, 76, 66, 110, 118, 78, 55, 83, 70, 120,
			89, 119, 86, 55, 110, 100, 113, 83, 89, 71, 105, 78, 57, 84, 84, 112, 117, 34, 44, 34, 98, 97,
			98, 101, 34, 58, 34, 53, 71, 114, 119, 118, 97, 69, 70, 53, 122, 88, 98, 50, 54, 70, 122, 57,
			114, 99, 81, 112, 68, 87, 83, 53, 55, 67, 116, 69, 82, 72, 112, 78, 101, 104, 88, 67, 80, 99,
			78, 111, 72, 71, 75, 117, 116, 81, 89, 34, 44, 34, 105, 109, 95, 111, 110, 108, 105, 110, 101,
			34, 58, 34, 53, 71, 114, 119, 118, 97, 69, 70, 53, 122, 88, 98, 50, 54, 70, 122, 57, 114, 99,
			81, 112, 68, 87, 83, 53, 55, 67, 116, 69, 82, 72, 112, 78, 101, 104, 88, 67, 80, 99, 78, 111,
			72, 71, 75, 117, 116, 81, 89, 34, 44, 34, 97, 117, 116, 104, 111, 114, 105, 116, 121, 95, 100,
			105, 115, 99, 111, 118, 101, 114, 121, 34, 58, 34, 53, 71, 114, 119, 118, 97, 69, 70, 53, 122,
			88, 98, 50, 54, 70, 122, 57, 114, 99, 81, 112, 68, 87, 83, 53, 55, 67, 116, 69, 82, 72, 112, 78,
			101, 104, 88, 67, 80, 99, 78, 111, 72, 71, 75, 117, 116, 81, 89, 34, 125, 93,
		},
		goValue: nextKey{
			"5GNJqTPyNqANBkUVMN1LPPrxXnFouWXoe2wNSmmEoLctxiZY",
			"5GNJqTPyNqANBkUVMN1LPPrxXnFouWXoe2wNSmmEoLctxiZY",
			keyOwner{
				Grandpa:            "5FA9nQDVg267DEd8m1ZypXLBnvN7SFxYwV7ndqSYGiN9TTpu",
				Babe:               "5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY",
				ImOnline:           "5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY",
				AuthorityDiscovery: "5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY",
			},
		},
	},
}

func TestNextKeyMarshal(t *testing.T) {
	for _, tt := range tcNextKey {
		t.Run(tt.name, func(t *testing.T) {
			marshalledValue, err := tt.goValue.MarshalJSON()
			if err != nil {
				t.Fatalf("Couldn't marshal nextKey: %v", err)
			}
			if !reflect.DeepEqual(marshalledValue, tt.jsonValue) {
				t.Errorf("Unexpected marshal value : \nactual: %v \nExpected: %v", marshalledValue, tt.jsonValue)
			}
		})
	}
}

func TestNextKeyUnmarshal(t *testing.T) {
	for _, tt := range tcNextKey {
		t.Run(tt.name, func(t *testing.T) {
			var nk nextKey
			err := nk.UnmarshalJSON(tt.jsonValue)
			if err != nil {
				t.Fatalf("Couldn't unmarshal nextKey: %v", err)
			}
			if !reflect.DeepEqual(nk, tt.goValue) {
				t.Errorf("Unexpected unmarshal value : \nactual: %v \nExpected: %v", nk, tt.goValue)
			}
		})
	}
}

var tcMembersFields = []struct {
	name      string
	jsonValue []byte
	goValue   membersFields
}{
	{
		name: "test1",
		jsonValue: []byte{
			91, 34, 53, 71, 114, 119, 118, 97, 69, 70, 53, 122, 88, 98, 50, 54, 70, 122, 57, 114, 99,
			81, 112, 68, 87, 83, 53, 55, 67, 116, 69, 82, 72, 112, 78, 101, 104, 88, 67, 80, 99, 78,
			111, 72, 71, 75, 117, 116, 81, 89, 34, 44, 49, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48, 48,
			48, 48, 48, 48, 48, 48, 48, 93,
		},
		goValue: membersFields{"5GrwvaEF5zXb26Fz9rcQpDWS57CtERHpNehXCPcNoHGKutQY", 1000000000000000000},
	},
}

func TestMembersFieldsMarshal(t *testing.T) {
	for _, tt := range tcMembersFields {
		t.Run(tt.name, func(t *testing.T) {
			marshalledValue, err := tt.goValue.MarshalJSON()
			if err != nil {
				t.Fatalf("Couldn't marshal membersFields: %v", err)
			}
			if !reflect.DeepEqual(marshalledValue, tt.jsonValue) {
				t.Errorf("Unexpected marshal value : \nactual: %v \nExpected: %v", marshalledValue, tt.jsonValue)
			}
		})
	}
}

func TestMembersFieldsUnmarshal(t *testing.T) {
	for _, tt := range tcMembersFields {
		t.Run(tt.name, func(t *testing.T) {
			var mf membersFields
			err := mf.UnmarshalJSON(tt.jsonValue)
			if err != nil {
				t.Fatalf("Couldn't unmarshal membersFields: %v", err)
			}
			if !reflect.DeepEqual(mf, tt.goValue) {
				t.Errorf("Unexpected unmarshal value : \nactual: %v \nExpected: %v", mf, tt.goValue)
			}
		})
	}
}
