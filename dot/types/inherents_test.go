package types

import (
	"reflect"
	"testing"
)

func TestInherentsData_Encode(t *testing.T) {
	tests := []struct {
		name             string
		getInherentsData func(t *testing.T) *InherentsData
		want             []byte
		wantErr          bool
	}{
		{
			/*
				let mut data = InherentData::new();
				let timestamp: u64 = 99;
				data.put_data(*b"babeslot", &timestamp).unwrap();
				data.put_data(*b"timstap0", &timestamp).unwrap();
			*/
			getInherentsData: func(t *testing.T) *InherentsData {
				id := NewInherentsData()
				err := id.SetInherent(Babeslot, 99)
				if err != nil {
					t.Fatalf("%v2", err)
				}
				err = id.SetInherent(Timstap0, 99)
				if err != nil {
					t.Fatalf("%v", err)
				}
				return id
			},
			want: []byte{8, 98, 97, 98, 101, 115, 108, 111, 116, 32, 99, 0, 0, 0, 0, 0, 0, 0, 116, 105, 109, 115, 116, 97, 112, 48, 32, 99, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			/*
				let mut data = InherentData::new();
				data.put_data(TEST_INHERENT_0, &inherent_0).unwrap();
				data.put_data(TEST_INHERENT_1, &inherent_1).unwrap();
			*/
			getInherentsData: func(t *testing.T) *InherentsData {
				id := NewInherentsData()
				err := id.SetInherent([]byte("testinh0"), []int32{1, 2, 3})
				if err != nil {
					t.Fatalf("%v", err)
				}
				err = id.SetInherent([]byte("testinh1"), uint32(7))
				if err != nil {
					t.Fatalf("%v", err)
				}
				return id
			},
			want: []byte{8, 116, 101, 115, 116, 105, 110, 104, 48, 52, 12, 1, 0, 0, 0, 2, 0, 0, 0, 3, 0, 0, 0, 116, 101, 115, 116, 105, 110, 104, 49, 16, 7, 0, 0, 0},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idata := tt.getInherentsData(t)
			got, err := idata.Encode()
			if (err != nil) != tt.wantErr {
				t.Errorf("InherentsData.Encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("InherentsData.Encode() = %v, want %v", got, tt.want)
			}
		})
	}
}
