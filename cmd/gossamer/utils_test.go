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

package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli"
)

// newTestContext creates a cli context for a test given a set of flags and values
func newTestContext(description string, flags []string, values []interface{}) (*cli.Context, error) {
	if len(flags) != len(values) {
		return nil, fmt.Errorf("number of flags and values are not same, number of flags: %d, number of values: %d", len(flags), len(values))
	}

	// Define flags with its name and default value
	set := flag.NewFlagSet(description, 0)
	for i := range values {
		switch v := values[i].(type) {
		case bool:
			set.Bool(flags[i], v, "")
		case string:
			set.String(flags[i], v, "")
		case uint:
			set.Uint(flags[i], v, "")
		case int64:
			set.Int64(flags[i], v, "")
		case []string:
			set.Var(&cli.StringSlice{}, flags[i], "")
		default:
			return nil, fmt.Errorf("unexpected cli value type: %T", values[i])
		}
	}

	ctx := cli.NewContext(app, set, nil)

	for i := range values {
		switch v := values[i].(type) {
		case bool:
			err := ctx.Set(flags[i], strconv.FormatBool(v))
			if err != nil {
				return nil, fmt.Errorf("failed to set cli flag: %T, err: %w", flags[i], err)
			}
		case string:
			err := ctx.Set(flags[i], values[i].(string))
			if err != nil {
				return nil, fmt.Errorf("failed to set cli flag: %T, err: %w", flags[i], err)
			}
		case uint:
			err := ctx.Set(flags[i], strconv.Itoa(int(values[i].(uint))))
			if err != nil {
				return nil, fmt.Errorf("failed to set cli flag: %T, err: %w", flags[i], err)
			}
		case int64:
			err := ctx.Set(flags[i], strconv.Itoa(int(values[i].(int64))))
			if err != nil {
				return nil, fmt.Errorf("failed to set cli flag: %T, err: %w", flags[i], err)
			}
		case []string:
			for _, str := range values[i].([]string) {
				err := ctx.Set(flags[i], str)
				if err != nil {
					return nil, fmt.Errorf("failed to set cli flag: %T, err: %w", flags[i], err)
				}
			}
		default:
			return nil, fmt.Errorf("unexpected cli value type: %T", values[i])
		}
	}

	return ctx, nil
}

func Test_setupLogger(t *testing.T) {
	testApp := cli.NewApp()
	testApp.Writer = ioutil.Discard

	testcases := []struct {
		description string
		flags       []string
		values      []interface{}
		expected    error
	}{
		{
			"Test gossamer --log info",
			[]string{"log"},
			[]interface{}{"info"},
			nil,
		},
		{
			"Test gossamer --log debug",
			[]string{"log"},
			[]interface{}{"debug"},
			nil,
		},
		{
			"Test gossamer --log trace",
			[]string{"log"},
			[]interface{}{"trace"},
			nil,
		},
		{
			"Test gossamer --log blah",
			[]string{"log"},
			[]interface{}{"blah"},
			errors.New("level is not recognised: blah"),
		},
	}

	for _, c := range testcases {
		c := c // bypass scopelint false positive
		t.Run(c.description, func(t *testing.T) {
			ctx, err := newTestContext(c.description, c.flags, c.values)
			require.Nil(t, err)

			_, err = setupLogger(ctx)
			if c.expected != nil {
				assert.EqualError(t, err, c.expected.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
